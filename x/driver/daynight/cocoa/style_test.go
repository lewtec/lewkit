package cocoa

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/stretchr/testify/require"
)

func TestFromInterfaceStyle(t *testing.T) {
	require.Equal(t, daynight.Dark, fromInterfaceStyle("Dark"))
	require.Equal(t, daynight.Light, fromInterfaceStyle(""))
	require.Equal(t, daynight.Light, fromInterfaceStyle("Light"))
}
