package win32

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
)

type factory struct{}

func (factory) ID() string   { return "window_win32" }
func (factory) Name() string { return "Win32" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error {
	return driver.RequireGOOS("windows")
}

func (factory) New(context.Context) (window.Driver, error) {
	return wdriver{}, nil
}

type wdriver struct{}
