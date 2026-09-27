// Package pulse sets the default sink with pactl.
package pulse

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/driver/volume"
)

type factory struct{}

func (factory) ID() string   { return "volume_pulse" }
func (factory) Name() string { return "PulseAudio (pactl)" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(ctx context.Context) error { return requireBinary(ctx, "pactl") }

func (factory) New(context.Context) (volume.Driver, error) { return backend{}, nil }

func requireBinary(ctx context.Context, name string) error {
	return execdriver.RequireBinary(ctx, name)
}

func init() {
	driver.Register[volume.Driver](factory{})
}
