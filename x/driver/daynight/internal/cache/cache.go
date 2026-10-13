// Package cache stores one day or night mode and publishes a change.
package cache

import (
	"context"
	"sync"

	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/event"
)

// Cache is the mode last handed to Watch.
type Cache struct {
	mu     sync.Mutex
	scheme daynight.Mode
	bus    *event.Bus[daynight.Mode]
}

// New starts at scheme.
func New(scheme daynight.Mode) *Cache {
	return &Cache{scheme: scheme, bus: event.New[daynight.Mode]()}
}

// Current returns the stored mode.
func (c *Cache) Current(context.Context) (daynight.Mode, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.scheme, nil
}

// Watch sends the stored mode, then each later change.
func (c *Cache) Watch(ctx context.Context) (<-chan daynight.Mode, error) {
	c.mu.Lock()
	current := c.scheme
	c.mu.Unlock()
	return daynight.Changes(ctx, current, c.bus.Subscribe(ctx)), nil
}

// Publish stores scheme and sends it when the value changed.
func (c *Cache) Publish(scheme daynight.Mode) {
	c.mu.Lock()
	if c.scheme == scheme {
		c.mu.Unlock()
		return
	}
	c.scheme = scheme
	c.mu.Unlock()
	c.bus.Publish(scheme)
}
