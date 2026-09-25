// Package xdg launches files and URLs with xdg-open.
package xdg

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/opener"
)

type factory struct{}

func (factory) ID() string   { return "opener_xdg" }
func (factory) Name() string { return "xdg-open" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if err := driver.RequireAnyEnv(ctx, "DISPLAY", "WAYLAND_DISPLAY"); err != nil {
		return err
	}
	return opener.RequireTool("xdg-open")
}

func (factory) New(context.Context) (opener.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Open(ctx context.Context, target string) error {
	return opener.Start(ctx, "xdg-open", target)
}

func init() { driver.Register[opener.Driver](factory{}) }
