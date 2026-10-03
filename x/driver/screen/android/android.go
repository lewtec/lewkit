package android

import (
	"context"
	"errors"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	host "github.com/lewtec/lewkit/x/driver/android"
	"github.com/lewtec/lewkit/x/driver/screen"
)

func init() { driver.Register[screen.Driver](factory{}) }

var errNoLayout = errors.New("android display layout reset is unavailable")

type factory struct{}

func (factory) ID() string   { return "screen_android" }
func (factory) Name() string { return "Android screen" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(ctx context.Context) error {
	client, err := host.Open(ctx)
	if err != nil {
		return err
	}
	if _, err := client.Interactive(ctx); err != nil {
		return fmt.Errorf("%w: %w", driver.ErrIncompatible, err)
	}
	return nil
}

func (factory) New(context.Context) (screen.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) SetDPMS(ctx context.Context, on bool) error {
	client, err := host.Open(ctx)
	if err != nil {
		return err
	}
	if on {
		return client.Wake(ctx)
	}
	return client.Sleep(ctx)
}

func (backend) IsDPMSOn(ctx context.Context) (bool, error) {
	client, err := host.Open(ctx)
	if err != nil {
		return false, err
	}
	return client.Interactive(ctx)
}

func (backend) Reset(context.Context) error { return errNoLayout }
