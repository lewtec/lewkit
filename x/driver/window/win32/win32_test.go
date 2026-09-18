package win32

import (
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckCompatibility(t *testing.T) {
	err := factory{}.CheckCompatibility(t.Context())
	if runtime.GOOS == "windows" {
		require.NoError(t, err)
		return
	}
	require.ErrorIs(t, err, driver.ErrIncompatible)
	assert.Contains(t, err.Error(), "windows")
}
