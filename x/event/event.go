// Package event is a subscribe/publish bus.
//
//	ch := bus.Subscribe(ctx)
//	bus.Publish(v)
//
// The channel is closed when ctx is done. Publish never blocks; a full
// or closed subscriber is skipped. PublishKeep drops the oldest queued
// value to make room. [CreateTimer] is a ctx-scoped tick
// channel (one buffered, extras dropped). [FPS] is a smoothed frame rate.
package event

import (
	"context"
	"sync"
)

// Bus fans values out to subscribers.
type Bus[T any] struct {
	mu   sync.Mutex
	subs map[chan T]struct{}
}

// New returns an empty bus.
func New[T any]() *Bus[T] {
	return &Bus[T]{subs: make(map[chan T]struct{})}
}

// Subscribe registers a listener. The channel is closed when ctx is done.
func (b *Bus[T]) Subscribe(ctx context.Context) <-chan T {
	ch := make(chan T, 16)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()
	go func() {
		<-ctx.Done()
		b.mu.Lock()
		delete(b.subs, ch)
		close(ch)
		b.mu.Unlock()
	}()
	return ch
}

// Publish sends val to every subscriber. It does not wait.
func (b *Bus[T]) Publish(val T) {
	b.publish(val, false)
}

// PublishKeep sends val to every subscriber. A full subscriber drops its
// oldest value so val is the one that stays.
func (b *Bus[T]) PublishKeep(val T) {
	b.publish(val, true)
}

func (b *Bus[T]) publish(val T, keep bool) {
	b.mu.Lock()
	subs := make([]chan T, 0, len(b.subs))
	for ch := range b.subs {
		subs = append(subs, ch)
	}
	b.mu.Unlock()
	for _, ch := range subs {
		func() {
			defer func() { recover() }()
			select {
			case ch <- val:
				return
			default:
			}
			if !keep {
				return
			}
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- val:
			default:
			}
		}()
	}
}
