// Package pulse sets the default sink with pactl.
package pulse

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/volume"
)

type factory struct{}

func (factory) ID() string   { return "volume_pulse" }
func (factory) Name() string { return "PulseAudio (pactl)" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error { return requireBinary("pactl") }

func (factory) New(context.Context) (volume.Driver, error) { return backend{}, nil }

func requireBinary(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%w: %s not found", driver.ErrIncompatible, name)
	}
	return nil
}

func init() {
	driver.Register[volume.Driver](factory{})
}
