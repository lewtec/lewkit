package mem

import (
	"fmt"
	"testing"

	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/stretchr/testify/require"
)

func TestMemoryWindowIsOptIn(t *testing.T) {
	t.Setenv("LEWKIT_ENABLE_MEMORY_DRIVER", "")
	if err := (factory{}).CheckCompatibility(t.Context()); err == nil {
		require.FailNow(t, fmt.Sprint("memory window is compatible without LEWKIT_ENABLE_MEMORY_DRIVER"))
	}
	t.Setenv("LEWKIT_ENABLE_MEMORY_DRIVER", "1")
	require.NoError(t, (factory{}).CheckCompatibility(t.Context()))
	w, err := window.Open(t.Context(), window.Config{Width: 8, Height: 8})
	require.NoError(t, err)
	_ = w.Close()
}
