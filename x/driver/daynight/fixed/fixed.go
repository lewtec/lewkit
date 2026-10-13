// Package fixed pins day or night from LEWKIT_DAYNIGHT.
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
	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/daynight/internal/cache"
)

type factory struct{}

func (factory) ID() string   { return "daynight_fixed" }
func (factory) Name() string { return "Fixed" }
func (factory) Weight() int  { return 90 }

func (factory) CheckCompatibility(ctx context.Context) error {
	switch driver.GetEnv(ctx, "LEWKIT_DAYNIGHT") {
	case "dark", "light":
		return nil
	default:
		return fmt.Errorf("%w: LEWKIT_DAYNIGHT", driver.ErrIncompatible)
	}
}

var (
	liveMu sync.Mutex
	live   *cache.Cache
)

func (factory) New(ctx context.Context) (daynight.Driver, error) {
	liveMu.Lock()
	defer liveMu.Unlock()
	if live == nil {
		live = cache.New(parsePinned(driver.GetEnv(ctx, "LEWKIT_DAYNIGHT")))
	}
	return live, nil
}

func parsePinned(text string) daynight.Mode {
	if text == "dark" {
		return daynight.Dark
	}
	return daynight.Light
}

// Set stores scheme and publishes it when the value changed.
// Tests use it after Open. A process that did not select this driver
// ignores Set.
func Set(scheme daynight.Mode) {
	liveMu.Lock()
	src := live
	liveMu.Unlock()
	if src == nil {
		return
	}
	src.Publish(scheme)
}
