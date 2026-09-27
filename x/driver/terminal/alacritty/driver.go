package alacritty

import (
	"context"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/driver/terminal"
)

type factory struct{}

func (factory) ID() string   { return "terminal_alacritty" }
func (factory) Name() string { return "Alacritty" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(ctx context.Context) error {
	return terminal.RequireBinary(ctx, "alacritty")
}

func (factory) New(context.Context) (terminal.Driver, error) {
	return backend{}, nil
}

type backend struct{}

func (backend) Open(ctx context.Context, opts terminal.Options) error {
	cmd := execdriver.MustCommand(ctx, "alacritty", terminal.BuildOpenArgs(opts, "-T", true)...)
	return execdriver.Start(ctx, cmd)
}
