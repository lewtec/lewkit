// Package xclip writes the clipboard with xclip.
package xclip

import (
	"context"
	"image"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/clipboard"
)

type factory struct{}

func (factory) ID() string   { return "clipboard_xclip" }
func (factory) Name() string { return "X11 (xclip)" }
func (factory) Weight() int  { return 40 }

func (factory) CheckCompatibility(ctx context.Context) error {
	return clipboard.RequireEnvTool(ctx, "DISPLAY", "xclip")
}

func (factory) New(context.Context) (clipboard.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) WriteText(ctx context.Context, text string) error {
	return clipboard.PipeText(ctx, text, "xclip", "-selection", "clipboard")
}

func (backend) WriteImage(ctx context.Context, img image.Image) error {
	return clipboard.PipeImage(ctx, img, "xclip", "-selection", "clipboard", "-t", "image/png")
}

func init() { driver.Register[clipboard.Driver](factory{}) }
