package android

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/volume"
	ffiandroid "github.com/lewtec/lewkit/x/ffi/android"
)

func open(ctx context.Context) (*ffiandroid.Client, error) {
	client, err := ffiandroid.ForAndroid(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", driver.ErrIncompatible, err)
	}
	return client, nil
}

func init() { driver.Register[volume.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "volume_android" }
func (factory) Name() string { return "Android volume" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(ctx context.Context) error {
	client, err := open(ctx)
	if err != nil {
		return err
	}
	if _, _, _, err := client.StreamVolume(ctx, musicStream); err != nil {
		return fmt.Errorf("%w: %w", driver.ErrIncompatible, err)
	}
	return nil
}

func (factory) New(context.Context) (volume.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) SetVolume(ctx context.Context, level float64) error {
	client, err := open(ctx)
	if err != nil {
		return err
	}
	_, max, _, err := client.StreamVolume(ctx, musicStream)
	if err != nil {
		return err
	}
	return client.SetStreamVolume(ctx, musicStream, index(level, max))
}

func (backend) GetVolume(ctx context.Context) (float64, error) {
	cur, max, err := music(ctx)
	if err != nil {
		return 0, err
	}
	return fraction(cur, max), nil
}

func (backend) ToggleMute(ctx context.Context) error {
	client, err := open(ctx)
	if err != nil {
		return err
	}
	return client.ToggleStreamMute(ctx, musicStream)
}

func (backend) GetMute(ctx context.Context) (bool, error) {
	_, _, muted, err := volumes(ctx)
	return muted, err
}

func (backend) SinkName(context.Context) (string, error) { return "music", nil }

func music(ctx context.Context) (int32, int32, error) {
	cur, max, _, err := volumes(ctx)
	return cur, max, err
}

func volumes(ctx context.Context) (int32, int32, bool, error) {
	client, err := open(ctx)
	if err != nil {
		return 0, 0, false, err
	}
	return client.StreamVolume(ctx, musicStream)
}
