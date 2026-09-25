// Package systemd locks and stops the machine with loginctl and systemctl.
package systemd

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/power"
)

type factory struct{}

func (factory) ID() string   { return "power_systemd" }
func (factory) Name() string { return "systemd" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error {
	if err := requireBinary("loginctl"); err != nil {
		return err
	}
	return requireBinary("systemctl")
}

func (factory) New(context.Context) (power.Driver, error) { return backend{}, nil }

func requireBinary(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%w: %s not found", driver.ErrIncompatible, name)
	}
	return nil
}

func init() {
	driver.Register[power.Driver](factory{})
}
