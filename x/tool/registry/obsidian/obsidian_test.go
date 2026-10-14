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

func TestObsidianRegistered(t *testing.T) {
	t.Parallel()

	installed, err := registry.NewTool("obsidian")
	require.NoError(t, err)
	checker, ok := installed.(tool.Checker)
	require.True(t, ok)
	require.Equal(t, []string{"binary:obsidian"}, checkNames(checker))

	pin := installed.(tool.Pinner).Pin()
	require.Equal(t, "semver", pin.Versioning)
	require.Equal(t, "github-releases", pin.Datasource)
	require.Equal(t, obsidianRepo, pin.Name)
}

func TestObsidianListVersionsFiltersTags(t *testing.T) {
	t.Parallel()

	installed := &obsidianTool{inner: stubTool{versions: []string{
		"v1.9.0",
		"v1.14.4",
		"v1.14.4-beta.1",
		"not-a-release",
		"v1.14.4",
	}}}
	got, err := installed.ListVersions(t.Context())
	require.NoError(t, err)
	require.Equal(t, []string{"1.14.4", "1.9.0"}, got)
}

func TestObsidianVersion(t *testing.T) {
	t.Parallel()

	cases := []struct {
		tag     string
		version string
		ok      bool
	}{
		{"v1.14.4", "1.14.4", true},
		{"1.14.4", "1.14.4", true},
		{"V1.2.0", "1.2.0", true},
		{"v1.14.4-beta.1", "", false},
		{"latest", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		version, ok := obsidianVersion(tc.tag)
		require.Equal(t, tc.ok, ok)
		require.Equal(t, tc.version, version)
	}
}

func TestObsidianPlatforms(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		want []obsidianPlatform
	}{
		{"obsidian-1.14.4.tar.gz", []obsidianPlatform{{osName: "linux", arch: "amd64"}}},
		{"obsidian-1.14.4-arm64.tar.gz", []obsidianPlatform{{osName: "linux", arch: "arm64"}}},
		{"Obsidian-1.14.4.AppImage", []obsidianPlatform{{osName: "linux", arch: "amd64"}}},
		{"Obsidian-1.14.4-arm64.AppImage", []obsidianPlatform{{osName: "linux", arch: "arm64"}}},
		{"Obsidian-1.14.4.dmg", []obsidianPlatform{{osName: "darwin", arch: "amd64"}, {osName: "darwin", arch: "arm64"}}},
		{"Obsidian-1.14.4-arm64.dmg", []obsidianPlatform{{osName: "darwin", arch: "arm64"}}},
		{"Obsidian-1.14.4.exe", []obsidianPlatform{{osName: "windows", arch: "amd64"}}},
		{"obsidian_1.14.4_amd64.deb", nil},
		{"obsidian-1.14.4.asar.gz", nil},
		{"Obsidian-1.14.4.apk", nil},
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, obsidianPlatforms(tc.name), tc.name)
	}
}

func TestObsidianSelectsPortableArchive(t *testing.T) {
	t.Parallel()

	artifacts := obsidianArtifacts(releaseAssets())

	linuxAMD64 := tool.SelectArtifact(artifacts, "linux", "amd64", "obsidian")
	require.NotNil(t, linuxAMD64)
	require.Equal(t, "https://example.test/obsidian-1.14.4.tar.gz", linuxAMD64.URL)
	require.Equal(t, "sha256:aabb", linuxAMD64.Hash)

	linuxARM64 := tool.SelectArtifact(artifacts, "linux", "arm64", "obsidian")
	require.NotNil(t, linuxARM64)
	require.Equal(t, "https://example.test/obsidian-1.14.4-arm64.tar.gz", linuxARM64.URL)

	for _, arch := range []string{"amd64", "arm64"} {
		darwin := tool.SelectArtifact(artifacts, "darwin", arch, "obsidian")
		require.NotNil(t, darwin)
		require.Equal(t, "https://example.test/Obsidian-1.14.4.dmg", darwin.URL)
	}

	windows := tool.SelectArtifact(artifacts, "windows", "amd64", "obsidian")
	require.NotNil(t, windows)
	require.Equal(t, "https://example.test/Obsidian-1.14.4.exe", windows.URL)
	require.Equal(t, int64(42), windows.Size)
	require.Equal(t, int64(7), windows.GitHubAssetID)

	require.Nil(t, tool.SelectArtifact(artifacts, "windows", "arm64", "obsidian"))
}

func TestObsidianListArtifacts(t *testing.T) {
	t.Parallel()

	const releaseURL = "https://api.github.com/repos/obsidianmd/obsidian-releases/releases/tags/v1.14.4"
	installed := &obsidianTool{
		fetchURL: func(_ context.Context, rawURL string) ([]byte, error) {
			require.Equal(t, releaseURL, rawURL)
			return []byte(`{"assets":[
				{"id":7,"name":"obsidian-1.14.4.tar.gz","size":42,"digest":"sha256:aabb","browser_download_url":"https://example.test/obsidian-1.14.4.tar.gz","url":"https://api.github.com/asset/7"},
				{"name":"Obsidian-1.14.4.AppImage","browser_download_url":"https://example.test/Obsidian-1.14.4.AppImage"},
				{"name":"Obsidian-1.14.4.dmg","browser_download_url":"https://example.test/Obsidian-1.14.4.dmg"},
				{"name":"Obsidian-1.14.4.exe","browser_download_url":"https://example.test/Obsidian-1.14.4.exe"}
			]}`), nil
		},
	}

	artifacts, err := installed.ListArtifacts(t.Context(), "v1.14.4")
	require.NoError(t, err)
	selected := tool.SelectArtifact(artifacts, "linux", "amd64", "obsidian")
	require.NotNil(t, selected)
	require.Equal(t, "https://example.test/obsidian-1.14.4.tar.gz", selected.URL)
	require.Equal(t, "sha256:aabb", selected.Hash)
	require.Equal(t, "https://api.github.com/asset/7", selected.GitHubAssetAPIURL)
}

