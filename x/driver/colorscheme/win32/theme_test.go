package win32

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/colorscheme"
	"github.com/stretchr/testify/require"
)

func TestFromLightTheme(t *testing.T) {
	require.Equal(t, colorscheme.Dark, fromLightTheme(0))
	require.Equal(t, colorscheme.Light, fromLightTheme(1))
}
