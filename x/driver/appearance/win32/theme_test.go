package win32

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/appearance"
	"github.com/stretchr/testify/require"
)

func TestFromLightTheme(t *testing.T) {
	require.Equal(t, appearance.Dark, fromLightTheme(0))
	require.Equal(t, appearance.Light, fromLightTheme(1))
}
