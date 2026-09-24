package fixed

import (
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/stretchr/testify/require"
)

func TestWatch(t *testing.T) {
	t.Setenv("LEWKIT_DAYNIGHT", "light")
	t.Setenv("LEWKIT_FORCE_DAYNIGHT_DRIVER", "daynight_fixed")
	changes, err := daynight.Watch(t.Context())
	require.NoError(t, err)
	require.Equal(t, daynight.Light, <-changes)
	Set(daynight.Dark)
	select {
	case scheme := <-changes:
		require.Equal(t, daynight.Dark, scheme)
	case <-time.After(time.Second):
		require.Fail(t, "scheme did not change")
	}
}
