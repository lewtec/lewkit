// Package power locks and stops the session.
//
//	err := power.Lock(ctx)
//	err = power.Wake(ctx, "aa:bb:cc:dd:ee:ff")
//
// Import [github.com/lewtec/lewkit/x/driver/power/systemd]. The backend
// calls loginctl and systemctl.
package power

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
)

// Driver runs session and machine power actions.
type Driver interface {
	Lock(ctx context.Context) error
	Logout(ctx context.Context) error
	Suspend(ctx context.Context) error
	Hibernate(ctx context.Context) error
	Reboot(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

// Lock locks the session.
func Lock(ctx context.Context) error {
	return driver.With(ctx, func(d Driver) error { return d.Lock(ctx) })
}

// Logout ends the session.
func Logout(ctx context.Context) error {
	return driver.With(ctx, func(d Driver) error { return d.Logout(ctx) })
}

// Suspend suspends the machine.
func Suspend(ctx context.Context) error {
	return driver.With(ctx, func(d Driver) error { return d.Suspend(ctx) })
}

// Hibernate hibernates the machine.
func Hibernate(ctx context.Context) error {
	return driver.With(ctx, func(d Driver) error { return d.Hibernate(ctx) })
}

// Reboot reboots the machine.
func Reboot(ctx context.Context) error {
	return driver.With(ctx, func(d Driver) error { return d.Reboot(ctx) })
}

// Shutdown powers the machine off.
func Shutdown(ctx context.Context) error {
	return driver.With(ctx, func(d Driver) error { return d.Shutdown(ctx) })
}
