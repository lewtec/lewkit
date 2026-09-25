package rofi

import (
	"context"
	"os/exec"

	"github.com/lewtec/lewkit/x/driver/launcher"
)

type chooserFactory struct{}

func (chooserFactory) ID() string   { return "rofi" }
func (chooserFactory) Name() string { return "Rofi" }
func (chooserFactory) Weight() int  { return 50 }

func (chooserFactory) CheckCompatibility(ctx context.Context) error {
	return launcher.RequireDisplayBinary(ctx, "rofi")
}

func (chooserFactory) New(context.Context) (launcher.Chooser, error) {
	return backend{}, nil
}

type driverFactory struct{}

func (driverFactory) ID() string   { return "rofi" }
func (driverFactory) Name() string { return "Rofi" }
func (driverFactory) Weight() int  { return 50 }

func (driverFactory) CheckCompatibility(ctx context.Context) error {
	return launcher.RequireDisplayBinary(ctx, "rofi")
}

func (driverFactory) New(context.Context) (launcher.Driver, error) {
	return backend{}, nil
}

type backend struct{}

func (backend) Choose(ctx context.Context, opts launcher.ChooseOptions) (*launcher.Item, error) {
	return launcher.ChooseViaCmd(ctx, opts, "rofi", true, "-dmenu", "-p", opts.Prompt, "-show-icons")
}

func (backend) RunApp(ctx context.Context) error {
	return exec.CommandContext(ctx, "rofi", "-show", "combi", "-combi-modi", "drun", "-show-icons").Run()
}

func (backend) SwitchWindow(ctx context.Context) error {
	return exec.CommandContext(ctx, "rofi", "-show", "combi", "-combi-modi", "window", "-show-icons").Run()
}
