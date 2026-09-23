package dotfiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoot(t *testing.T) {
	prev := Candidates
	t.Cleanup(func() { Candidates = prev })

	home := t.TempDir()
	dotfiles := filepath.Join(home, ".dotfiles")
	require.NoError(t, os.Mkdir(dotfiles, 0o755))
	Candidates = []string{"~/.dotfiles", filepath.Join(home, "later")}
	got, err := Root(home)
	require.NoError(t, err)
	require.Equal(t, dotfiles, got)

	first := filepath.Join(home, "first")
	require.NoError(t, os.Mkdir(first, 0o755))
	Candidates = []string{first, dotfiles}
	got, err = Root(home)
	require.NoError(t, err)
	require.Equal(t, first, got)

	viaEnv := filepath.Join(home, "from-env")
	require.NoError(t, os.Mkdir(viaEnv, 0o755))
	t.Setenv("LEWKIT_DOTFILES_TEST", viaEnv)
	Candidates = []string{"$LEWKIT_DOTFILES_TEST"}
	got, err = Root(home)
	require.NoError(t, err)
	require.Equal(t, viaEnv, got)

	Candidates = []string{filepath.Join(home, "missing")}
	_, err = Root(home)
	require.ErrorIs(t, err, ErrNotFound)
}
