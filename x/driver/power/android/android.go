package android

import (
	"context"
	"errors"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/power"
	ffiandroid "github.com/lewtec/lewkit/x/ffi/android"
)

func open(ctx context.Context) (*ffiandroid.Client, error) {
	client, err := ffiandroid.ForAndroid(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", driver.ErrIncompatible, err)
	}
	return client, nil
}

func init() { driver.Register[power.Driver](factory{}) }

var (
	errLogout    = errors.New("android session logout is unavailable")
	errHibernate = errors.New("android hibernate is unavailable")
)

type factory struct{}

func (factory) ID() string   { return "power_android" }
func (factory) Name() string { return "Android power" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(ctx context.Context) error {
	client, err := open(ctx)
	if err != nil {
		return err
	}
	if _, err := client.Interactive(ctx); err != nil {
		return fmt.Errorf("%w: %w", driver.ErrIncompatible, err)
	}
	return nil
}

func (factory) New(context.Context) (power.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Lock(ctx context.Context) error {
	client, err := open(ctx)
	if err != nil {
		return err
	}
	return client.Lock(ctx)
}

func (backend) Logout(context.Context) error { return errLogout }

func (backend) Suspend(ctx context.Context) error {
	client, err := open(ctx)
	if err != nil {
		return err
	}
	return client.ForceSuspend(ctx)
}

func (backend) Hibernate(context.Context) error { return errHibernate }

func (backend) Reboot(ctx context.Context) error {
	client, err := open(ctx)
	if err != nil {
		return err
	}
	return client.Reboot(ctx)
}

func (backend) Shutdown(ctx context.Context) error {
	client, err := open(ctx)
	if err != nil {
		return err
	}
	return client.Shutdown(ctx)
}
