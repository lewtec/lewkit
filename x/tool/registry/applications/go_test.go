package applications

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGoListArtifactsAcceptsVersionWithoutGoPrefix(t *testing.T) {
	t.Parallel()

	installed := &goTool{}
	ctx := t.Context()

	versions, err := installed.ListVersions(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, versions)

	artifacts, err := installed.ListArtifacts(ctx, versions[0])
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	require.Contains(t, artifacts[0].URL, "go"+versions[0])
	require.Contains(t, artifacts[0].URL, runtime.GOOS+"-"+runtime.GOARCH)
	require.Contains(t, artifacts[0].Hash, "sha256:")
}

func TestGoListArtifactsAcceptsGoPrefixedVersion(t *testing.T) {
	t.Parallel()

	installed := &goTool{}
	ctx := t.Context()

	versions, err := installed.ListVersions(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, versions)

	prefixed := "go" + versions[0]
	artifacts, err := installed.ListArtifacts(ctx, prefixed)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	require.Contains(t, artifacts[0].URL, "go"+versions[0])
}
