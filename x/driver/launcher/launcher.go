// Package launcher asks the user to choose, type, or confirm.
//
//	item, err := launcher.Choose(ctx, launcher.ChooseOptions{Prompt: "Open", Items: items})
//	text, err := launcher.Prompt(ctx, "Name")
//	ok, err := launcher.Confirm(ctx, "Delete?")
//
// Import a backend (rofi, wofi, zenity, terminal). Rofi and wofi also
// launch applications and switch windows.
package launcher

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
)

// Item is one choice. Label is shown. An empty label shows Value.
// Icon is a name or path some choosers can show.
type Item struct {
	Label string
	Icon  string
	Value string
}

// ChooseOptions is one list selection.
type ChooseOptions struct {
	Prompt string
	Items  []Item
}

// Chooser selects one item from a list.
type Chooser interface {
	Choose(ctx context.Context, opts ChooseOptions) (*Item, error)
}

// Prompter reads one line of text.
type Prompter interface {
	Prompt(ctx context.Context, prompt string) (string, error)
}

// Confirmer asks a yes or no question.
type Confirmer interface {
	Confirm(ctx context.Context, message string) (bool, error)
}

// Driver is a chooser that can also launch applications and switch windows.
// Only backends that implement both register this interface.
type Driver interface {
	Chooser
	RunApp(ctx context.Context) error
	SwitchWindow(ctx context.Context) error
}

// Choose selects an item. Graphical choosers outrank the terminal.
func Choose(ctx context.Context, opts ChooseOptions) (*Item, error) {
	return driver.WithResult(ctx, func(d Chooser) (*Item, error) {
		return d.Choose(ctx, opts)
	})
}

// Prompt asks for one line of text.
func Prompt(ctx context.Context, prompt string) (string, error) {
	return driver.WithResult(ctx, func(d Prompter) (string, error) {
		return d.Prompt(ctx, prompt)
	})
}

// Confirm asks a yes or no question.
func Confirm(ctx context.Context, message string) (bool, error) {
	return driver.WithResult(ctx, func(d Confirmer) (bool, error) {
		return d.Confirm(ctx, message)
	})
}

// RunApp opens the application launcher of the active [Driver].
func RunApp(ctx context.Context) error {
	d, err := driver.Get[Driver](ctx)
	if err != nil {
		return err
	}
	return d.RunApp(ctx)
}

// SwitchWindow opens the window switcher of the active [Driver].
func SwitchWindow(ctx context.Context) error {
	d, err := driver.Get[Driver](ctx)
	if err != nil {
		return err
	}
	return d.SwitchWindow(ctx)
}
