// Package cocoa shows NSOpenPanel and NSSavePanel.
package cocoa

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/filedialog"
)

type factory struct{}

func (factory) ID() string   { return "filedialog_cocoa" }
func (factory) Name() string { return "NSOpenPanel" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("%w: not darwin", driver.ErrIncompatible)
	}
	return nil
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
