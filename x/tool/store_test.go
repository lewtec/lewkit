package tool

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type memoryBackend struct{}

func (memoryBackend) Name() string { return "memory" }

var (
	errUnknownPackage    = errors.New("unknown package")
	errUnexpectedVersion = errors.New("unexpected version")
)

func (memoryBackend) Tool(ref string) (Tool, error) {
	switch ref {
	case "demo", "left", "right":
		return memoryTool{name: ref}, nil
	default:
		return nil, errUnknownPackage
	}
}

type memoryTool struct {
	name string
}

func (memoryTool) ListVersions(context.Context) ([]string, error) {
	return []string{"1.2.3"}, nil
}

func (t memoryTool) Install(_ context.Context, version, destination string) error {
	if version != "1.2.3" {
		return errUnexpectedVersion
	}
	directory := filepath.Join(destination, "bin")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(directory, t.name), []byte("ok"), 0o755); err != nil {
		return err
	}
	if t.name == "left" || t.name == "right" {
		return os.WriteFile(filepath.Join(directory, "run"), []byte(t.name), 0o755)
	}
	return nil
}

func (t memoryTool) InstallChecks() []Check {
	return []Check{Binary(t.name)}
}

var memoryOnce sync.Once

func useMemory() {
	memoryOnce.Do(func() { Register("memory", memoryBackend{}) })
}

func TestEnsureInstallsAndReuses(t *testing.T) {
	useMemory()
	store, err := Open(t.TempDir())
	require.NoError(t, err)
	first, err := store.Ensure(t.Context(), "memory:demo@1.2.3", "demo")
	require.NoError(t, err)
	second, err := store.Ensure(t.Context(), "memory:demo@latest", "demo")
	require.NoError(t, err)
	require.Equal(t, first, second)
	replaced, err := store.Ensure(WithNoCache(t.Context()), "memory:demo@1.2.3", "demo")
	require.NoError(t, err)
	require.Equal(t, first, replaced)
	installed, err := store.ListInstalled()
	require.NoError(t, err)
	require.Len(t, installed, 1)
	require.Equal(t, "1.2.3", installed[0].Version)
	resolved, err := store.Resolve(t.Context(), "demo")
	require.NoError(t, err)
	require.Equal(t, first, resolved)
}

func TestEnsureCommandUsesRightmostBinary(t *testing.T) {
	useMemory()
	store, err := Open(t.TempDir())
	require.NoError(t, err)

	path, err := store.EnsureCommand(t.Context(), []string{"memory:left@1.2.3", "memory:right@1.2.3"}, "run")
	require.NoError(t, err)
	require.Contains(t, path, "memory-right")

	path, err = store.EnsureCommand(t.Context(), []string{"memory:left@1.2.3", "memory:demo@1.2.3"}, "run")
	require.NoError(t, err)
	require.Contains(t, path, "memory-left")

	_, err = store.EnsureCommand(t.Context(), []string{"memory:demo@1.2.3"}, "run")
	require.Error(t, err)
	require.Contains(t, err.Error(), "provide a binary")
}

func TestOpenRejectsEmptyRoot(t *testing.T) {
	_, err := Open("  ")
	require.ErrorIs(t, err, ErrEmptyStore)
}
