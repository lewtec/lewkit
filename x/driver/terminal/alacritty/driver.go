package alacritty

import (
	"context"
	"os/exec"

	"github.com/lewtec/lewkit/x/driver/terminal"
)

type factory struct{}

func (factory) ID() string   { return "terminal_alacritty" }
func (factory) Name() string { return "Alacritty" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error {
	return terminal.RequireBinary("alacritty")
}

func (factory) New(context.Context) (terminal.Driver, error) {
	return backend{}, nil
}

type backend struct{}

func (backend) Open(ctx context.Context, opts terminal.Options) error {
	cmd := exec.CommandContext(ctx, "alacritty", terminal.BuildOpenArgs(opts, "-T", true)...)
	return cmd.Start()
}
