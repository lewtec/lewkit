// Package notification posts a local notification.
//
//	err := notification.Notify(ctx, notification.Notification{
//		Title: "lewkit", Message: "done",
//	})
//
// Import the dbus or notify-send backend. DBus speaks
// org.freedesktop.Notifications. notify-send is the command fallback.
package notification

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
)

// Notification is one local alert.
// ID replaces an existing notification when the backend supports it.
// Urgency is "low", "normal", or "critical".
// Progress is 0..1 and is sent only when HasProgress is set.
type Notification struct {
	ID          uint32
	Title       string
	Message     string
	Urgency     string
	Icon        string
	Progress    float64
	HasProgress bool
}

// StatusID replaces the previous volume, brightness, or media status alert.
const StatusID uint32 = 100

// Driver posts one notification.
type Driver interface {
	Notify(ctx context.Context, n Notification) error
}

// Notify posts n with the selected backend.
func Notify(ctx context.Context, n Notification) error {
	return driver.With(ctx, func(d Driver) error {
		return d.Notify(ctx, n)
	})
}
