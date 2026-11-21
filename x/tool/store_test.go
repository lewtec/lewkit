package tool

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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
	if ref != "demo" {
		return nil, errUnknownPackage
	}
	return memoryTool{}, nil
}

type memoryTool struct{}

func (memoryTool) ListVersions(context.Context) ([]string, error) {
	return []string{"1.2.3"}, nil
}

func (memoryTool) Install(_ context.Context, version, destination string) error {
	if version != "1.2.3" {
		return errUnexpectedVersion
	}
	directory := filepath.Join(destination, "bin")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(directory, "demo"), []byte("ok"), 0o755)
}

func (memoryTool) InstallChecks() []Check {
	return []Check{Binary("demo")}
}

func TestEnsureInstallsAndReuses(t *testing.T) {
	Register("memory", memoryBackend{})
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

func TestOpenRejectsEmptyRoot(t *testing.T) {
	_, err := Open("  ")
	require.ErrorIs(t, err, ErrEmptyStore)
}
