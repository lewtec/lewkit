package pkg

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/lewtec/lewkit/x/driver/httpclient/native"
	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry"
	"github.com/lewtec/lewkit/x/tool/registry/internal/install"

	"github.com/stretchr/testify/require"
)

func TestHeliumRegistered(t *testing.T) {
	t.Parallel()

	installed, err := registry.NewTool("helium-browser")
	require.NoError(t, err)
	checker, ok := installed.(tool.Checker)
	require.True(t, ok)
	require.Equal(t, "binary:helium", checker.InstallChecks()[0].Name())
	require.Equal(t, "semver", installed.(tool.Pinner).Pin().Versioning)
}

func TestHeliumLatestFollowsHostRepository(t *testing.T) {
	t.Parallel()

	installed := heliumFixture()
	installed.goos = "darwin"
	versions, err := installed.ListVersions(t.Context())
	require.NoError(t, err)
	require.Equal(t, []string{"0.19.2.1"}, versions)

	installed.goos = "linux"
	versions, err = installed.ListVersions(t.Context())
	require.NoError(t, err)
	require.Equal(t, "0.19.3.1", versions[0])
}

func TestHeliumSelectsMacOSDiskImage(t *testing.T) {
	t.Parallel()

	installed := heliumFixture()
	installed.goos = "darwin"
	artifacts, err := installed.ListArtifacts(t.Context(), "latest")
	require.NoError(t, err)

	arm64 := tool.SelectArtifact(artifacts, "darwin", "arm64", "helium")
	require.NotNil(t, arm64)
	require.Equal(t, "https://github.com/imputnet/helium-macos/releases/download/0.19.2.1/helium_0.19.2.1_arm64-macos.dmg", arm64.URL)
	require.Equal(t, "sha256:c07dab9c1571cf68cf856b35c6d0bf21c2cf53f77c43245163018e788094ebf1", arm64.Hash)

	amd64 := tool.SelectArtifact(artifacts, "darwin", "amd64", "helium")
	require.NotNil(t, amd64)
	require.Equal(t, "https://github.com/imputnet/helium-macos/releases/download/0.19.2.1/helium_0.19.2.1_x86_64-macos.dmg", amd64.URL)
	require.Equal(t, "sha256:a032da1664b0b5aad30b3d02ce31e194ef5333d3b148f33a8182404c70a7e109", amd64.Hash)

	installed.goos = "linux"
	artifacts, err = installed.ListArtifacts(t.Context(), "latest")
	require.NoError(t, err)
	linux := tool.SelectArtifact(artifacts, "linux", "amd64", "helium")
	require.NotNil(t, linux)
	require.Equal(t, "https://github.com/imputnet/helium-linux/releases/download/0.19.3.1/helium-0.19.3.1-x86_64_linux.tar.xz", linux.URL)
}

func TestHeliumDMGIsNotCopiedAsTheBinary(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte("not-an-archive-dmg"))
	}))
	t.Cleanup(server.Close)

	raw, err := newHeliumBrowser()
	require.NoError(t, err)
	dest := t.TempDir()
	err = raw.(tool.ArtifactTool).InstallArtifact(t.Context(), tool.Artifact{
		OS:   "darwin",
		Arch: "arm64",
		URL:  server.URL + "/helium_0.19.2.1_arm64-macos.dmg",
	}, dest)
	require.ErrorIs(t, err, tool.ErrDiskImageMount)
	require.ErrorIs(t, err, errs.ErrUnavailable)
	require.Empty(t, tool.FindBinary(dest, "helium"))
	_, statErr := os.Stat(filepath.Join(dest, "helium_0.19.2.1_arm64-macos.dmg"))
	require.Error(t, statErr)
}

func TestHeliumAppBundleBinary(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	executable := filepath.Join(dest, "Helium.app", "Contents", "MacOS", "Helium")
	require.NoError(t, os.MkdirAll(filepath.Dir(executable), 0o755))
	require.NoError(t, os.WriteFile(executable, []byte("macho"), 0o755))

	raw, err := newHeliumBrowser()
	require.NoError(t, err)
	require.Equal(t, executable, tool.FindBinary(dest, "helium"))
	require.NoError(t, raw.(tool.Checker).InstallChecks()[0].Check(t.Context(), dest))
}

