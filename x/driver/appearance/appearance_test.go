package appearance

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	require.Equal(t, Dark, Parse("dark"))
	require.Equal(t, Dark, Parse("prefer-dark"))
	require.Equal(t, Dark, Parse("1"))
	require.Equal(t, Light, Parse("light"))
	require.Equal(t, Light, Parse(""))
	require.Equal(t, Light, Parse("2"))
}

func TestFromPortal(t *testing.T) {
	require.Equal(t, Dark, FromPortal(1))
	require.Equal(t, Light, FromPortal(0))
	require.Equal(t, Light, FromPortal(2))
}

func TestFromHosts(t *testing.T) {
	require.Equal(t, Dark, FromAppleInterfaceStyle("Dark"))
	require.Equal(t, Light, FromAppleInterfaceStyle(""))
	require.Equal(t, Dark, FromAppsUseLightTheme(0))
	require.Equal(t, Light, FromAppsUseLightTheme(1))
}
