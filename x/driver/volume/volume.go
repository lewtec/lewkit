// Package volume sets the default audio sink.
//
//	err := volume.SetVolume(ctx, 0.4)
//	err = volume.Increase(ctx)
//
// Import [github.com/lewtec/lewkit/x/driver/volume/pulse] to register the
// pactl backend. Increase and Decrease step by 0.05 and stay inside 0..1.
package volume

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
)

const step = 0.05

// Driver controls the default sink.
type Driver interface {
	SetVolume(ctx context.Context, volume float64) error
	GetVolume(ctx context.Context) (float64, error)
	ToggleMute(ctx context.Context) error
	GetMute(ctx context.Context) (bool, error)
	SinkName(ctx context.Context) (string, error)
}

// SetVolume sets the default sink volume. volume is a fraction from 0 to 1.
func SetVolume(ctx context.Context, level float64) error {
	return driver.With(ctx, func(d Driver) error { return d.SetVolume(ctx, level) })
}

// GetVolume returns the default sink volume as a fraction.
func GetVolume(ctx context.Context) (float64, error) {
	return driver.WithResult(ctx, func(d Driver) (float64, error) { return d.GetVolume(ctx) })
}

// ToggleMute flips mute on the default sink.
func ToggleMute(ctx context.Context) error {
	return driver.With(ctx, func(d Driver) error { return d.ToggleMute(ctx) })
}

// GetMute reports whether the default sink is muted.
func GetMute(ctx context.Context) (bool, error) {
	return driver.WithResult(ctx, func(d Driver) (bool, error) { return d.GetMute(ctx) })
}

// SinkName returns the default sink name.
func SinkName(ctx context.Context) (string, error) {
	return driver.WithResult(ctx, func(d Driver) (string, error) { return d.SinkName(ctx) })
}

// Increase raises the volume by 0.05, clamped to 0..1.
func Increase(ctx context.Context) error { return adjust(ctx, step) }

// Decrease lowers the volume by 0.05, clamped to 0..1.
func Decrease(ctx context.Context) error { return adjust(ctx, -step) }

func adjust(ctx context.Context, delta float64) error {
	return driver.With(ctx, func(d Driver) error {
		level, err := d.GetVolume(ctx)
		if err != nil {
			return err
		}
		return d.SetVolume(ctx, clamp01(level+delta))
	})
}

func clamp01(v float64) float64 {
	if v > 1 {
		return 1
	}
	if v < 0 {
		return 0
	}
	return v
}
