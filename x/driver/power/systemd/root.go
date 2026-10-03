// Package systemd locks and stops the machine with loginctl and systemctl.
package systemd

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/driver/power"
)

type factory struct{}

func (factory) ID() string   { return "power_systemd" }
func (factory) Name() string { return "systemd" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if err := execdriver.RequireBinary(ctx, "loginctl"); err != nil {
		return err
	}
	return execdriver.RequireBinary(ctx, "systemctl")
}

func (factory) New(context.Context) (power.Driver, error) { return backend{}, nil }

func init() {
	driver.Register[power.Driver](factory{})
}
