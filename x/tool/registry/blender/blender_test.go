package pkg

import (
	"context"
	"errors"
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

func TestBlenderRegistered(t *testing.T) {
	t.Parallel()

	installed, err := registry.NewTool("blender")
	require.NoError(t, err)
	checker, ok := installed.(tool.Checker)
	require.True(t, ok)
	require.Equal(t, "binary:blender", checker.InstallChecks()[0].Name())
	require.Equal(t, "semver", installed.(tool.Pinner).Pin().Versioning)
}

func TestParseBlenderName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name, version, goos, goarch string
		ok                          bool
	}{
		{"blender-5.2.2-linux-x64.tar.xz", "5.2.2", "linux", "amd64", true},
		{"blender-4.2.0-linux-arm64.tar.xz", "4.2.0", "linux", "arm64", true},
		{"blender-2.83.0-linux64.tar.xz", "2.83.0", "linux", "amd64", true},
		{"blender-5.2.2-macos-arm64.dmg", "5.2.2", "darwin", "arm64", true},
		{"blender-4.2.0-macos-x64.dmg", "4.2.0", "darwin", "amd64", true},
		{"blender-2.83.0-macOS.dmg", "2.83.0", "darwin", "amd64", true},
		{"blender-5.2.2-windows-x64.zip", "5.2.2", "windows", "amd64", true},
		{"blender-5.2.2-windows-x64.msi", "5.2.2", "windows", "amd64", true},
		{"blender-5.2.2-windows-arm64.zip", "5.2.2", "windows", "arm64", true},
		{"blender-2.83.0-windows64.zip", "2.83.0", "windows", "amd64", true},
		{"blender-5.2.2.tar.xz", "", "", "", false},
		{"blender-5.2.2.sha256", "", "", "", false},
		{"blender-5.2.2.md5", "", "", "", false},
		{"GPL-license.txt", "", "", "", false},
	}
	for _, tc := range cases {
		version, goos, goarch, _, ok := parseBlenderName(tc.name)
		require.Equal(t, tc.ok, ok, tc.name)
		require.Equal(t, tc.version, version, tc.name)
		require.Equal(t, tc.goos, goos, tc.name)
		require.Equal(t, tc.goarch, goarch, tc.name)
	}
}

func TestBlenderBuildsPreferPortableArchive(t *testing.T) {
	t.Parallel()

	names := []string{
		"blender-5.2.2-windows-x64.msi",
		"blender-5.2.2-windows-x64.msix",
		"blender-5.2.2-windows-x64.zip",
		"blender-5.2.2-windows-arm64.msi",
		"blender-5.2.2-windows-arm64.zip",
		"blender-5.2.2.sha256",
	}
	windowsAMD64 := blenderBuilds("https://download.blender.org/release/Blender5.2/", names, "windows", "amd64")
	require.Len(t, windowsAMD64, 1)
	require.Equal(t, "blender-5.2.2-windows-x64.zip", windowsAMD64[0].Name)

	windowsARM64 := blenderBuilds("https://download.blender.org/release/Blender5.2/", names, "windows", "arm64")
	require.Len(t, windowsARM64, 1)
	require.Equal(t, "blender-5.2.2-windows-arm64.zip", windowsARM64[0].Name)
}

