// Package win32 shows the Windows common item dialog.
package win32

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/chooser"
)

type factory struct{}

func (factory) ID() string   { return "chooser_win32" }
func (factory) Name() string { return "CommonItemDialog" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("%w: not windows", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (chooser.Driver, error) {
	return opener{}, nil
}

type opener struct{}

var _ driver.DriverFactory[chooser.Driver] = factory{}
var _ driver.Weighter = factory{}

func init() {
	driver.Register[chooser.Driver](&factory{})
}
