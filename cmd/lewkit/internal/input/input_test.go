package input

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestReadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "in.bin")
	if err := os.WriteFile(path, []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Read(t.Context(), path)
	if err != nil || string(got) != "abc" {
		t.Fatalf("read %q %v", got, err)
	}
}

func TestReadCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := Read(ctx, "missing"); err == nil {
		t.Fatal("expected cancel")
	}
}
