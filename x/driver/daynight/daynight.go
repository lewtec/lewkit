// Package daynight reports whether the system theme is light or dark.
//
//	mode, err := daynight.Current(ctx)
//	changes, err := daynight.Watch(ctx)
//
// Import [github.com/lewtec/lewkit/x/driver/prelude] or one implementation
// (portal, cocoa, win32). LEWKIT_DAYNIGHT=light or dark pins a mode
// for tests and wins over the host.
//
// Linux reads org.freedesktop.appearance color-scheme from the desktop
// portal and the SettingChanged signal. macOS reads AppleInterfaceStyle
// and AppleInterfaceThemeChangedNotification. Windows reads
// AppsUseLightTheme and the registry change notification.
package daynight

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
)

// Mode is the system light or dark preference.
type Mode uint8

const (
	// Light is the light preference.
	Light Mode = iota
	// Dark is the dark preference.
	Dark
)

func (scheme Mode) String() string {
	if scheme == Dark {
		return "dark"
	}
	return "light"
}

// Driver reads the system day or night mode.
type Driver interface {
	Current(ctx context.Context) (Mode, error)
	// Watch sends the current mode, then each later change.
	// The channel closes when ctx is done.
	Watch(ctx context.Context) (<-chan Mode, error)
}

// Current returns the active mode.
func Current(ctx context.Context) (Mode, error) {
	return driver.WithResult(ctx, func(source Driver) (Mode, error) {
		return source.Current(ctx)
	})
}

// Watch returns the current mode and every later change.
func Watch(ctx context.Context) (<-chan Mode, error) {
	return driver.WithResult(ctx, func(source Driver) (<-chan Mode, error) {
		return source.Watch(ctx)
	})
}
