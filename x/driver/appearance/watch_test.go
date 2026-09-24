package appearance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestChangesSkipsDuplicates(t *testing.T) {
	next := make(chan Scheme, 4)
	ctx := t.Context()
	out := Changes(ctx, Light, next)
	require.Equal(t, Light, <-out)
	next <- Light
	next <- Dark
	require.Equal(t, Dark, <-out)
	next <- Dark
	select {
	case scheme := <-out:
		require.Failf(t, "scheme", "duplicate %s", scheme)
	case <-time.After(30 * time.Millisecond):
	}
}
