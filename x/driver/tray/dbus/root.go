package dbus

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/tray"
)

type factory struct{}

func (factory) ID() string   { return "tray_dbus" }
func (factory) Name() string { return "StatusNotifier" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("%w: not linux", driver.ErrIncompatible)
	}
	return driver.RequireEnv(ctx, "DBUS_SESSION_BUS_ADDRESS")
}

func (factory) New(context.Context) (tray.Driver, error) {
	return opener{}, nil
}

var _ driver.DriverFactory[tray.Driver] = factory{}

func init() {
	driver.Register[tray.Driver](&factory{})
}
