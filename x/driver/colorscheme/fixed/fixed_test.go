package fixed

import (
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/colorscheme"
	"github.com/stretchr/testify/require"
)

func TestWatch(t *testing.T) {
	t.Setenv("LEWKIT_COLORSCHEME", "light")
	t.Setenv("LEWKIT_FORCE_COLORSCHEME_DRIVER", "colorscheme_fixed")
	changes, err := colorscheme.Watch(t.Context())
	require.NoError(t, err)
	require.Equal(t, colorscheme.Light, <-changes)
	Set(colorscheme.Dark)
	select {
	case scheme := <-changes:
		require.Equal(t, colorscheme.Dark, scheme)
	case <-time.After(time.Second):
		require.Fail(t, "scheme did not change")
	}
}
