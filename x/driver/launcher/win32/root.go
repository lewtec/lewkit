// Package win32 asks with a Win32 dialog.
// Choose is a list, Prompt is a text field, and Confirm is a yes or no box.
package win32

import (
	"context"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/launcher"
)

var (
	ErrChooseNoItems = launcher.ErrChooseNoItems
)

type backend struct{}

func (backend) Choose(ctx context.Context, opts launcher.ChooseOptions) (*launcher.Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(opts.Items) == 0 {
		return nil, ErrChooseNoItems
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
	launcher.RegisterAsk("launcher_win32", 60, "Win32 (Choose)", "Win32 (Prompt)", "Win32 (Confirm)", func(context.Context) error {
		return driver.ForGOOS(build.GOOSWindows)
	}, func(context.Context) (backend, error) {
		return backend{}, nil
	})
}
