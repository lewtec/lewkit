// Package messagebox shows one message and waits until the user dismisses it.
//
//	err := messagebox.Show(ctx, "Hello", "Saved.")
//
// Import a backend (android, host, cocoa, win32, zenity). App mode uses
// the packaged host or the Android dialog. A terminal is not required.
// [Show] is a critical alert. [ShowNotice] selects informational, warning,
// or critical. An empty style is informational.
package messagebox

import (
	"context"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
)

const (
	// StyleInformational is a plain message.
	StyleInformational = "informational"
	// StyleWarning is a warning.
	StyleWarning = "warning"
	// StyleCritical is an error.
	StyleCritical = "critical"
)

// Notice is one message. Style is informational, warning, or critical.
// An empty style is informational.
type Notice struct {
	Title   string
	Message string
	Style   string
}

// Driver shows one message.
type Driver interface {
	Show(ctx context.Context, n Notice) error
}

// Show presents message under title as a critical alert.
// An empty title is "Error".
func Show(ctx context.Context, title, message string) error {
	return ShowNotice(ctx, Notice{Title: title, Message: message, Style: StyleCritical})
}

// ShowNotice presents n. An empty title is "Error".
func ShowNotice(ctx context.Context, n Notice) error {
	if strings.TrimSpace(n.Title) == "" {
		n.Title = "Error"
	}
	n.Style = NormalizeStyle(n.Style)
	return driver.With(ctx, func(d Driver) error {
		return d.Show(ctx, n)
	})
}

// NormalizeStyle returns informational, warning, or critical.
func NormalizeStyle(style string) string {
	switch strings.ToLower(strings.TrimSpace(style)) {
	case StyleWarning, "warn":
		return StyleWarning
	case StyleCritical, "error":
		return StyleCritical
	default:
		return StyleInformational
	}
}
