// Package pbcopy writes the clipboard with pbcopy.
package pbcopy

import (
	"context"
	"fmt"
	"image"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/clipboard"
)

type factory struct{}

func (factory) ID() string   { return "clipboard_pbcopy" }
func (factory) Name() string { return "pbcopy" }
func (factory) Weight() int  { return 60 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if err := driver.ForGOOS("darwin"); err != nil {
		return err
	}
	return clipboard.RequireTool(ctx, "pbcopy")
}

func (factory) New(context.Context) (clipboard.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) WriteText(ctx context.Context, text string) error {
	return clipboard.PipeText(ctx, text, "pbcopy")
}

func (backend) WriteImage(context.Context, image.Image) error {
	return fmt.Errorf("%w: pbcopy text only", driver.ErrIncompatible)
}

func init() { driver.Register[clipboard.Driver](factory{}) }
