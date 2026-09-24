package cocoa

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/daynight"
)

type factory struct{}

func (factory) ID() string   { return "daynight_cocoa" }
func (factory) Name() string { return "AppKit" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(ctx context.Context) error { return available(ctx) }

func (factory) New(ctx context.Context) (daynight.Driver, error) { return open(ctx) }

func init() {
	driver.Register[daynight.Driver](factory{})
}