func TestObsidianListArtifactsLatest(t *testing.T) {
	t.Parallel()

	installed := &obsidianTool{
		inner: stubTool{versions: []string{"v1.2.0", "v1.3.0"}},
		fetchURL: func(_ context.Context, rawURL string) ([]byte, error) {
			require.Equal(t, "https://api.github.com/repos/obsidianmd/obsidian-releases/releases/tags/v1.3.0", rawURL)
			return []byte(`{"assets":[{"name":"obsidian-1.3.0.tar.gz","browser_download_url":"https://example.test/obsidian-1.3.0.tar.gz"}]}`), nil
		},
	}
	artifacts, err := installed.ListArtifacts(t.Context(), "latest")
	require.NoError(t, err)
	require.Equal(t, "https://example.test/obsidian-1.3.0.tar.gz", artifacts[0].URL)
}

func TestObsidianListArtifactsEmpty(t *testing.T) {
	t.Parallel()

	installed := &obsidianTool{
		fetchURL: func(context.Context, string) ([]byte, error) {
			return []byte(`{"assets":[{"name":"Obsidian-1.14.4.apk","browser_download_url":"https://example.test/app.apk"}]}`), nil
		},
	}
	_, err := installed.ListArtifacts(t.Context(), "1.14.4")
	require.ErrorIs(t, err, ErrNoPlatformArtifact)
	require.ErrorIs(t, err, install.ErrNoPlatformArtifact)
	require.ErrorIs(t, err, errs.ErrNotFound)
}

func TestObsidianDecodeRelease(t *testing.T) {
	t.Parallel()

	installed := &obsidianTool{
		fetchURL: func(context.Context, string) ([]byte, error) {
			return []byte(`{`), nil
		},
	}
	_, err := installed.ListArtifacts(t.Context(), "1.14.4")
	require.ErrorIs(t, err, ErrDecodeRelease)
	require.ErrorIs(t, err, errs.ErrInvalid)
}

func TestObsidianDMGIsNotCopiedAsTheBinary(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte("not-an-archive-dmg"))
	}))
	t.Cleanup(server.Close)

	raw, err := newObsidian()
	require.NoError(t, err)
	dest := t.TempDir()
	err = raw.(tool.ArtifactTool).InstallArtifact(t.Context(), tool.Artifact{
		OS:   "darwin",
		Arch: "amd64",
		URL:  server.URL + "/Obsidian-1.14.4.dmg",
	}, dest)
	require.ErrorIs(t, err, tool.ErrDiskImageMount)
	require.Empty(t, tool.FindBinary(dest, "obsidian"))
	_, statErr := os.Stat(filepath.Join(dest, "Obsidian"))
	require.Error(t, statErr)
}

func releaseAssets() []obsidianAsset {
	return []obsidianAsset{
		{Name: "obsidian-1.14.4.tar.gz", BrowserDownloadURL: "https://example.test/obsidian-1.14.4.tar.gz", Digest: "sha256:aabb", Size: 11},
		{Name: "Obsidian-1.14.4.AppImage", BrowserDownloadURL: "https://example.test/Obsidian-1.14.4.AppImage"},
		{Name: "obsidian-1.14.4-arm64.tar.gz", BrowserDownloadURL: "https://example.test/obsidian-1.14.4-arm64.tar.gz"},
		{Name: "Obsidian-1.14.4-arm64.AppImage", BrowserDownloadURL: "https://example.test/Obsidian-1.14.4-arm64.AppImage"},
		{Name: "Obsidian-1.14.4.dmg", BrowserDownloadURL: "https://example.test/Obsidian-1.14.4.dmg", Digest: "ccdd"},
		{Name: "Obsidian-1.14.4.exe", BrowserDownloadURL: "https://example.test/Obsidian-1.14.4.exe", Size: 42, ID: 7, APIURL: "https://api.github.com/asset/7"},
		{Name: "obsidian_1.14.4_amd64.deb", BrowserDownloadURL: "https://example.test/obsidian.deb"},
		{Name: "obsidian-1.14.4.asar.gz", BrowserDownloadURL: "https://example.test/obsidian.asar.gz"},
		{Name: "Obsidian-1.14.4.apk", BrowserDownloadURL: "https://example.test/obsidian.apk"},
	}
}

func checkNames(checker tool.Checker) []string {
	var names []string
	for _, check := range checker.InstallChecks() {
		names = append(names, check.Name())
	}
	return names
}

type stubTool struct {
	versions []string
}

func (t stubTool) ListVersions(context.Context) ([]string, error) {
	return append([]string(nil), t.versions...), nil
}

func (t stubTool) Install(context.Context, string, string) error { return nil }

func (t stubTool) Pin() tool.Pin {
	return tool.Pin{Name: obsidianRepo, Datasource: "github-releases"}
}
