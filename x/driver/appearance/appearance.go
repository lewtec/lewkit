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
// 0 from the portal means no preference and is reported as light.
package appearance

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
)

// Scheme is the system light or dark preference.
type Scheme uint8

const (
	// Light is the default preference, including a portal value of no preference.
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

// Parse accepts "dark", "light", "1", and "0". Anything else is light.
func Parse(text string) Scheme {
	switch text {
	case "dark", "Dark", "1", "prefer-dark":
		return Dark
	default:
		return Light
	}
}

// FromPortal maps the desktop-portal color-scheme value.
// 1 is prefer-dark. 0 (no preference) and 2 (prefer-light) are light.
func FromPortal(value uint32) Scheme {
	if value == 1 {
		return Dark
	}
	return Light
}

// FromAppleInterfaceStyle maps AppleInterfaceStyle. "Dark" is dark.
func FromAppleInterfaceStyle(value string) Scheme {
	if value == "Dark" {
		return Dark
	}
	return Light
}

// FromAppsUseLightTheme maps the Windows AppsUseLightTheme DWORD.
// 0 is dark. A missing value is passed as 1 (light).
func FromAppsUseLightTheme(value uint32) Scheme {
	if value == 0 {
		return Dark
	}
	return Light
}
