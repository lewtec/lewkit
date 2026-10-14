// Package gtk asks with a GTK 4 dialog.
// Choose is a list, Prompt is a text field, and Confirm is a yes or no box.
package gtk

import (
	"context"

	"github.com/lewtec/lewkit/x/driver/internal/gtkui"
	"github.com/lewtec/lewkit/x/driver/launcher"
	gtklib "github.com/lewtec/lewkit/x/ffi/native/gtk"

	_ "github.com/lewtec/lewkit/x/driver/thread/std"
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
	var index int
	var ok bool
	err := gtkui.Run(ctx, func() error {
		var callErr error
		index, ok, callErr = gtklib.Choose(ctx, opts.Prompt, labels)
		return callErr
	})
	if err != nil || !ok || index < 0 || index >= len(opts.Items) {
		return nil, err
	}
	item := opts.Items[index]
	return &item, nil
}

func (backend) Prompt(ctx context.Context, prompt string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var text string
	var ok bool
	err := gtkui.Run(ctx, func() error {
		var callErr error
		text, ok, callErr = gtklib.Prompt(ctx, prompt)
		return callErr
	})
	if err != nil || !ok {
		return "", err
	}
	return text, nil
}

func (backend) Confirm(ctx context.Context, message string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	var ok bool
	err := gtkui.Run(ctx, func() error {
		var callErr error
		ok, callErr = gtklib.Confirm(ctx, message)
		return callErr
	})
	if err != nil {
		return false, err
	}
	return ok, nil
}

func init() {
	launcher.RegisterAsk("launcher_gtk", 80, "GTK (Choose)", "GTK (Prompt)", "GTK (Confirm)", gtkui.Require, func(context.Context) (backend, error) {
		return backend{}, nil
	})
}
