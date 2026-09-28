package common

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveWorkDir_CreatesMissingPath(t *testing.T) {
	want := filepath.Join(t.TempDir(), "nested", "work")
	got, cleanup, err := ResolveWorkDir(want, "eletrocromo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	if cleanup {
		t.Fatal("explicit work dir should not be marked cleanup")
	}
	abs, err := filepath.Abs(want)
	if err != nil {
		t.Fatal(err)
	}
	if got != abs {
		t.Fatalf("got %q want %q", got, abs)
	}
	st, err := os.Stat(got)
	if err != nil {
		t.Fatal(err)
	}
	if !st.IsDir() {
		t.Fatalf("%s is not a directory", got)
	}
}

func TestResolveWorkDir_EmptyMakesTemp(t *testing.T) {
	dir, cleanup, err := ResolveWorkDir("", "eletrocromo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	if !cleanup {
		t.Fatal("temp work dir should be marked cleanup")
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("cleanup temp work dir: %v", err)
		}
	})
	if !strings.Contains(filepath.Base(dir), "eletrocromo-test-") {
		t.Fatalf("temp name %q missing prefix", dir)
	}
	st, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !st.IsDir() {
		t.Fatalf("%s is not a directory", dir)
	}
}
