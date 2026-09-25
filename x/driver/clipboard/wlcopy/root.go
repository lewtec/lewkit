// Package wlcopy writes the clipboard with wl-copy.
package wlcopy

import (
	"context"
	"image"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/clipboard"
)

type factory struct{}

func (factory) ID() string   { return "clipboard_wlcopy" }
func (factory) Name() string { return "Wayland (wl-copy)" }
func (factory) Weight() int  { return 60 }

func (factory) CheckCompatibility(ctx context.Context) error {
	return clipboard.RequireEnvTool(ctx, "WAYLAND_DISPLAY", "wl-copy")
}

func (factory) New(context.Context) (clipboard.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) WriteText(ctx context.Context, text string) error {
	return clipboard.PipeText(ctx, text, "wl-copy")
}

func (backend) WriteImage(ctx context.Context, img image.Image) error {
	return clipboard.PipeImage(ctx, img, "wl-copy", "-t", "image/png")
}

func init() { driver.Register[clipboard.Driver](factory{}) }
