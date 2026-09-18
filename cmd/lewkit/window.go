package main

import (
	"context"
	"errors"
	"image"
	"os"
	"time"

	"github.com/lewtec/lewkit/x/cmd"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/driver/window"
	lewimage "github.com/lewtec/lewkit/x/image"
)

type windowCmd struct {
	triangle *triangleCmd
}

func (windowCmd) Description() string {
	return "host window demos"
}

func (windowCmd) Run(ctx context.Context) error {
	text, err := cmd.Usage[windowCmd]("lewkit window")
	if err != nil {
		return err
	}
	_, err = os.Stdout.WriteString(text)
	return err
}

type triangleCmd struct {
	width  cmd.IntArg[int] `long:"width" default:"800" help:"window width"`
	height cmd.IntArg[int] `long:"height" default:"600" help:"window height"`
}

func (triangleCmd) Description() string {
	return "draw the RGB triangle and redraw on resize"
}

func (c *triangleCmd) Run(ctx context.Context) error {
	w, err := window.Open(ctx, window.Config{
		Title:  "lewkit triangle",
		Width:  c.width.Value(),
		Height: c.height.Value(),
	})
	if err != nil {
		return err
	}
	defer w.Close()
	return paintWindow(ctx, w)
}

func paintWindow(ctx context.Context, w window.Window) error {
	var last image.Point
	tick := time.NewTicker(16 * time.Millisecond)
	defer tick.Stop()
	for {
		if err := paintIfResized(w, &last); err != nil {
			if errors.Is(err, window.ErrClosed) {
				return nil
			}
			return err
		}
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case <-tick.C:
		}
	}
}

func paintIfResized(w window.Window, last *image.Point) error {
	frame := w.Frame()
	if frame == nil {
		return window.ErrClosed
	}
	size := frame.Rect.Size()
	if size == *last {
		return nil
	}
	lewimage.Triangle(frame)
	if err := w.Draw(); err != nil {
		return err
	}
	*last = size
	return nil
}
