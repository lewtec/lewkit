package experiments

import (
	"context"
	"errors"
	"fmt"
	"image"
	"time"

	"github.com/lewtec/lewkit/x/cmd"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/event"
	"github.com/lewtec/lewkit/x/taskgroup"
)

// Window is `lewkit experiments window`.
type Window struct {
	Triangle *triangleCmd
	Perlin   *perlinCmd
	Compute  *Compute
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
	return runDemo(ctx, c.run)
}

func (c *triangleCmd) run(ctx context.Context) error {
	return animateWindow(ctx, windowDemo{
		title:  "lewkit triangle",
		name:   "triangle",
		width:  c.width.Value(),
		height: c.height.Value(),
	}, func(ctx context.Context) (demoPainter, error) {
		return newTrianglePainter(ctx)
	})
}

type perlinCmd struct {
	width  cmd.IntArg[int] `long:"width" default:"800" help:"window width"`
	height cmd.IntArg[int] `long:"height" default:"600" help:"window height"`
}

func (perlinCmd) Description() string {
	return "animate Perlin noise"
}

func (c *perlinCmd) Run(ctx context.Context) error {
	return runDemo(ctx, c.run)
}

func (c *perlinCmd) run(ctx context.Context) error {
	return animateWindow(ctx, windowDemo{
		title:  "lewkit perlin",
		name:   "perlin",
		width:  c.width.Value(),
		height: c.height.Value(),
	}, func(ctx context.Context) (demoPainter, error) {
		return newPerlinPainter(ctx)
	})
}

type demoPainter interface {
	Draw(context.Context, *image.RGBA, float64) error
	Close() error
}

type windowDemo struct {
	title, name   string
	width, height int
}

func animateWindow(ctx context.Context, demo windowDemo, newPainter func(context.Context) (demoPainter, error)) error {
	w, err := window.Open(ctx, window.Config{
		Title:  demo.title,
		Width:  demo.width,
		Height: demo.height,
	})
	if err != nil {
		return err
	}
	p, err := newPainter(ctx)
	if err != nil {
		return errors.Join(err, w.Close())
	}
	taskgroup.Go(ctx, demo.name, taskgroup.CPU, func(ctx context.Context, st *taskgroup.Status) error {
		defer w.Close()
		defer p.Close()
		var fps event.FPS
		return window.Animate(ctx, w, time.Second/60, func(dst *image.RGBA, elapsed time.Duration) error {
			st.Update(fmt.Sprintf("%.0f fps", fps.Get()))
			return p.Draw(ctx, dst, elapsed.Seconds())
		})
	})
	return nil
}
