package applications

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEsbuildPlatform(t *testing.T) {
	t.Parallel()

	cases := []struct {
		goos, goarch, want string
		ok                 bool
	}{
		{"linux", "amd64", "linux-x64", true},
		{"linux", "arm64", "linux-arm64", true},
		{"darwin", "amd64", "darwin-x64", true},
		{"darwin", "arm64", "darwin-arm64", true},
		{"windows", "amd64", "win32-x64", true},
		{"windows", "arm64", "win32-arm64", true},
		{"windows", "386", "win32-ia32", true},
		{"linux", "386", "linux-ia32", true},
		{"plan9", "amd64", "", false},
		{"linux", "sparc64", "", false},
	}
	for _, tc := range cases {
		got, ok := esbuildPlatform(tc.goos, tc.goarch)
		require.Equal(t, tc.ok, ok)
		require.Equal(t, tc.want, got)
	}
}

func TestEsbuildArtifactURL(t *testing.T) {
	t.Parallel()
	got := esbuildArtifactURL("linux-x64", "0.28.1")
	require.Equal(t, "https://registry.npmjs.org/@esbuild/linux-x64/-/linux-x64-0.28.1.tgz", got)
}

func TestNormalizeEsbuildVersion(t *testing.T) {
	t.Parallel()

	cases := []struct{ in, want string }{
		{"0.28.1", "0.28.1"},
		{"v0.28.1", "0.28.1"},
		{" latest ", "latest"},
		{"", ""},
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, normalizeEsbuildVersion(tc.in))
	}
}

func TestEsbuildListArtifacts(t *testing.T) {
	t.Parallel()

	installed := &esbuildTool{}
	artifacts, err := installed.ListArtifacts(t.Context(), "0.28.1")
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	require.Contains(t, artifacts[0].URL, "@esbuild/")
	require.Contains(t, artifacts[0].URL, "-0.28.1.tgz")
	if artifacts[0].Hash != "" {
		require.Contains(t, artifacts[0].Hash, "sha1:")
	}
}

func TestEsbuildListArtifactsAcceptsVPrefix(t *testing.T) {
	t.Parallel()

	installed := &esbuildTool{}
	artifacts, err := installed.ListArtifacts(t.Context(), "v0.28.1")
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	require.Contains(t, artifacts[0].URL, "-0.28.1.tgz")
}

func TestEsbuildEnrichLockfile(t *testing.T) {
	t.Parallel()

	pin := (&esbuildTool{}).Pin()
	require.Equal(t, "esbuild", pin.Name)
	require.Equal(t, "npm", pin.Datasource)
	require.Equal(t, "semver", pin.Versioning)
}

func TestEsbuildListVersions(t *testing.T) {
	t.Parallel()

	installed := &esbuildTool{}
	versions, err := installed.ListVersions(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, versions)
	for _, version := range versions {
		require.NotContains(t, version, "-")
	}
}
