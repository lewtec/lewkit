package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNameAndArgs(t *testing.T) {
	if got, want := Name(), filepath.Base(os.Args[0]); got != want {
		t.Fatalf("Name() = %q, want %q", got, want)
	}
	got := Args()
	want := os.Args[1:]
	if len(got) != len(want) {
		t.Fatalf("Args() len %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Args()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
