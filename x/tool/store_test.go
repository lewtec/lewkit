package tool

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
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
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.Ensure(t.Context(), "memory:demo@1.2.3", "demo")
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Ensure(t.Context(), "memory:demo@latest", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("second Ensure = %s, want %s", second, first)
	}
	installed, err := store.ListInstalled()
	if err != nil {
		t.Fatal(err)
	}
	if len(installed) != 1 || installed[0].Version != "1.2.3" {
		t.Fatalf("installed = %+v", installed)
	}
	resolved, err := store.Resolve(t.Context(), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if resolved != first {
		t.Fatalf("Resolve = %s, want %s", resolved, first)
	}
}

func TestOpenRejectsEmptyRoot(t *testing.T) {
	_, err := Open("  ")
	if !errors.Is(err, ErrEmptyStore) {
		t.Fatalf("Open(empty) = %v, want ErrEmptyStore", err)
	}
}
