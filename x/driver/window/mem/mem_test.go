package mem

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/window"
)

func TestMemoryWindowIsOptIn(t *testing.T) {
	t.Setenv("LEWKIT_ENABLE_MEMORY_DRIVER", "")
	if err := (factory{}).CheckCompatibility(t.Context()); err == nil {
		t.Fatal("memory window is compatible without LEWKIT_ENABLE_MEMORY_DRIVER")
	}
	t.Setenv("LEWKIT_ENABLE_MEMORY_DRIVER", "1")
	if err := (factory{}).CheckCompatibility(t.Context()); err != nil {
		t.Fatal(err)
	}
	w, err := window.Open(t.Context(), window.Config{Width: 8, Height: 8})
	if err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
}
