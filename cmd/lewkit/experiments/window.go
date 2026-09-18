package experiments

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"time"

	"github.com/lewtec/lewkit/x/cmd"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/driver/window"
	lewimage "github.com/lewtec/lewkit/x/image"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// Window is `lewkit experiments window`.
type Window struct {
	Triangle *triangleCmd
}

func (Window) Description() string {
	return "host window demos"
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
	meter := fpsMeter{t0: t0}
	tick := time.NewTicker(time.Second / 60)
	defer tick.Stop()
	if err := paint(w, 0, 0); err != nil {
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
			if err := paint(w, time.Since(t0).Seconds(), meter.hit()); err != nil {
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

func paint(w window.Window, turn float64, fps int) error {
	frame := w.Frame()
	if frame == nil {
		return window.ErrClosed
	}
	lewimage.TriangleTurn(frame, turn)
	drawFPS(frame, fps)
	return w.Draw()
}

func lastFrame(w window.Window) *image.RGBA {
	if b, ok := w.(interface{ Front() *image.RGBA }); ok {
		return b.Front()
	}
	return w.Frame()
}

type fpsMeter struct {
	t0   time.Time
	n    int
	last int
}

func (m *fpsMeter) hit() int {
	m.n++
	if time.Since(m.t0) >= time.Second {
		m.last = m.n
		m.n = 0
		m.t0 = time.Now()
	}
	return m.last
}

func drawFPS(dst *image.RGBA, fps int) {
	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(color.RGBA{R: 255, G: 255, B: 255, A: 255}),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(8, 16),
	}
	d.DrawString(fmt.Sprintf("%d fps", fps))
}

func ignoreClosed(err error) error {
	if errors.Is(err, window.ErrClosed) {
		return nil
	}
	return err
}
