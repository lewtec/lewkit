package gui

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkDirAction(t *testing.T) {
	openView, useCwd := workDirAction("", false)
	assert.True(t, openView)
	assert.False(t, useCwd)
	openView, useCwd = workDirAction("", true)
	assert.False(t, openView)
	assert.True(t, useCwd)
	openView, useCwd = workDirAction("/tmp", false)
	assert.False(t, openView)
	assert.False(t, useCwd)
}

func TestEnsureDirGiven(t *testing.T) {
	dir := t.TempDir()
	got, err := EnsureDir(t.Context(), dir)
	require.NoError(t, err)
	assert.Equal(t, absPath(t, dir), got)
}

func TestEnsureDirMissing(t *testing.T) {
	_, err := EnsureDir(t.Context(), filepathMissing(t))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func filepathMissing(t *testing.T) string {
	t.Helper()
	return t.TempDir() + "/nope"
}
