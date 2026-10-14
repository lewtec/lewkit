// Package cocoa asks with NSAlert. App mode leaves the dialog to the host.
package cocoa

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/launcher"
)

func available(context.Context) error {
	if err := driver.ForGOOS("darwin"); err != nil {
		return err
	}
	if driver.AppMode() {
		return fmt.Errorf("%w: app mode", driver.ErrIncompatible)
	}
	return nil
}

func open(ctx context.Context) (backend, error) {
	if err := available(ctx); err != nil {
		return backend{}, err
	}
	return backend{}, nil
}

type backend struct{}

func init() {
	launcher.RegisterAsk("launcher_cocoa", 60, "AppKit (Choose)", "AppKit (Prompt)", "AppKit (Confirm)", available, open)
}
