// Package cocoa shows an NSAlert. App mode leaves the dialog to the host.
package cocoa

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/messagebox"
)

type factory struct{}

func (factory) ID() string   { return "messagebox_cocoa" }
func (factory) Name() string { return "AppKit alert" }
func (factory) Weight() int  { return 60 }

func (factory) CheckCompatibility(ctx context.Context) error { return available(ctx) }

func (factory) New(ctx context.Context) (messagebox.Driver, error) { return open(ctx) }

func init() { driver.Register[messagebox.Driver](factory{}) }

var (
	_ driver.DriverFactory[messagebox.Driver] = factory{}
	_ driver.Weighter                         = factory{}
)
