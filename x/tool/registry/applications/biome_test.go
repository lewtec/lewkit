package applications

import (
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/tool"

	"github.com/stretchr/testify/require"
)

func TestBiomeListVersionsFiltersMonorepoTags(t *testing.T) {
	t.Parallel()

	installed := &biomeTool{inner: stubTool{versions: []string{
		"@biomejs/js-api@6.0.0",
		"@biomejs/biome@2.5.0",
		"@biomejs/js-api@5.9.0",
		"@biomejs/biome@2.4.16",
		"@biomejs/biome@2.4.16-beta.1",
		"v1.9.4",
	}}}

	got, err := installed.ListVersions(t.Context())
	require.NoError(t, err)
	require.Equal(t, []string{"2.5.0", "2.4.16"}, got)
}

func TestBiomeVersionFromTag(t *testing.T) {
	t.Parallel()

	cases := []struct {
		tag string
		ver string
		ok  bool
	}{
		{"@biomejs/biome@2.5.0", "2.5.0", true},
		{"@biomejs/js-api@6.0.0", "", false},
		{"@biomejs/biome@2.5.0-rc.1", "", false},
		{"2.5.0", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		version, ok := biomeVersionFromTag(tc.tag)
		require.Equal(t, tc.ok, ok)
		require.Equal(t, tc.ver, version)
	}
}

func TestBiomeTagForVersion(t *testing.T) {
	t.Parallel()
	require.Equal(t, "@biomejs/biome@2.5.0", biomeTagForVersion("2.5.0"))
	require.Equal(t, "@biomejs/biome@2.5.0", biomeTagForVersion("@biomejs/biome@2.5.0"))
}

func TestParseBiomeAssetURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		url  string
		os   string
		arch string
		ok   bool
	}{
		{"https://example/biome-darwin-arm64", "darwin", "arm64", true},
		{"https://example/biome-linux-x64", "linux", "amd64", true},
		{"https://example/biome-linux-x64-musl", "linux", "amd64", true},
		{"https://example/biome-win32-x64.exe", "windows", "amd64", true},
		{"https://example/biome-win32-arm64.exe", "windows", "arm64", true},
		{"https://example/checksums.txt", "", "", false},
	}
	for _, tc := range cases {
		osName, arch, ok := parseBiomeAssetURL(tc.url)
		require.Equal(t, tc.ok, ok)
		require.Equal(t, tc.os, osName)
		require.Equal(t, tc.arch, arch)
	}
}

func TestSelectBiomeArtifactPrefersNonMusl(t *testing.T) {
	t.Parallel()

	artifacts := []tool.Artifact{
		{OS: "linux", Arch: "amd64", URL: "https://example/biome-linux-x64-musl"},
		{OS: "linux", Arch: "amd64", URL: "https://example/biome-linux-x64"},
		{OS: "darwin", Arch: "arm64", URL: "https://example/biome-darwin-arm64"},
	}
	got := selectBiomeArtifact(artifacts, "linux", "amd64")
	require.NotNil(t, got)
	require.True(t, strings.HasSuffix(got.URL, "biome-linux-x64"))
}

func TestBiomeEnrichLockfileExtractVersion(t *testing.T) {
	t.Parallel()

	installed := &biomeTool{inner: stubTool{}}
	pin := installed.Pin()
	require.Equal(t, "semver", pin.Versioning)
	require.NotEmpty(t, pin.ExtractVersion)
}
