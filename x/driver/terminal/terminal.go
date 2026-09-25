// Package terminal opens a host terminal emulator.
//
//	err := terminal.Open(ctx, terminal.Options{Title: "Shell", Command: "sh"})
//
// Import a backend (alacritty, foot, kitty).
package terminal

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
)

// Options is one terminal window.
// Command is the program to run inside it. Empty Command opens the emulator alone.
type Options struct {
	Title   string
	Command string
	Args    []string
}

// Driver starts a terminal emulator.
type Driver interface {
	Open(ctx context.Context, opts Options) error
}

// Open starts the preferred terminal emulator.
func Open(ctx context.Context, opts Options) error {
	return driver.With(ctx, func(d Driver) error {
		return d.Open(ctx, opts)
	})
}
