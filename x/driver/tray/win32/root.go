package win32

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/tray"
)

type factory struct{}

func (factory) ID() string   { return "tray_win32" }
func (factory) Name() string { return "ShellNotify" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error {
	return driver.ForGOOS("windows")
}

func (factory) New(context.Context) (tray.Driver, error) {
	return opener{}, nil
}

var _ driver.DriverFactory[tray.Driver] = factory{}

func init() {
	driver.Register[tray.Driver](&factory{})
}
