package atomic

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCommitReplacesDirectory(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "tool")
	if err := os.MkdirAll(filepath.Join(destination, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "bin", "old"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	operation := NewOperation(destination, true)
	if err := os.MkdirAll(filepath.Join(operation.StagingPath(), "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(operation.StagingPath(), "bin", "new"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := operation.Commit(); err != nil {
		t.Fatal(err)
	}

	body, err := os.ReadFile(filepath.Join(destination, "bin", "new"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "new" {
		t.Fatalf("body = %q", body)
	}
	if _, err := os.Stat(filepath.Join(destination, "bin", "old")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old tree still present: %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "tool" {
		t.Fatalf("root entries = %v, want only tool", names(entries))
	}
}

func TestCommitCreatesMissingDirectory(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "tool")
	operation := NewOperation(destination, true)
	if err := os.MkdirAll(operation.StagingPath(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(operation.StagingPath(), "marker"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := operation.Commit(); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(destination, "marker"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "ok" {
		t.Fatalf("body = %q", body)
	}
}

func TestWriteStringReplacesFile(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "note")
	if err := WriteString(path, "one"); err != nil {
		t.Fatal(err)
	}
	if err := WriteString(path, "two"); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "two" {
		t.Fatalf("body = %q", body)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "note" {
		t.Fatalf("directory entries = %v, want only note", names(entries))
	}
}

func names(entries []os.DirEntry) []string {
	out := make([]string, len(entries))
	for i, entry := range entries {
		out[i] = entry.Name()
	}
	return out
}
