package c

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoHeader(t *testing.T) {
	if _, _, err := Go(nil, "ping.h"); !errors.Is(err, ErrNilContext) {
		t.Fatalf("nil context: %v", err)
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "ping.h")
	body := "#define ANSWER 7\nint ping(int x);\n"
	if err := os.WriteFile(file, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	path, src, err := Go(t.Context(), file)
	if err != nil {
		t.Fatal(err)
	}
	if path != "ping" {
		t.Fatalf("import path %s", path)
	}
	if !strings.Contains(src, "package ping") || !strings.Contains(src, "ANSWER") || !strings.Contains(src, "ping") {
		t.Fatalf("source:\n%s", src)
	}
}

func TestGoRejectsPackageName(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "foo-bar.h")
	if err := os.WriteFile(file, []byte("int x;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := Go(t.Context(), file)
	if !errors.Is(err, ErrName) {
		t.Fatalf("name: %v", err)
	}
}
