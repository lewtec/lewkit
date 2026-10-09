package gtk

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNeedsDisplay(t *testing.T) {
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")
	err := chooserFactory{}.CheckCompatibility(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
	if driver.ForGOOS("linux") == nil {
		assert.Contains(t, err.Error(), "DISPLAY")
	}
}
