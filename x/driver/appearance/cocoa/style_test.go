package cocoa

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/appearance"
	"github.com/stretchr/testify/require"
)

func TestFromInterfaceStyle(t *testing.T) {
	require.Equal(t, appearance.Dark, fromInterfaceStyle("Dark"))
	require.Equal(t, appearance.Light, fromInterfaceStyle(""))
	require.Equal(t, appearance.Light, fromInterfaceStyle("Light"))
}
