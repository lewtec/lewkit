package common

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveCreateOut_Empty(t *testing.T) {
	_, err := ResolveCreateOut("  ", false)
	require.ErrorIs(t, err, ErrOutDirRequired)
}

func TestResolveCreateOut_CreatesAndWritesJSON(t *testing.T) {
	out, err := ResolveCreateOut(filepath.Join(t.TempDir(), "host"), false)
	require.NoError(t, err)
	require.NoError(t, WriteHostJSON(out, []byte("{}\n")))
	raw, err := os.ReadFile(filepath.Join(out, HostConfigFile))
	require.NoError(t, err)
	if string(raw) != "{}\n" {
		require.FailNowf(t, "", "json: %q", raw)
	}
}

func TestPrepareOutDir_CreatesAndForceWipes(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "host")
	require.NoError(t, PrepareOutDir(out, false))
	require.NoError(t, os.WriteFile(filepath.Join(out, "stale.txt"), []byte("old"), 0o644))
	err := PrepareOutDir(out, false)
	require.ErrorIs(t, err, ErrOutDirNotEmpty)
	require.NoError(t, PrepareOutDir(out, true))
	entries, err := os.ReadDir(out)
	require.NoError(t, err)
	if len(entries) != 0 {
		require.FailNowf(t, "", "force wipe left %d entries", len(entries))
	}
}

func TestPrepareOutDir_RejectsFile(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "notdir")
	require.NoError(t, os.WriteFile(out, []byte("x"), 0o644))
	err := PrepareOutDir(out, true)
	require.ErrorIs(t, err, ErrOutPathNotDir)
}

func TestResolveGoMain_RejectsFile(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "main.go")
	require.NoError(t, os.WriteFile(file, []byte("package main\n"), 0o644))
	_, err := ResolveGoMain(file, root)
	require.ErrorIs(t, err, ErrGoMainNotDir)
}
