// Package qt shows the Qt file chooser through the KDE portal backend.
package qt

import (
	"context"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/filedialog"
	"github.com/lewtec/lewkit/x/driver/filedialog/portal"
)

type factory struct{}

func (factory) ID() string   { return "filedialog_qt" }
func (factory) Name() string { return "Qt" }

func (factory) Weight() int {
	if portal.PreferQt() {
		return 80
	}
	return 40
}

func (factory) CheckCompatibility(ctx context.Context) error {
	if err := driver.ForGOOS(build.GOOSLinux); err != nil {
		return err
	}
	return portal.Available(ctx, portal.KDE)
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