func TestBlenderListVersionsByPlatform(t *testing.T) {
	t.Parallel()

	cases := []struct {
		goos, goarch string
		want         []string
	}{
		{"linux", "amd64", []string{"5.2.2", "5.2.0", "4.2.0"}},
		{"linux", "arm64", nil},
		{"darwin", "arm64", []string{"5.2.2", "5.2.0", "4.2.0"}},
		{"darwin", "amd64", []string{"4.2.0"}},
		{"windows", "amd64", []string{"5.2.2", "4.2.0"}},
		{"windows", "arm64", []string{"5.2.2"}},
	}
	for _, tc := range cases {
		t.Run(tc.goos+"/"+tc.goarch, func(t *testing.T) {
			t.Parallel()
			installed := fixtureBlender(t, tc.goos, tc.goarch)
			got, err := installed.ListVersions(t.Context())
			if tc.want == nil {
				require.ErrorIs(t, err, ErrNoVersions)
				require.ErrorIs(t, err, install.ErrNoVersions)
				require.ErrorIs(t, err, errs.ErrNotFound)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestBlenderListArtifacts(t *testing.T) {
	t.Parallel()

	installed := fixtureBlender(t, "linux", "amd64")
	artifacts, err := installed.ListArtifacts(t.Context(), "v5.2.2")
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	require.Equal(t, "https://download.blender.org/release/Blender5.2/blender-5.2.2-linux-x64.tar.xz", artifacts[0].URL)
	require.Equal(t, "sha256:"+repeatHex('a'), artifacts[0].Hash)
	require.Equal(t, "linux", artifacts[0].OS)
	require.Equal(t, "amd64", artifacts[0].Arch)

	latest, err := installed.ListArtifacts(t.Context(), "latest")
	require.NoError(t, err)
	require.Equal(t, artifacts[0].URL, latest[0].URL)
}

func TestBlenderListArtifactsMissingHash(t *testing.T) {
	t.Parallel()

	installed := fixtureBlender(t, "darwin", "amd64")
	artifacts, err := installed.ListArtifacts(t.Context(), "4.2.0")
	require.NoError(t, err)
	require.Equal(t, "https://download.blender.org/release/Blender4.2/blender-4.2.0-macos-x64.dmg", artifacts[0].URL)
	require.Empty(t, artifacts[0].Hash)
}

func TestBlenderListArtifactsMissingPlatform(t *testing.T) {
	t.Parallel()

	installed := fixtureBlender(t, "linux", "arm64")
	_, err := installed.ListArtifacts(t.Context(), "5.2.2")
	require.ErrorIs(t, err, ErrNoPlatformArtifact)
	require.ErrorIs(t, err, errs.ErrNotFound)
}

func TestBlenderUnsupportedPlatform(t *testing.T) {
	t.Parallel()

	installed := &blenderTool{goos: "plan9", goarch: "amd64"}
	_, err := installed.ListVersions(t.Context())
	require.ErrorIs(t, err, ErrNoPlatformArtifact)
	_, err = installed.ListArtifacts(t.Context(), "5.2.2")
	require.ErrorIs(t, err, ErrNoPlatformArtifact)
}

func TestBlenderDMGIsNotCopiedAsTheBinary(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte("not-an-archive-dmg"))
	}))
	t.Cleanup(server.Close)

	raw, err := newBlender()
	require.NoError(t, err)
	dest := t.TempDir()
	err = raw.(tool.ArtifactTool).InstallArtifact(t.Context(), tool.Artifact{
		OS:   "darwin",
		Arch: "arm64",
		URL:  server.URL + "/blender-5.2.2-macos-arm64.dmg",
	}, dest)
	require.ErrorIs(t, err, tool.ErrDiskImageMount)
	require.Empty(t, tool.FindBinary(dest, "blender"))
	_, statErr := os.Stat(filepath.Join(dest, "blender-5.2.2-macos-arm64.dmg"))
	require.Error(t, statErr)
}

func TestBlenderSeriesNamesIgnoreNonReleases(t *testing.T) {
	t.Parallel()

	names := blenderSeriesNames([]byte(releaseIndexHTML))
	require.Equal(t, []string{"Blender5.2", "Blender4.2"}, names)
}

func fixtureBlender(t *testing.T, goos, goarch string) *blenderTool {
	t.Helper()
	return &blenderTool{
		baseURL: "https://download.blender.org/release/",
		goos:    goos,
		goarch:  goarch,
		fetchURL: func(_ context.Context, rawURL string) ([]byte, error) {
			switch rawURL {
			case "https://download.blender.org/release/":
				return []byte(releaseIndexHTML), nil
			case "https://download.blender.org/release/Blender5.2/":
				return []byte(series52HTML), nil
			case "https://download.blender.org/release/Blender4.2/":
				return []byte(series42HTML), nil
			case "https://download.blender.org/release/Blender5.2/blender-5.2.2.sha256":
				return []byte(repeatHex('a') + "  blender-5.2.2-linux-x64.tar.xz\n"), nil
			case "https://download.blender.org/release/Blender4.2/blender-4.2.0.sha256":
				return nil, errors.New("missing")
			default:
				return nil, errors.New("unexpected url " + rawURL)
			}
		},
	}
}

func repeatHex(digit byte) string {
	out := make([]byte, 64)
	for i := range out {
		out[i] = digit
	}
	return string(out)
}

const releaseIndexHTML = `
<a href="../">../</a>
<a href="Blender5.2/">Blender5.2/</a>
<a href="Blender4.2/">Blender4.2/</a>
<a href="BlenderBenchmark2.0/">BlenderBenchmark2.0/</a>
<a href="Blender2.50alpha/">Blender2.50alpha/</a>
`

const series52HTML = `
<a href="../">../</a>
<a href="blender-5.2.0-linux-x64.tar.xz">blender-5.2.0-linux-x64.tar.xz</a>
<a href="blender-5.2.0-macos-arm64.dmg">blender-5.2.0-macos-arm64.dmg</a>
<a href="blender-5.2.2-linux-x64.tar.xz">blender-5.2.2-linux-x64.tar.xz</a>
<a href="blender-5.2.2-macos-arm64.dmg">blender-5.2.2-macos-arm64.dmg</a>
<a href="blender-5.2.2-windows-x64.msi">blender-5.2.2-windows-x64.msi</a>
<a href="blender-5.2.2-windows-x64.zip">blender-5.2.2-windows-x64.zip</a>
<a href="blender-5.2.2-windows-arm64.zip">blender-5.2.2-windows-arm64.zip</a>
<a href="blender-5.2.2.sha256">blender-5.2.2.sha256</a>
<a href="blender-5.2.2.tar.xz">blender-5.2.2.tar.xz</a>
`

const series42HTML = `
<a href="../">../</a>
<a href="blender-4.2.0-linux-x64.tar.xz">blender-4.2.0-linux-x64.tar.xz</a>
<a href="blender-4.2.0-macos-arm64.dmg">blender-4.2.0-macos-arm64.dmg</a>
<a href="blender-4.2.0-macos-x64.dmg">blender-4.2.0-macos-x64.dmg</a>
<a href="blender-4.2.0-windows-x64.zip">blender-4.2.0-windows-x64.zip</a>
`
