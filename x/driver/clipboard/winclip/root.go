// Package winclip writes the clipboard with the Windows clip command.
package winclip

import (
	"context"
	"fmt"
	"image"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/clipboard"
)

type factory struct{}

func (factory) ID() string   { return "clipboard_winclip" }
func (factory) Name() string { return "clip" }
func (factory) Weight() int  { return 60 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if err := driver.ForGOOS(build.GOOSWindows); err != nil {
		return err
	}
	return clipboard.RequireTool(ctx, "clip")
}

func (factory) New(context.Context) (clipboard.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) WriteText(ctx context.Context, text string) error {
	return clipboard.PipeText(ctx, text, "clip")
}

func (backend) WriteImage(context.Context, image.Image) error {
	return fmt.Errorf("%w: clip text only", driver.ErrIncompatible)
}

func init() { driver.Register[clipboard.Driver](factory{}) }
