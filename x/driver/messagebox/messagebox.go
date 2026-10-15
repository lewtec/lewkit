// Package messagebox shows one message and waits until the user dismisses it.
//
//	err := messagebox.Show(ctx, "Hello", "Saved.")
//
// Import a backend (android, host, cocoa, gtk, win32, zenity). App mode uses
// the packaged host or the Android dialog. A terminal is not required.
// [Show] is a critical alert. [ShowNotice] selects informational, warning,
// or critical. An empty style is informational.
package messagebox

import (
	"context"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
)

// Style is informational, warning, or critical.
//
//lewkit:enum fold
type Style string

const (
	// StyleInformational is a plain message.
	StyleInformational Style = "informational"
	// StyleWarning is a warning.
	StyleWarning Style = "warning" // warn
	// StyleCritical is an error.
	StyleCritical Style = "critical" // error
)

// Notice is one message. Style is informational, warning, or critical.
// An empty style is informational.
type Notice struct {
	Title   string
	Message string
	Style   Style
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
	n.Style = NormalizeStyle(string(n.Style))
	return driver.With(ctx, func(d Driver) error {
		return d.Show(ctx, n)
	})
}

// NormalizeStyle returns informational, warning, or critical.
// Unknown text, including an empty string, is informational.
func NormalizeStyle(style string) Style {
	parsed, err := ParseStyle(style)
	if err != nil {
		return StyleInformational
	}
	return parsed
}
