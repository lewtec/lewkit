// Package cocoa asks with NSAlert. App mode leaves the dialog to the host.
package cocoa

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/launcher"
)

type base struct{}

func (base) ID() string  { return "launcher_cocoa" }
func (base) Weight() int { return 60 }

func (base) CheckCompatibility(ctx context.Context) error { return available(ctx) }

func available(context.Context) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("%w: not darwin", driver.ErrIncompatible)
	}
	if driver.AppMode() {
		return fmt.Errorf("%w: app mode", driver.ErrIncompatible)
	}
	return nil
}

type chooserFactory struct{ base }

func (chooserFactory) Name() string { return "AppKit (Choose)" }

func (chooserFactory) New(ctx context.Context) (launcher.Chooser, error) { return open(ctx) }

type prompterFactory struct{ base }

func (prompterFactory) Name() string { return "AppKit (Prompt)" }

func (prompterFactory) New(ctx context.Context) (launcher.Prompter, error) { return open(ctx) }

type confirmerFactory struct{ base }

func (confirmerFactory) Name() string { return "AppKit (Confirm)" }

func (confirmerFactory) New(ctx context.Context) (launcher.Confirmer, error) { return open(ctx) }

func open(ctx context.Context) (backend, error) {
	if err := available(ctx); err != nil {
		return backend{}, err
	}
	return backend{}, nil
}

type backend struct{}

func init() {
	driver.Register[launcher.Chooser](chooserFactory{})
	driver.Register[launcher.Prompter](prompterFactory{})
	driver.Register[launcher.Confirmer](confirmerFactory{})
}

var (
	_ driver.DriverFactory[launcher.Chooser]   = chooserFactory{}
	_ driver.DriverFactory[launcher.Prompter]  = prompterFactory{}
	_ driver.DriverFactory[launcher.Confirmer] = confirmerFactory{}
	_ driver.Weighter                          = chooserFactory{}
	_ driver.Weighter                          = prompterFactory{}
	_ driver.Weighter                          = confirmerFactory{}
)
