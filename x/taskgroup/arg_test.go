package taskgroup

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArgApplyOverridesPositiveFlags(t *testing.T) {
	got := cmd.ParseOK[Arg](t, "--io", "8", "--cpu", "2")
	base := Limits{IO: 4, CPU: 16, Internet: 4}
	lim := got.Apply(base)
	assert.Equal(t, 8, lim.IO)
	assert.Equal(t, 2, lim.CPU)
	assert.Equal(t, 4, lim.Internet)
}

func TestArgApplyKeepsBaseWhenFlagsOmitted(t *testing.T) {
	got := cmd.ParseOK[Arg](t)
	base := Limits{IO: 3, CPU: 5, Internet: 7}
	require.Equal(t, base, got.Apply(base))
}

func TestArgEnterReturnsSession(t *testing.T) {
	arg := cmd.ParseOK[Arg](t, "--io", "2")
	s, ctx := arg.Enter(t.Context(), DefaultLimits())
	require.NotNil(t, s)
	require.Equal(t, s, FromContext(ctx))
	t.Cleanup(func() {
		if err := s.Wait(); err != nil {
			t.Logf("Wait: %v", err)
		}
	})

	var ran atomic.Bool
	Go(ctx, "t", IO, func(context.Context, *Status) error {
		ran.Store(true)
		return nil
	})
	require.NoError(t, s.Wait())
	require.True(t, ran.Load())
}
