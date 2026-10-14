// Package win32 shows the Windows common item dialog.
package win32

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/filedialog"
)

type factory struct{}

func (factory) ID() string   { return "filedialog_win32" }
func (factory) Name() string { return "CommonItemDialog" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error {
	return driver.ForGOOS("windows")
}

func (factory) New(context.Context) (filedialog.Driver, error) {
	return opener{}, nil
}

type opener struct{}

var _ driver.DriverFactory[filedialog.Driver] = factory{}
var _ driver.Weighter = factory{}

func init() {
	driver.Register[filedialog.Driver](&factory{})
}
