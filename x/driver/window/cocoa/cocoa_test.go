package cocoa

import (
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenNeedsBind(t *testing.T) {
	_, err := cdriver{}.Open(t.Context(), window.Config{Width: 8, Height: 8})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Bind")
}

func TestCheckCompatibility(t *testing.T) {
	err := factory{}.CheckCompatibility(t.Context())
	if runtime.GOOS == "darwin" {
		require.NoError(t, err)
		return
	}
	require.ErrorIs(t, err, driver.ErrIncompatible)
	assert.Contains(t, err.Error(), "darwin")
}
