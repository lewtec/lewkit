package common

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveCreateOut_Empty(t *testing.T) {
	_, err := ResolveCreateOut("  ", false)
	if !errors.Is(err, ErrOutDirRequired) {
		t.Fatalf("want ErrOutDirRequired, got %v", err)
	}
}

func TestResolveCreateOut_CreatesAndWritesJSON(t *testing.T) {
	out, err := ResolveCreateOut(filepath.Join(t.TempDir(), "host"), false)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteHostJSON(out, []byte("{}\n")); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(out, HostConfigFile))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "{}\n" {
		t.Fatalf("json: %q", raw)
	}
}

func TestPrepareOutDir_CreatesAndForceWipes(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "host")
	if err := PrepareOutDir(out, false); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "stale.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := PrepareOutDir(out, false); !errors.Is(err, ErrOutDirNotEmpty) {
		t.Fatalf("want ErrOutDirNotEmpty, got %v", err)
	}
	if err := PrepareOutDir(out, true); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("force wipe left %d entries", len(entries))
	}
}

func TestPrepareOutDir_RejectsFile(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "notdir")
	if err := os.WriteFile(out, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := PrepareOutDir(out, true); !errors.Is(err, ErrOutPathNotDir) {
		t.Fatalf("want ErrOutPathNotDir, got %v", err)
	}
}

func TestResolveGoMain_RejectsFile(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "main.go")
	if err := os.WriteFile(file, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ResolveGoMain(file, root)
	if !errors.Is(err, ErrGoMainNotDir) {
		t.Fatalf("want ErrGoMainNotDir, got %v", err)
	}
}
