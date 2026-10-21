//go:build !linux || android

package x11

import (
	"context"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
)

var _ driver.DriverFactory[window.Driver] = factory{}

type factory struct{}

func (factory) ID() string   { return "window_x11" }
func (factory) Name() string { return "X11" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error {
	return driver.ForGOOS(build.GOOSLinux)
}

func (factory) New(context.Context) (window.Driver, error) {
	return nil, driver.ForGOOS(build.GOOSLinux)
}

func init() {
	driver.Register[window.Driver](&factory{})
}
