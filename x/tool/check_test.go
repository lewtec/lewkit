package tool

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestBinaryCandidates(t *testing.T) {
	base := filepath.Join("tools", "pkg", "1.0.0")
	got := BinaryCandidates(base, "gh")
	want := []string{
		filepath.Join(base, "bin", "gh"),
		filepath.Join(base, "bin", "gh.exe"),
		filepath.Join(base, "bin", "gh.cmd"),
		filepath.Join(base, "bin", "gh.bat"),
		filepath.Join(base, "gh"),
		filepath.Join(base, "gh.exe"),
		filepath.Join(base, "gh.cmd"),
		filepath.Join(base, "gh.bat"),
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestCheckRejectsParent(t *testing.T) {
	destination := t.TempDir()
	err := FileExists("../outside").Check(t.Context(), destination)
	if !errors.Is(err, ErrPathEscapes) {
		t.Fatalf("Check() = %v, want ErrPathEscapes", err)
	}
}

func TestCheckEmptyRelativePath(t *testing.T) {
	destination := t.TempDir()
	err := FileExists(".").Check(t.Context(), destination)
	if !errors.Is(err, ErrEmptyRelativePath) {
		t.Fatalf("Check() = %v, want ErrEmptyRelativePath", err)
	}
}

func TestBinaryCheck(t *testing.T) {
	destination := t.TempDir()
	path := filepath.Join(destination, "bin")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "demo"), []byte("ok"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Binary("demo").Check(t.Context(), destination); err != nil {
		t.Fatal(err)
	}
	if err := Binary("missing").Check(t.Context(), destination); !errors.Is(err, ErrBinaryNotFound) {
		t.Fatalf("missing binary = %v, want ErrBinaryNotFound", err)
	}
}
