// Package android asks with an alert dialog on the foreground activity.
package android

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/androidask"
	"github.com/lewtec/lewkit/x/driver/launcher"
)

type base struct{}

func (base) ID() string  { return "launcher_android" }
func (base) Weight() int { return 80 }

func (base) CheckCompatibility(ctx context.Context) error { return androidask.Platform(ctx) }

type chooserFactory struct{ base }

func (chooserFactory) Name() string { return "Android (Choose)" }

func (chooserFactory) New(ctx context.Context) (launcher.Chooser, error) { return open(ctx) }

type prompterFactory struct{ base }

func (prompterFactory) Name() string { return "Android (Prompt)" }

func (prompterFactory) New(ctx context.Context) (launcher.Prompter, error) { return open(ctx) }

type confirmerFactory struct{ base }

func (confirmerFactory) Name() string { return "Android (Confirm)" }

func (confirmerFactory) New(ctx context.Context) (launcher.Confirmer, error) { return open(ctx) }

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
)
