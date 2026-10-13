// Package gtk shows a message with a GTK 4 window.
package gtk

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/internal/gtkui"
	"github.com/lewtec/lewkit/x/driver/messagebox"
	gtklib "github.com/lewtec/lewkit/x/ffi/native/gtk"

	_ "github.com/lewtec/lewkit/x/driver/thread/std"
)

type factory struct{}

func (factory) ID() string   { return "messagebox_gtk" }
func (factory) Name() string { return "GTK alert" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(ctx context.Context) error { return gtkui.Require(ctx) }

func (factory) New(context.Context) (messagebox.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Show(ctx context.Context, n messagebox.Notice) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return gtkui.Run(ctx, func() error {
		return gtklib.Show(ctx, n.Title, n.Message, n.Style)
	})
}

func init() { driver.Register[messagebox.Driver](factory{}) }

var (
	_ driver.DriverFactory[messagebox.Driver] = factory{}
	_ driver.Weighter                         = factory{}
)
