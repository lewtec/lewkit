// Package dbus controls media players over MPRIS.
package dbus

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/media"
)

type factory struct{}

func (factory) ID() string   { return "media_mpris" }
func (factory) Name() string { return "MPRIS (DBus)" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(ctx context.Context) error { return available(ctx) }

func (factory) New(ctx context.Context) (media.Driver, error) { return open(ctx) }

func init() {
	driver.Register[media.Driver](factory{})
}
