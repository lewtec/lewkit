// Package appearance reports the system color scheme and each change.
//
//	scheme, err := appearance.Current(ctx)
//	changes, err := appearance.Watch(ctx)
//
// Import [github.com/lewtec/lewkit/x/driver/prelude] or one implementation
// (portal, cocoa, win32). LEWKIT_APPEARANCE=light or dark pins a scheme
// for tests and wins over the host.
//
// Linux reads org.freedesktop.appearance color-scheme from the desktop
// portal and the SettingChanged signal. macOS reads AppleInterfaceStyle
// and AppleInterfaceThemeChangedNotification. Windows reads
// AppsUseLightTheme and the registry change notification.
package appearance

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
)

// Scheme is the system light or dark preference.
type Scheme uint8

const (
	// Light is the light preference.
	Light Scheme = iota
	// Dark is the dark preference.
	Dark
)

func (scheme Scheme) String() string {
	if scheme == Dark {
		return "dark"
	}
	return "light"
}

// Driver reads the system color scheme.
type Driver interface {
	Current(ctx context.Context) (Scheme, error)
	// Watch sends the current scheme, then each later change.
	// The channel closes when ctx is done.
	Watch(ctx context.Context) (<-chan Scheme, error)
}

// Current returns the active scheme.
func Current(ctx context.Context) (Scheme, error) {
	return driver.WithResult(ctx, func(source Driver) (Scheme, error) {
		return source.Current(ctx)
	})
}

// Watch returns the current scheme and every later change.
func Watch(ctx context.Context) (<-chan Scheme, error) {
	return driver.WithResult(ctx, func(source Driver) (<-chan Scheme, error) {
		return source.Watch(ctx)
	})
}
