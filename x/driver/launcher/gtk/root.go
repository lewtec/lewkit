// Package gtk asks with a GTK 4 dialog.
// Choose is a list, Prompt is a text field, and Confirm is a yes or no box.
package gtk

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/launcher"
	"github.com/lewtec/lewkit/x/driver/thread"
	gtklib "github.com/lewtec/lewkit/x/ffi/native/gtk"

	_ "github.com/lewtec/lewkit/x/driver/thread/std"
)

type base struct{}

func (base) ID() string  { return "launcher_gtk" }
func (base) Weight() int { return 80 }

func (base) CheckCompatibility(ctx context.Context) error {
	if err := driver.ForGOOS("linux"); err != nil {
		return err
	}
	if driver.GetEnv(ctx, "DISPLAY") == "" && driver.GetEnv(ctx, "WAYLAND_DISPLAY") == "" {
		return fmt.Errorf("%w: neither DISPLAY nor WAYLAND_DISPLAY set", driver.ErrIncompatible)
	}
	if err := gtklib.Available(ctx); err != nil {
		return fmt.Errorf("%w: %v", driver.ErrIncompatible, err)
	}
	return nil
}

type chooserFactory struct{ base }

func (chooserFactory) Name() string { return "GTK (Choose)" }

func (chooserFactory) New(context.Context) (launcher.Chooser, error) { return backend{}, nil }

type prompterFactory struct{ base }

func (prompterFactory) Name() string { return "GTK (Prompt)" }

func (prompterFactory) New(context.Context) (launcher.Prompter, error) { return backend{}, nil }

type confirmerFactory struct{ base }

func (confirmerFactory) Name() string { return "GTK (Confirm)" }

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
	var index int
	var ok bool
	err := present(ctx, func() error {
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
	err := present(ctx, func() error {
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
	err := present(ctx, func() error {
		var callErr error
		ok, callErr = gtklib.Confirm(ctx, message)
		return callErr
	})
	if err != nil {
		return false, err
	}
	return ok, nil
}

// present runs fn on the thread that owns GTK.
// The first dialog claims the UI thread when one is bound.
func present(ctx context.Context, fn func() error) error {
	if gtklib.Owned() && !gtklib.OnOwner() {
		return fn()
	}
	if bound, on := uiThread(); bound && !on {
		done := make(chan error, 1)
		thread.Go(func() { done <- fn() })
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-done:
			return err
		}
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	return fn()
}

func uiThread() (bound, on bool) {
	defer func() { _ = recover() }()
	if !thread.Bound() {
		return false, false
	}
	return true, thread.On()
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
