package gotool

import (
	"testing"

	"github.com/lewtec/lewkit/x/toolchain/load"
	"github.com/stretchr/testify/require"
)

func TestWindowsExeSuffix(t *testing.T) {
	require.Equal(t, `C:\a\hello.exe`, windowsExe("windows", `C:\a\hello`))
	require.Equal(t, `C:\a\hello.exe`, windowsExe("windows", `C:\a\hello.exe`))
	require.Equal(t, `C:\a\HELLO.EXE`, windowsExe("windows", `C:\a\HELLO.EXE`))
	require.Equal(t, "/tmp/hello", windowsExe("linux", "/tmp/hello"))
}

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
