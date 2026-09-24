package cocoa

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/colorscheme"
	"github.com/stretchr/testify/require"
)

func TestFromInterfaceStyle(t *testing.T) {
	require.Equal(t, colorscheme.Dark, fromInterfaceStyle("Dark"))
	require.Equal(t, colorscheme.Light, fromInterfaceStyle(""))
	require.Equal(t, colorscheme.Light, fromInterfaceStyle("Light"))
}
