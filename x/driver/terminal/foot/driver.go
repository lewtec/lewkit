package foot

import (
	"context"
	"os/exec"

	"github.com/lewtec/lewkit/x/driver/terminal"
)

type factory struct{}

func (factory) ID() string   { return "terminal_foot" }
func (factory) Name() string { return "Foot" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(ctx context.Context) error {
	return terminal.RequireEnvBinary(ctx, "WAYLAND_DISPLAY", "foot")
}

func (factory) New(context.Context) (terminal.Driver, error) {
	return backend{}, nil
}

type backend struct{}

func (backend) Open(ctx context.Context, opts terminal.Options) error {
	cmd := exec.CommandContext(ctx, "foot", terminal.BuildOpenArgs(opts, "-T", false)...)
	return cmd.Start()
}
