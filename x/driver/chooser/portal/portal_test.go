package portal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPreferQt(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "")
	t.Setenv("XDG_SESSION_DESKTOP", "")
	require.False(t, PreferQt())

	t.Setenv("XDG_CURRENT_DESKTOP", "KDE")
	require.True(t, PreferQt())

	t.Setenv("XDG_CURRENT_DESKTOP", "")
	t.Setenv("XDG_SESSION_DESKTOP", "plasma")
	require.True(t, PreferQt())
}
