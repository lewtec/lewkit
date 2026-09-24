package cocoa

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/tray"
)

type factory struct{}

func (factory) ID() string   { return "tray_cocoa" }
func (factory) Name() string { return "NSStatusItem" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("%w: not darwin", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (tray.Driver, error) {
	return opener{}, nil
}

var _ driver.DriverFactory[tray.Driver] = factory{}

func init() {
	driver.Register[tray.Driver](&factory{})
}
