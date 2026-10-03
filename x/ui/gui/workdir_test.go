package gui

import (
	"context"
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

func TestChooseDirCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := ChooseDir(ctx, WelcomeArgs{Title: "lewkit"})
	require.ErrorIs(t, err, context.Canceled)
}

func TestFinishChooseRemembers(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	welcome := NewWelcome(WelcomeArgs{Title: "gaderno"})
	welcome.picked = dir
	got, err := finishChoose(t.Context(), welcome, os.ErrClosed)
	require.NoError(t, err)
	assert.Equal(t, absPath(t, dir), got)
	recent, err := Recent()
	require.NoError(t, err)
	require.Len(t, recent, 1)
	assert.Equal(t, absPath(t, dir), recent[0].Path)
}

func TestFinishChooseClosed(t *testing.T) {
	welcome := NewWelcome(WelcomeArgs{Title: "lewkit"})
	_, err := finishChoose(t.Context(), welcome, nil)
	require.ErrorIs(t, err, ErrCanceled)
}

func TestFinishChooseOpenError(t *testing.T) {
	welcome := NewWelcome(WelcomeArgs{Title: "lewkit"})
	_, err := finishChoose(t.Context(), welcome, os.ErrClosed)
	require.ErrorIs(t, err, os.ErrClosed)
}

func filepathMissing(t *testing.T) string {
	t.Helper()
	return t.TempDir() + "/nope"
}
