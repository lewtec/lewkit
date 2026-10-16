// Package darwin reads battery status and charge level from AppleSmartBattery.
// ioreg prints the IORegistry entry. Level is CurrentCapacity over MaxCapacity.
package darwin

import (
	"context"
	"fmt"
	"os"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/battery"
)

const ioregBin = "/usr/sbin/ioreg"

type factory struct{}

func (factory) ID() string   { return "battery_darwin" }
func (factory) Name() string { return "AppleSmartBattery" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if err := requireDarwin(); err != nil {
		return err
	}
	_, err := read(ctx)
	return err
}

func (factory) New(context.Context) (battery.Driver, error) { return backend{}, nil }

func requireDarwin() error {
	if err := driver.ForGOOS(build.GOOSDarwin); err != nil {
		return err
	}
	if _, err := os.Stat(ioregBin); err != nil {
		return fmt.Errorf("%w: %s", driver.ErrIncompatible, ioregBin)
	}
	return nil
}

func init() {
	driver.Register[battery.Driver](factory{})
}
