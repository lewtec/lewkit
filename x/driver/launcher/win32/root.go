// Package win32 asks with a Win32 dialog.
// Choose is a list, Prompt is a text field, and Confirm is a yes or no box.
package win32

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/launcher"
)

type base struct{}

func (base) ID() string  { return "launcher_win32" }
func (base) Weight() int { return 60 }

func (base) CheckCompatibility(context.Context) error {
	return driver.ForGOOS("windows")
}

type chooserFactory struct{ base }

func (chooserFactory) Name() string { return "Win32 (Choose)" }

func (chooserFactory) New(context.Context) (launcher.Chooser, error) { return backend{}, nil }

type prompterFactory struct{ base }

func (prompterFactory) Name() string { return "Win32 (Prompt)" }

func (prompterFactory) New(context.Context) (launcher.Prompter, error) { return backend{}, nil }

type confirmerFactory struct{ base }

func (confirmerFactory) Name() string { return "Win32 (Confirm)" }

func (confirmerFactory) New(context.Context) (launcher.Confirmer, error) { return backend{}, nil }

type backend struct{}

func (backend) Choose(ctx context.Context, opts launcher.ChooseOptions) (*launcher.Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(opts.Items) == 0 {
		return nil, fmt.Errorf("choose: no items")
	}
	labels := make([]string, len(opts.Items))
	for i, item := range opts.Items {
		labels[i] = launcher.ItemLabel(item)
	}
	text, ok, err := ask(ctx, "choose", opts.Prompt, "", labels)
	if err != nil || !ok {
		return nil, err
	}
	return launcher.MatchSelected(opts.Items, text), nil
}

func (backend) Prompt(ctx context.Context, prompt string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	text, ok, err := ask(ctx, "prompt", prompt, "", nil)
	if err != nil || !ok {
		return "", err
	}
	return text, nil
}

func (backend) Confirm(ctx context.Context, message string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	_, ok, err := ask(ctx, "confirm", message, "", nil)
	if err != nil {
		return false, err
	}
	return ok, nil
}

func init() {
	driver.Register[launcher.Chooser](chooserFactory{})
	driver.Register[launcher.Prompter](prompterFactory{})
	driver.Register[launcher.Confirmer](confirmerFactory{})
}

var (
	_ driver.DriverFactory[launcher.Chooser]   = chooserFactory{}
	_ driver.DriverFactory[launcher.Prompter]  = prompterFactory{}
	_ driver.DriverFactory[launcher.Confirmer] = confirmerFactory{}
	_ driver.Weighter                          = base{}
)
