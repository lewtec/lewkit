// Package brightness reads and sets the backlight.
//
//	dev, err := brightness.Status(ctx)
//	err = brightness.Increase(ctx)
//	note := brightness.StatusNotification(dev.Name, dev.Brightness)
//
// Import [github.com/lewtec/lewkit/x/driver/brightness/brightnessctl].
// Increase and Decrease step by 0.05 and stay inside 0..1.
package brightness

import (
	"context"
	"errors"

	"github.com/lewtec/lewkit/x/driver"
)

const step = 0.05

// ErrDeviceNotFound means brightnessctl returned no device.
var ErrDeviceNotFound = errors.New("brightness device not found")

// Device is one backlight and its level from 0 to 1.
type Device struct {
	Name       string
	Brightness float64
}

// Driver reads and sets the backlight.
type Driver interface {
	SetBrightness(ctx context.Context, brightness float64) error
	Status(ctx context.Context) (*Device, error)
}

// SetBrightness sets the backlight. brightness is a fraction from 0 to 1.
func SetBrightness(ctx context.Context, level float64) error {
	return driver.With(ctx, func(d Driver) error { return d.SetBrightness(ctx, level) })
}

// Status returns the first backlight brightnessctl reports.
func Status(ctx context.Context) (*Device, error) {
	return driver.WithResult(ctx, func(d Driver) (*Device, error) { return d.Status(ctx) })
}

// Increase raises the backlight by 0.05, clamped to 0..1.
func Increase(ctx context.Context) error { return adjust(ctx, step) }

// Decrease lowers the backlight by 0.05, clamped to 0..1.
func Decrease(ctx context.Context) error { return adjust(ctx, -step) }

func adjust(ctx context.Context, delta float64) error {
	return driver.With(ctx, func(d Driver) error {
		status, err := d.Status(ctx)
		if err != nil {
			return err
		}
		return d.SetBrightness(ctx, clamp01(status.Brightness+delta))
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
