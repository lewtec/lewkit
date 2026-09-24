// Package gtk shows the GTK file chooser through the gtk portal backend.
package gtk

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/chooser"
	"github.com/lewtec/lewkit/x/driver/chooser/portal"
)

type factory struct{}

func (factory) ID() string   { return "chooser_gtk" }
func (factory) Name() string { return "GTK" }

func (factory) Weight() int {
	if portal.PreferQt() {
		return 40
	}
	return 60
}

func (factory) CheckCompatibility(ctx context.Context) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("%w: not linux", driver.ErrIncompatible)
	}
	return portal.Available(ctx, portal.GTK)
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
