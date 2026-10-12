package mem

import (
	"image"
	"testing"

	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigScale(t *testing.T) {
	t.Setenv("LEWKIT_ENABLE_MEMORY_DRIVER", "1")
	w, err := opener{}.Open(t.Context(), window.Config{Width: 8, Height: 6, Scale: 2})
	require.NoError(t, err)
	t.Cleanup(func() { _ = w.Close() })
	assert.Equal(t, image.Pt(16, 12), w.Size())
	assert.Equal(t, window.Scale(2), w.Scale())

	plain, err := opener{}.Open(t.Context(), window.Config{Width: 8, Height: 6})
	require.NoError(t, err)
	t.Cleanup(func() { _ = plain.Close() })
	assert.Equal(t, image.Pt(8, 6), plain.Size())
	assert.Equal(t, window.Scale(1), plain.Scale())
}
