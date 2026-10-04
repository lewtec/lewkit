// Package ios reads light or dark from the packaged iOS host.
// The host writes daynight.txt and updates it when the appearance changes.
// LEWKIT_DAYNIGHT still wins: that driver has the higher weight.
package ios

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/iosbox"
)

type factory struct{}

func (factory) ID() string   { return "daynight_ios" }
func (factory) Name() string { return "iOS appearance" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	if runtime.GOOS != "ios" {
		return fmt.Errorf("%w: not ios", driver.ErrIncompatible)
	}
	if !iosbox.Available() {
		return fmt.Errorf("%w: no ios host", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (daynight.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Current(ctx context.Context) (daynight.Mode, error) {
	return read(ctx)
}

func (backend) Watch(ctx context.Context) (<-chan daynight.Mode, error) {
	mode, err := read(ctx)
	if err != nil {
		return nil, err
	}
	next := make(chan daynight.Mode)
	go poll(ctx, mode, next)
	return daynight.Changes(ctx, mode, next), nil
}

var watchEvery = 400 * time.Millisecond

func poll(ctx context.Context, last daynight.Mode, next chan<- daynight.Mode) {
	defer close(next)
	tick := time.NewTicker(watchEvery)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			mode, err := read(ctx)
			if err != nil || mode == last {
				continue
			}
			last = mode
			select {
			case next <- mode:
			case <-ctx.Done():
				return
			}
		}
	}
}

func read(ctx context.Context) (daynight.Mode, error) {
	if err := ctx.Err(); err != nil {
		return daynight.Light, err
	}
	text, err := iosbox.ReadMode(ctx)
	if err != nil {
		return daynight.Light, err
	}
	if text == "dark" {
		return daynight.Dark, nil
	}
	return daynight.Light, nil
}

func init() { driver.Register[daynight.Driver](factory{}) }

var (
	_ driver.DriverFactory[daynight.Driver] = factory{}
	_ driver.Weighter                       = factory{}
)
