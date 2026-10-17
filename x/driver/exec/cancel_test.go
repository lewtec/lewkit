package exec

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDisarmDropsTheCancelHook(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cmd := MustCommand(ctx, "sh", "-c", "true")
	require.NoError(t, Start(ctx, cmd))
	_, armed := cancelHooks.Load(cmd)
	require.True(t, armed)
	require.NoError(t, Wait(ctx, cmd))
	_, armed = cancelHooks.Load(cmd)
	require.False(t, armed)
	cancel()
	_, armed = cancelHooks.Load(cmd)
	require.False(t, armed)
}
