package mem

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
)

func TestMemoryWindowIsOptIn(t *testing.T) {
	if err := (factory{}).CheckCompatibility(t.Context()); err == nil {
		t.Fatal("memory window is compatible without LEWKIT_ENABLE_MEMORY_DRIVER")
	}
	ctx := driver.WithEnv(t.Context(), []string{"LEWKIT_ENABLE_MEMORY_DRIVER=1"})
	if err := (factory{}).CheckCompatibility(ctx); err != nil {
		t.Fatal(err)
	}
	w, err := window.Open(ctx, window.Config{Width: 8, Height: 8})
	if err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
}
