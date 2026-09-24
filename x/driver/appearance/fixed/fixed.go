// Package fixed pins the color scheme from LEWKIT_APPEARANCE.
//
// The factory is incompatible unless that variable is light or dark, so a
// normal process keeps the host driver. Set publishes a later change to
// every Watch on this driver.
package fixed

import (
	"context"
	"fmt"
	"sync"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/appearance"
	"github.com/lewtec/lewkit/x/event"
)

type factory struct{}

func (factory) ID() string   { return "appearance_fixed" }
func (factory) Name() string { return "Fixed" }
func (factory) Weight() int  { return 90 }

func (factory) CheckCompatibility(ctx context.Context) error {
	switch driver.GetEnv(ctx, "LEWKIT_APPEARANCE") {
	case "dark", "light":
		return nil
	default:
		return fmt.Errorf("%w: LEWKIT_APPEARANCE", driver.ErrIncompatible)
	}
}

var (
	liveMu sync.Mutex
	live   *source
)

func (factory) New(ctx context.Context) (appearance.Driver, error) {
	liveMu.Lock()
	defer liveMu.Unlock()
	if live == nil {
		live = &source{
			scheme: parsePinned(driver.GetEnv(ctx, "LEWKIT_APPEARANCE")),
			bus:    event.New[appearance.Scheme](),
		}
	}
	return live, nil
}

func parsePinned(text string) appearance.Scheme {
	if text == "dark" {
		return appearance.Dark
	}
	return appearance.Light
}

type source struct {
	mu     sync.Mutex
	scheme appearance.Scheme
	bus    *event.Bus[appearance.Scheme]
}

func (src *source) Current(context.Context) (appearance.Scheme, error) {
	src.mu.Lock()
	defer src.mu.Unlock()
	return src.scheme, nil
}

func (src *source) Watch(ctx context.Context) (<-chan appearance.Scheme, error) {
	src.mu.Lock()
	current := src.scheme
	src.mu.Unlock()
	return appearance.Changes(ctx, current, src.bus.Subscribe(ctx)), nil
}

// Set stores scheme and publishes it when the value changed.
// Tests use it after Open. A process that did not select this driver
// ignores Set.
func Set(scheme appearance.Scheme) {
	liveMu.Lock()
	src := live
	liveMu.Unlock()
	if src == nil {
		return
	}
	src.mu.Lock()
	if src.scheme == scheme {
		src.mu.Unlock()
		return
	}
	src.scheme = scheme
	src.mu.Unlock()
	src.bus.Publish(scheme)
}
