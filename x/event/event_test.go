package event

import (
	"context"
	"testing"

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

func TestSubscribeCancelCloses(t *testing.T) {
	bus := New[int]()
	ctx, cancel := context.WithCancel(t.Context())
	ch := bus.Subscribe(ctx)
	cancel()
	_, ok := <-ch
	assert.False(t, ok)
}
