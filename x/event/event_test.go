package event

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscribePublish(t *testing.T) {
	bus := New[int]()
	ch := bus.Subscribe(t.Context())
	bus.Publish(7)
	got, ok := <-ch
	require.True(t, ok)
	assert.Equal(t, 7, got)
}

func TestCreateTimer(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	ch := CreateTimer(ctx, time.Millisecond)
	select {
	case <-ch:
	case <-time.After(time.Second):
		require.FailNow(t, "first tick")
	}
	select {
	case <-ch:
	case <-time.After(time.Second):
		require.FailNow(t, "second tick")
	}
	cancel()
	time.Sleep(5 * time.Millisecond)
	n := 0
	for {
		select {
		case <-ch:
			n++
			require.LessOrEqual(t, n, 8, "timer kept sending after cancel")
		default:
			return
		}
	}
}

func TestCreateTimerDropsIfSlow(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	ch := CreateTimer(ctx, time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	got := 0
	for {
		select {
		case <-ch:
			got++
		default:
			assert.Equal(t, 1, got)
			return
		}
	}
}

func TestSubscribeCancelCloses(t *testing.T) {
	bus := New[int]()
	ctx, cancel := context.WithCancel(t.Context())
	ch := bus.Subscribe(ctx)
	cancel()
	_, ok := <-ch
	assert.False(t, ok)
}
