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
		if err := drainEvents(evs); err != nil {
			return ignoreClosed(err)
		}
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case ev, ok := <-evs:
			if !ok {
				return nil
			}
			if err := handle(ev); err != nil {
				return ignoreClosed(err)
			}
		case <-tick.C:
			if err := paint(w, time.Since(t0).Seconds()); err != nil {
				return ignoreClosed(err)
			}
		}
	}
}

func drainEvents(evs <-chan window.Event) error {
	for {
		select {
		case ev, ok := <-evs:
			if !ok {
				return window.ErrClosed
			}
			if err := handle(ev); err != nil {
				return err
			}
		default:
			return nil
		}
	}
}

func handle(ev window.Event) error {
	if _, ok := ev.(window.Close); ok {
		return window.ErrClosed
	}
	return nil
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
