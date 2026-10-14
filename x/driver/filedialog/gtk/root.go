// Package gtk shows the GTK file chooser through the gtk portal backend.
package gtk

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/filedialog"
	"github.com/lewtec/lewkit/x/driver/filedialog/portal"
)

type factory struct{}

func (factory) ID() string   { return "filedialog_gtk" }
func (factory) Name() string { return "GTK" }

func (factory) Weight() int {
	if portal.PreferQt() {
		return 40
	}
	return 60
}

func (factory) CheckCompatibility(ctx context.Context) error {
	if err := driver.ForGOOS("linux"); err != nil {
		return err
	}
	return portal.Available(ctx, portal.GTK)
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
