package main

import (
	"context"
	"errors"
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
	return "draw the RGB triangle, one turn per second"
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
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	evs := w.Subscribe(ctx)
	t0 := time.Now()
	tick := time.NewTicker(time.Second / 60)
	defer tick.Stop()
	if err := paint(w, 0); err != nil {
		return ignoreClosed(err)
	}
	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case ev, ok := <-evs:
			if !ok {
				return nil
			}
			if err := handle(w, ev, time.Since(t0).Seconds()); err != nil {
				return ignoreClosed(err)
			}
		case now := <-tick.C:
			if err := paint(w, now.Sub(t0).Seconds()); err != nil {
				return ignoreClosed(err)
			}
		}
	}
}

func handle(w window.Window, ev window.Event, turn float64) error {
	switch ev.(type) {
	case window.Resize:
		return paint(w, turn)
	case window.Expose:
		return w.Draw()
	case window.Close:
		return window.ErrClosed
	default:
		return nil
	}
}

func paint(w window.Window, turn float64) error {
	frame := w.Frame()
	if frame == nil {
		return window.ErrClosed
	}
	lewimage.TriangleTurn(frame, turn)
	return w.Draw()
}

func ignoreClosed(err error) error {
	if errors.Is(err, window.ErrClosed) {
		return nil
	}
	return err
}
