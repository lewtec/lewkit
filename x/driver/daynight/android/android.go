//go:build android && cgo

package android

import (
	"context"
	"fmt"
	"sync"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/event"
	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
)

func init() { driver.Register[daynight.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "daynight_android" }
func (factory) Name() string { return "Android night mode" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	n, err := androidffi.JavaVMs()
	if err != nil || n < 1 {
		return fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(ctx context.Context) (daynight.Driver, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	registerNight()
	return backend{}, nil
}

type backend struct{}

var (
	mu      sync.Mutex
	current daynight.Mode
	known   bool
	bus     = event.New[daynight.Mode]()
	once    sync.Once
)

func registerNight() {
	once.Do(func() {
		entry.HandleNight(func(mode int) {
			scheme, ok := schemeFromBit(mode)
			if !ok {
				return
			}
			publish(scheme)
		})
	})
}

func (backend) Current(ctx context.Context) (daynight.Mode, error) {
	if err := ctx.Err(); err != nil {
		return daynight.Light, err
	}
	return read()
}

func (backend) Watch(ctx context.Context) (<-chan daynight.Mode, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	scheme, err := read()
	if err != nil {
		return nil, err
	}
	return daynight.Changes(ctx, scheme, bus.Subscribe(ctx)), nil
}

func read() (daynight.Mode, error) {
	text, err := androidffi.StaticString("colorScheme")
	if err != nil {
		return daynight.Light, err
	}
	scheme, err := schemeFromHost(text)
	if err != nil {
		return daynight.Light, err
	}
	mu.Lock()
	current = scheme
	known = true
	mu.Unlock()
	return scheme, nil
}

func publish(scheme daynight.Mode) {
	mu.Lock()
	same := known && current == scheme
	current = scheme
	known = true
	mu.Unlock()
	if !same {
		bus.Publish(scheme)
	}
}
