package kitty

import (
	"context"
	"os/exec"

	"github.com/lewtec/lewkit/x/driver/terminal"
)

type factory struct{}

func (factory) ID() string   { return "terminal_kitty" }
func (factory) Name() string { return "Kitty" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error {
	return terminal.RequireBinary("kitty")
}

func (factory) New(context.Context) (terminal.Driver, error) {
	return backend{}, nil
}

type backend struct{}

func (backend) Open(ctx context.Context, opts terminal.Options) error {
	cmd := exec.CommandContext(ctx, "kitty", terminal.BuildOpenArgs(opts, "--title", false)...)
	return cmd.Start()
}
