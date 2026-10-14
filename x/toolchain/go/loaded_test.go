package gotool

import (
	"testing"

	"github.com/lewtec/lewkit/x/toolchain/load"
	"github.com/stretchr/testify/require"
)

func TestBuildLoadedRejectsTwoMains(t *testing.T) {
	_, err := BuildLoaded(t.Context(), Loaded{
		Graph: &load.Graph{Pkgs: []load.Package{{Name: "main"}, {Name: "main"}}},
	})
	require.EqualError(t, err, "release go: 2 main packages")
}

func TestBuildLoadedNeedsMain(t *testing.T) {
	_, err := BuildLoaded(t.Context(), Loaded{
		Graph:       &load.Graph{Pkgs: []load.Package{{Name: "p"}}},
		RequireMain: true,
	})
	require.EqualError(t, err, "release go: -o needs a main package")
}

func TestBuildLoadedRejectsAssemblyMain(t *testing.T) {
	_, err := BuildLoaded(t.Context(), Loaded{
		Graph: &load.Graph{Pkgs: []load.Package{{Name: "main", Asm: true}}},
	})
	require.EqualError(t, err, "release go: main package has assembly")
}

func TestBuildLoadedMissingArchive(t *testing.T) {
	_, err := BuildLoaded(t.Context(), Loaded{
		Graph: &load.Graph{Pkgs: []load.Package{{
			Name:       "p",
			ImportPath: "example.com/p",
			Asm:        true,
		}}},
		Std: map[string]string{},
	})
	require.ErrorIs(t, err, load.ErrNoArchive)
	require.EqualError(t, err, "load: archive is missing: example.com/p")
}
