package cocoa

//go:generate go run ../../../../cmd/lewkit generate header --package cocoa --bind CoreGraphics --out coregraphics_bind.go cg.h

import (
	"context"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
)

type factory struct{}

func (factory) ID() string   { return "window_cocoa" }
func (factory) Name() string { return "Cocoa" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error {
	return driver.ForGOOS(build.GOOSDarwin)
}

func (factory) New(context.Context) (window.Driver, error) {
	return cdriver{}, nil
}

type cdriver struct{}
