package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/driver/daynight"
)

func TestPublish(t *testing.T) {
	live := New(daynight.Light)
	got, err := live.Current(t.Context())
	require.NoError(t, err)
	require.Equal(t, daynight.Light, got)
	changes, err := live.Watch(t.Context())
	require.NoError(t, err)
	require.Equal(t, daynight.Light, <-changes)
	live.Publish(daynight.Light)
	live.Publish(daynight.Dark)
	select {
	case scheme := <-changes:
		require.Equal(t, daynight.Dark, scheme)
	case <-time.After(time.Second):
		require.Fail(t, "scheme did not change")
	}
}
