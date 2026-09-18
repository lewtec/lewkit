package experiments

import (
	"context"
	"fmt"
	"image"
	"time"

	"github.com/lewtec/lewkit/x/cmd"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/driver/window"
	lewimage "github.com/lewtec/lewkit/x/image"
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
	fps := fpsMeter{t0: time.Now()}
	return window.Animate(ctx, w, time.Second/60, func(dst *image.RGBA, elapsed time.Duration) error {
		lewimage.TriangleTurn(dst, elapsed.Seconds())
		lewimage.Label(dst, 8, 16, fmt.Sprintf("%d fps", fps.hit()))
		return nil
	})
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
