package android

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/daynight"
	ffiandroid "github.com/lewtec/lewkit/x/ffi/android"
)

func init() { driver.Register[daynight.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "daynight_android" }
func (factory) Name() string { return "Android night mode" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(ctx context.Context) error {
	_, err := current(ctx)
	if err == nil || errors.Is(err, driver.ErrIncompatible) {
		return err
	}
	return fmt.Errorf("%w: %w", driver.ErrIncompatible, err)
}

func (factory) New(context.Context) (daynight.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Current(ctx context.Context) (daynight.Mode, error) { return current(ctx) }

func (backend) Watch(ctx context.Context) (<-chan daynight.Mode, error) {
	mode, err := current(ctx)
	if err != nil {
		return nil, err
	}
	next := make(chan daynight.Mode, 1)
	go poll(ctx, mode, next)
	return daynight.Changes(ctx, mode, next), nil
}

func poll(ctx context.Context, last daynight.Mode, next chan<- daynight.Mode) {
	defer close(next)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mode, err := current(ctx)
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

func current(ctx context.Context) (daynight.Mode, error) {
	client, err := open(ctx)
	if err != nil {
		return daynight.Light, err
	}
	dark, err := client.Night(ctx)
	if err != nil {
		return daynight.Light, err
	}
	if dark {
		return daynight.Dark, nil
	}
	return daynight.Light, nil
}

func open(ctx context.Context) (*ffiandroid.Client, error) {
	client, err := ffiandroid.ForAndroid(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", driver.ErrIncompatible, err)
	}
	return client, nil
}