func TestHeliumCachedDiskImageFailsCheck(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	body := make([]byte, 512)
	copy(body, []byte("koly"))
	require.NoError(t, os.WriteFile(filepath.Join(dest, "helium"), body, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dest, "helium_0.19.2.1_arm64-macos.dmg"), body, 0o755))

	raw, err := newHeliumBrowser()
	require.NoError(t, err)
	err = raw.(tool.Checker).InstallChecks()[0].Check(t.Context(), dest)
	require.ErrorIs(t, err, tool.ErrBinaryNotFound)
	require.Empty(t, tool.FindBinary(dest, "helium"))
}

func TestHeliumListArtifactsEmpty(t *testing.T) {
	t.Parallel()

	installed := &heliumBrowserTool{
		goos: "darwin",
		sources: []heliumSource{{
			goos: "darwin",
			tool: heliumStub{versions: []string{"0.19.2.1"}},
		}},
	}
	_, err := installed.ListArtifacts(t.Context(), "0.19.2.1")
	require.ErrorIs(t, err, install.ErrNoPlatformArtifact)
	require.ErrorIs(t, err, errs.ErrNotFound)
}

func heliumFixture() *heliumBrowserTool {
	return &heliumBrowserTool{
		sources: []heliumSource{
			{
				goos: "linux",
				tool: heliumStub{
					versions: []string{"0.19.3.1", "0.19.2.1"},
					artifacts: []tool.Artifact{
						{OS: "linux", Arch: "amd64", URL: "https://github.com/imputnet/helium-linux/releases/download/0.19.3.1/helium-0.19.3.1-x86_64.AppImage"},
						{OS: "linux", Arch: "amd64", URL: "https://github.com/imputnet/helium-linux/releases/download/0.19.3.1/helium-0.19.3.1-x86_64_linux.tar.xz.asc"},
						{OS: "linux", Arch: "amd64", URL: "https://github.com/imputnet/helium-linux/releases/download/0.19.3.1/helium-0.19.3.1-x86_64_linux.tar.xz"},
					},
				},
			},
			{
				goos: "darwin",
				tool: heliumStub{
					versions: []string{"0.19.2.1"},
					artifacts: []tool.Artifact{
						{
							OS:   "darwin",
							Arch: "arm64",
							URL:  "https://github.com/imputnet/helium-macos/releases/download/0.19.2.1/helium_0.19.2.1_arm64-macos.dmg",
							Hash: "sha256:c07dab9c1571cf68cf856b35c6d0bf21c2cf53f77c43245163018e788094ebf1",
						},
						{
							OS:   "darwin",
							Arch: "amd64",
							URL:  "https://github.com/imputnet/helium-macos/releases/download/0.19.2.1/helium_0.19.2.1_x86_64-macos.dmg",
							Hash: "sha256:a032da1664b0b5aad30b3d02ce31e194ef5333d3b148f33a8182404c70a7e109",
						},
					},
				},
			},
			{
				goos: "windows",
				tool: heliumStub{
					versions: []string{"0.19.2.1"},
					artifacts: []tool.Artifact{
						{OS: "windows", Arch: "amd64", URL: "https://github.com/imputnet/helium-windows/releases/download/0.19.2.1/helium_0.19.2.1_x64-windows.zip"},
					},
				},
			},
		},
	}
}

type heliumStub struct {
	versions  []string
	artifacts []tool.Artifact
}

func (stub heliumStub) ListVersions(context.Context) ([]string, error) {
	return stub.versions, nil
}

func (heliumStub) Install(context.Context, string, string) error { return nil }

func (stub heliumStub) ListArtifacts(context.Context, string) ([]tool.Artifact, error) {
	if len(stub.artifacts) == 0 {
		return nil, install.ErrNoPlatformArtifact
	}
	return stub.artifacts, nil
}

func (heliumStub) InstallArtifact(context.Context, tool.Artifact, string) error { return nil }
