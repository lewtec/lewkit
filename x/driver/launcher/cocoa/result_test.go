package cocoa

import (
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/require"
)

func TestButtonIndex(t *testing.T) {
	index, ok := buttonIndex(alertFirstButton, 2)
	require.True(t, ok)
	require.Equal(t, 0, index)
	index, ok = buttonIndex(alertFirstButton+1, 2)
	require.True(t, ok)
	require.Equal(t, 1, index)
	_, ok = buttonIndex(alertFirstButton+2, 2)
	require.False(t, ok)
	_, ok = buttonIndex(0, 2)
	require.False(t, ok)
}

func TestNotDarwin(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip()
	}
	err := chooserFactory{}.CheckCompatibility(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
	err = prompterFactory{}.CheckCompatibility(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
	err = confirmerFactory{}.CheckCompatibility(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
}
