package taskgroup

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsolateDoesNotCancelParentSiblings(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	release := make(chan struct{})
	var siblingRan atomic.Bool

	Go(ctx, "sibling", CPU, func(context.Context, *Status) error {
		<-release
		siblingRan.Store(true)
		return nil
	})

	isolated := errors.New("isolated failure")
	err := Isolate(ctx, func(ctx context.Context) error {
		Go(ctx, "boom", CPU, func(context.Context, *Status) error {
			return isolated
		})
		return MustFromContext(ctx).waitLive(ctx, taskFromContext(ctx))
	})
	require.Error(t, err)
	close(release)
	require.NoError(t, MustFromContext(ctx).Wait())
	assert.True(t, siblingRan.Load())
}

func TestGoIsolatedWithoutSessionRunsSync(t *testing.T) {
	var ran bool
	err := GoIsolated(t.Context(), "x", CPU, func(ctx context.Context, s *Status) error {
		ran = true
		require.NotNil(t, s)
		return nil
	})
	require.NoError(t, err)
	assert.True(t, ran)
}

func TestGoIsolatedNamedChild(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	err := GoIsolated(ctx, "install:foo", Control, func(ctx context.Context, s *Status) error {
		s.Update("done")
		return nil
	})
	require.NoError(t, err)
	require.NoError(t, MustFromContext(ctx).Wait())
}
