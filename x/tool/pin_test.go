package tool

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type semverInner struct{}

func (semverInner) Pin() Pin { return Pin{Name: "inner", Datasource: "github-releases"} }

func TestInheritSemver(t *testing.T) {
	t.Parallel()
	got := InheritSemver(semverInner{})
	require.Equal(t, Pin{Name: "inner", Datasource: "github-releases", Versioning: "semver"}, got)
	got = InheritSemver("plain")
	require.Equal(t, SemverPin(), got)
}
