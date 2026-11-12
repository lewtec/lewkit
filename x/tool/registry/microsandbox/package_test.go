package pkg

import (
	"testing"

	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry"

	"github.com/stretchr/testify/require"
)

func TestMicrosandboxVersionFromTag(t *testing.T) {
	t.Parallel()

	cases := []struct {
		tag     string
		version string
		ok      bool
	}{
		{"v0.7.14", "0.7.14", true},
		{"0.7.14", "", false},
		{"v0.8.0-beta.1", "", false},
		{"v", "", false},
		{"", "", false},
	}
	for _, testCase := range cases {
		version, ok := microsandboxVersionFromTag(testCase.tag)
		require.Equal(t, testCase.ok, ok)
		require.Equal(t, testCase.version, version)
	}
}

func TestSelectMicrosandboxArtifactPrefersBundle(t *testing.T) {
	t.Parallel()

	artifacts := []tool.Artifact{
		{OS: "linux", Arch: "amd64", URL: "https://example/msb-linux-x86_64"},
		{OS: "linux", Arch: "amd64", URL: "https://example/microsandbox-linux-x86_64.tar.gz"},
		{OS: "linux", Arch: "arm64", URL: "https://example/microsandbox-linux-aarch64.tar.gz"},
		{OS: "darwin", Arch: "arm64", URL: "https://example/microsandbox-darwin-aarch64.tar.gz"},
		{OS: "windows", Arch: "amd64", URL: "https://example/microsandbox-windows-x86_64.tar.gz"},
		{OS: "windows", Arch: "amd64", URL: "https://example/microsandbox-windows-x86_64.zip"},
	}

	chosen := selectMicrosandboxArtifact(artifacts, "linux", "amd64")
	require.NotNil(t, chosen)
	require.Contains(t, chosen.URL, "microsandbox-linux-x86_64.tar.gz")

	chosen = selectMicrosandboxArtifact(artifacts, "linux", "arm64")
	require.NotNil(t, chosen)
	require.Contains(t, chosen.URL, "microsandbox-linux-aarch64.tar.gz")

	chosen = selectMicrosandboxArtifact(artifacts, "windows", "amd64")
	require.NotNil(t, chosen)
	require.Contains(t, chosen.URL, "microsandbox-windows-x86_64.tar.gz")
}

func TestSelectMicrosandboxArtifactRejectsStandaloneBinary(t *testing.T) {
	t.Parallel()

	artifacts := []tool.Artifact{
		{OS: "linux", Arch: "amd64", URL: "https://example/msb-linux-x86_64"},
	}
	require.Nil(t, selectMicrosandboxArtifact(artifacts, "linux", "amd64"))
}

func TestIsMicrosandboxBundle(t *testing.T) {
	t.Parallel()

	for _, url := range []string{
		"https://example/microsandbox-linux-x86_64.tar.gz",
		"https://example/microsandbox-windows-x86_64.zip",
	} {
		require.True(t, isMicrosandboxBundle(url), url)
	}
	for _, url := range []string{
		"https://example/msb-linux-x86_64",
		"https://example/msb-metrics-linux-x86_64",
		"https://example/libmicrosandbox_go_ffi-linux-amd64.so",
	} {
		require.False(t, isMicrosandboxBundle(url), url)
	}
}

func TestMicrosandboxRegistryEntry(t *testing.T) {
	t.Parallel()

	installed, err := registry.NewTool("microsandbox")
	require.NoError(t, err)

	pinner, ok := installed.(tool.Pinner)
	require.True(t, ok)
	require.Equal(t, tool.Pin{Name: "superradcompany/microsandbox", Datasource: "github-releases", Versioning: "semver"}, pinner.Pin())

	checker, ok := installed.(tool.Checker)
	require.True(t, ok)
	checks := checker.InstallChecks()
	require.Len(t, checks, 1)
	require.Equal(t, "binary:msb", checks[0].Name())
}
