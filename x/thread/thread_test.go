package thread

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoFromOtherGoroutine(t *testing.T) {
	th := New()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	go func() {
		var got int
		th.Do(func() { got = 3 })
		assert.Equal(t, 3, got)
		cancel()
	}()
	th.Loop(ctx)
}

func TestDoNestedOnThread(t *testing.T) {
	th := New()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	go func() {
		var inner bool
		th.Do(func() {
			th.Do(func() { inner = true })
		})
		assert.True(t, inner)
		cancel()
	}()
	th.Loop(ctx)
}

func TestGoReturnsWhileLoopBusy(t *testing.T) {
	th := New()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	go th.Loop(ctx)
	th.Do(func() {
		done := make(chan struct{})
		go func() {
			th.Go(func() {})
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(time.Second):
			require.FailNow(t, "Go blocked while Loop busy")
		}
	})
}

func TestPackageEnqueue(t *testing.T) {
	assert.NotNil(t, Enqueue)
}

func TestEnqueueWhenJobsFull(t *testing.T) {
	th := New()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	go th.Loop(ctx)
	th.Do(func() {
		th.Go(func() {})
		done := make(chan struct{})
		go func() {
			th.Enqueue(func() {})
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(time.Second):
			require.FailNow(t, "Enqueue blocked while Loop busy")
		}
	})
}

func TestOnIdle(t *testing.T) {
	th := New()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	saw := make(chan struct{}, 1)
	th.OnIdle(func() {
		select {
		case saw <- struct{}{}:
		default:
		}
	})
	go func() {
		<-saw
		cancel()
	}()
	th.Loop(ctx)
	assert.True(t, th.Bound())
}
