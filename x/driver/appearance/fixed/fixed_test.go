package fixed

import (
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/appearance"
	"github.com/stretchr/testify/require"
)

func TestWatch(t *testing.T) {
	t.Setenv("LEWKIT_APPEARANCE", "light")
	t.Setenv("LEWKIT_FORCE_APPEARANCE_DRIVER", "appearance_fixed")
	changes, err := appearance.Watch(t.Context())
	require.NoError(t, err)
	require.Equal(t, appearance.Light, <-changes)
	Set(appearance.Dark)
	select {
	case scheme := <-changes:
		require.Equal(t, appearance.Dark, scheme)
	case <-time.After(time.Second):
		require.Fail(t, "scheme did not change")
	}
}
