package applications

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFlutterListVersionsAndArtifacts(t *testing.T) {
	t.Parallel()

	installed := &flutterTool{}
	ctx := t.Context()

	versions, err := installed.ListVersions(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, versions)

	artifacts, err := installed.ListArtifacts(ctx, versions[0])
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	artifact := artifacts[0]
	require.NotEmpty(t, artifact.URL)
	require.Contains(t, artifact.Hash, "sha256:")
	require.Contains(t, artifact.URL, "flutter_infra_release/releases")
	require.Contains(t, artifact.URL, versions[0])

	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			require.Contains(t, artifact.URL, "macos_arm64_")
		} else {
			require.Contains(t, artifact.URL, "flutter_macos_")
			require.NotContains(t, artifact.URL, "arm64")
		}
	case "linux":
		require.Contains(t, artifact.URL, "flutter_linux_")
	case "windows":
		require.Contains(t, artifact.URL, "flutter_windows_")
	}
}

func TestFlutterNormalizeAndLatest(t *testing.T) {
	t.Parallel()

	installed := &flutterTool{}
	ctx := t.Context()

	artifacts, err := installed.ListArtifacts(ctx, "latest")
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	versions, err := installed.ListVersions(ctx)
	require.NoError(t, err)
	if len(versions) > 0 {
		prefixed := "v" + versions[0]
		again, err := installed.ListArtifacts(ctx, prefixed)
		require.NoError(t, err)
		require.Len(t, again, 1)
	}
}
