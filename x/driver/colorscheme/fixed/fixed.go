// Package fixed pins the color scheme from LEWKIT_COLORSCHEME.
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
	"github.com/lewtec/lewkit/x/driver/colorscheme"
	"github.com/lewtec/lewkit/x/event"
)

type factory struct{}

func (factory) ID() string   { return "colorscheme_fixed" }
func (factory) Name() string { return "Fixed" }
func (factory) Weight() int  { return 90 }

func (factory) CheckCompatibility(ctx context.Context) error {
	switch driver.GetEnv(ctx, "LEWKIT_COLORSCHEME") {
	case "dark", "light":
		return nil
	default:
		return fmt.Errorf("%w: LEWKIT_COLORSCHEME", driver.ErrIncompatible)
	}
}

var (
	liveMu sync.Mutex
	live   *source
)

func (factory) New(ctx context.Context) (colorscheme.Driver, error) {
	liveMu.Lock()
	defer liveMu.Unlock()
	if live == nil {
		live = &source{
			scheme: parsePinned(driver.GetEnv(ctx, "LEWKIT_COLORSCHEME")),
			bus:    event.New[colorscheme.Scheme](),
		}
	}
	return live, nil
}

func parsePinned(text string) colorscheme.Scheme {
	if text == "dark" {
		return colorscheme.Dark
	}
	return colorscheme.Light
}

type source struct {
	mu     sync.Mutex
	scheme colorscheme.Scheme
	bus    *event.Bus[colorscheme.Scheme]
}

func (src *source) Current(context.Context) (colorscheme.Scheme, error) {
	src.mu.Lock()
	defer src.mu.Unlock()
	return src.scheme, nil
}

func (src *source) Watch(ctx context.Context) (<-chan colorscheme.Scheme, error) {
	src.mu.Lock()
	current := src.scheme
	src.mu.Unlock()
	return colorscheme.Changes(ctx, current, src.bus.Subscribe(ctx)), nil
}

// Set stores scheme and publishes it when the value changed.
// Tests use it after Open. A process that did not select this driver
// ignores Set.
func Set(scheme colorscheme.Scheme) {
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
