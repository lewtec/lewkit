package taskgroup

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListEmpty(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	assert.Nil(t, List(ctx, 8))
	assert.Nil(t, List(ctx, 0))
}

func TestListWalksTreeUntilFull(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	started := make(chan struct{})
	block := make(chan struct{})
	const n = 10_000

	Go(ctx, "apply", Control, func(ctx context.Context, s *Status) error {
		s.Progress(0, n)
		for range n {
			Go(ctx, "file", IO, func(context.Context, *Status) error {
				<-block
				return nil
			})
		}
		close(started)
		<-block
		return nil
	})
	<-started

	got := List(ctx, 8)
	require.Len(t, got, 8)
	assert.Equal(t, "apply", got[0].Name)
	assert.Equal(t, n, got[0].LiveChildren)
	assert.Equal(t, ID(0), got[0].Parent)
	parent := got[0].ID
	for i, row := range got[1:] {
		assert.Equal(t, "file", row.Name, "row %d", i+1)
		assert.Equal(t, parent, row.Parent, "row %d", i+1)
	}

	close(block)
	require.NoError(t, MustFromContext(ctx).Wait())
	assert.Empty(t, List(ctx, 8))
}

func TestListKeepsFinishedParentWhileChildrenRun(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	started := make(chan struct{})
	block := make(chan struct{})

	Go(ctx, "apply", Control, func(ctx context.Context, _ *Status) error {
		Go(ctx, "file", IO, func(context.Context, *Status) error {
			close(started)
			<-block
			return nil
		})
		return nil
	})
	<-started

	var got []Node
	require.Eventually(t, func() bool {
		got = List(ctx, 8)
		return len(got) >= 2 && got[0].Name == "apply" && got[0].State == Done
	}, time.Second, time.Millisecond)
	assert.Equal(t, 1, got[0].LiveChildren)

	close(block)
	require.NoError(t, MustFromContext(ctx).Wait())
	assert.Empty(t, List(ctx, 8))
}

func TestListSkipsIsolateBoundary(t *testing.T) {
	_, ctx := newTest(t, DefaultLimits())
	started := make(chan struct{})
	block := make(chan struct{})
	var saw atomic.Bool

	Go(ctx, "bundle", Control, func(ctx context.Context, _ *Status) error {
		return Isolate(ctx, func(ctx context.Context) error {
			Go(ctx, "bundle:icons", CPU, func(context.Context, *Status) error {
				saw.Store(true)
				close(started)
				<-block
				return nil
			})
			return MustFromContext(ctx).waitLive(ctx, taskFromContext(ctx))
		})
	})
	<-started

	got := List(ctx, 16)
	for _, n := range got {
		assert.NotEmpty(t, n.Name)
	}
	require.GreaterOrEqual(t, len(got), 2)
	assert.Equal(t, "bundle", got[0].Name)
	assert.Equal(t, "bundle:icons", got[1].Name)
	assert.Equal(t, got[0].ID, got[1].Parent)

	close(block)
	require.NoError(t, MustFromContext(ctx).Wait())
	assert.True(t, saw.Load())
}

func TestListNilContext(t *testing.T) {
	assert.Nil(t, List(t.Context(), 4))
}
