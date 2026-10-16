package dbus

import (
	"context"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/tray"
)

type factory struct{}

func (factory) ID() string   { return "tray_dbus" }
func (factory) Name() string { return "StatusNotifier" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if err := driver.ForGOOS(build.GOOSLinux); err != nil {
		return err
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
