package thread

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
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
