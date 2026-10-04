// Package win32 shows a message box.
package win32

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/messagebox"
)

type factory struct{}

func (factory) ID() string   { return "messagebox_win32" }
func (factory) Name() string { return "Win32 alert" }
func (factory) Weight() int  { return 60 }

func (factory) CheckCompatibility(ctx context.Context) error { return available(ctx) }

func (factory) New(ctx context.Context) (messagebox.Driver, error) { return open(ctx) }

func init() { driver.Register[messagebox.Driver](factory{}) }

var (
	_ driver.DriverFactory[messagebox.Driver] = factory{}
	_ driver.Weighter                         = factory{}
)
