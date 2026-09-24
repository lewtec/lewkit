package win32

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/stretchr/testify/require"
)

func TestFromLightTheme(t *testing.T) {
	require.Equal(t, daynight.Dark, fromLightTheme(0))
	require.Equal(t, daynight.Light, fromLightTheme(1))
}
