// Package screen sets display power and layout.
//
//	err := screen.SetDPMS(ctx, false)
//	on, err := screen.IsDPMSOn(ctx)
//
// Import sway and x11. Sway is preferred when both sessions are present.
// ToggleDPMS flips the current DPMS state. Lock stays on the power driver.
package screen

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
)

// Driver controls display power and output layout.
type Driver interface {
	SetDPMS(ctx context.Context, on bool) error
	IsDPMSOn(ctx context.Context) (bool, error)
	Reset(ctx context.Context) error
}

// SetDPMS turns display power on or off.
func SetDPMS(ctx context.Context, on bool) error {
	return driver.With(ctx, func(d Driver) error { return d.SetDPMS(ctx, on) })
}

// IsDPMSOn reports whether the display is on.
func IsDPMSOn(ctx context.Context) (bool, error) {
	return driver.WithResult(ctx, func(d Driver) (bool, error) { return d.IsDPMSOn(ctx) })
}

// Reset restores the output layout for this host.
func Reset(ctx context.Context) error {
	return driver.With(ctx, func(d Driver) error { return d.Reset(ctx) })
}

// ToggleDPMS flips DPMS.
func ToggleDPMS(ctx context.Context) error {
	return driver.With(ctx, func(d Driver) error {
		on, err := d.IsDPMSOn(ctx)
		if err != nil {
			return err
		}
		return d.SetDPMS(ctx, !on)
	})
}
