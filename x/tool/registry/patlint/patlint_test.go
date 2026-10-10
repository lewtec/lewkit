package pkg

import (
	"testing"

	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry"

	"github.com/stretchr/testify/require"
)

func TestPatlintPinsLewtecRelease(t *testing.T) {
	t.Parallel()

	installed, err := registry.NewTool("patlint")
	require.NoError(t, err)

	pinner, ok := installed.(tool.Pinner)
	require.True(t, ok)
	require.Equal(t, tool.Pin{Name: "lewtec/patlint", Datasource: "github-releases"}, pinner.Pin())

	checker, ok := installed.(tool.Checker)
	require.True(t, ok)
	checks := checker.InstallChecks()
	require.Len(t, checks, 1)
	require.Equal(t, "binary:patlint", checks[0].Name())
}

func TestPatlintReleaseSelectsGoreleaserArchive(t *testing.T) {
	t.Parallel()

	installed, err := registry.NewTool("patlint")
	require.NoError(t, err)

	versions, err := installed.ListVersions(t.Context())
	require.NoError(t, err)
	require.Contains(t, versions, "0.0.1")

	lister, ok := installed.(tool.ArtifactTool)
	require.True(t, ok)
	artifacts, err := lister.ListArtifacts(t.Context(), "0.0.1")
	require.NoError(t, err)

	chosen := tool.SelectArtifact(artifacts, "linux", "amd64", "patlint")
	require.NotNil(t, chosen)
	require.Contains(t, chosen.URL, "patlint_Linux_x86_64.tar.gz")
}
