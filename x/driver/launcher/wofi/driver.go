package wofi

import (
	"context"
	"os/exec"

	"github.com/lewtec/lewkit/x/driver/launcher"
)

type chooserFactory struct{}

func (chooserFactory) ID() string   { return "wofi" }
func (chooserFactory) Name() string { return "Wofi" }
func (chooserFactory) Weight() int  { return 50 }

func (chooserFactory) CheckCompatibility(ctx context.Context) error {
	return launcher.RequireEnvBinary(ctx, "WAYLAND_DISPLAY", "wofi")
}

func (chooserFactory) New(context.Context) (launcher.Chooser, error) {
	return backend{}, nil
}

type driverFactory struct{}

func (driverFactory) ID() string   { return "wofi" }
func (driverFactory) Name() string { return "Wofi" }
func (driverFactory) Weight() int  { return 50 }

func (driverFactory) CheckCompatibility(ctx context.Context) error {
	return launcher.RequireEnvBinary(ctx, "WAYLAND_DISPLAY", "wofi")
}

func (driverFactory) New(context.Context) (launcher.Driver, error) {
	return backend{}, nil
}

type backend struct{}

func (backend) Choose(ctx context.Context, opts launcher.ChooseOptions) (*launcher.Item, error) {
	return launcher.ChooseViaCmd(ctx, opts, "wofi", false, "--dmenu", "-p", opts.Prompt)
}

func (backend) RunApp(ctx context.Context) error {
	return exec.CommandContext(ctx, "wofi", "--show", "drun").Run()
}

func (backend) SwitchWindow(ctx context.Context) error {
	return backend{}.RunApp(ctx)
}
