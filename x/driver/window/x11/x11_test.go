package x11

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckNeedsDisplay(t *testing.T) {
	t.Setenv("DISPLAY", "")
	err := factory{}.CheckCompatibility(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
	assert.Contains(t, err.Error(), "DISPLAY")
}
