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
	"github.com/lewtec/lewkit/x/ui/gui"
)

// Window is `lewkit experiments window`.
type Window struct {
	Triangle *triangleCmd
	Perlin   *perlinCmd
	Compute  *Compute
	Scroll   *scrollCmd
	Notepad  *notepadCmd
	Counter  *counterCmd
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
	w, err := window.Open(ctx, window.Config{
		Title:  "lewkit triangle",
		Width:  c.width.Value(),
		Height: c.height.Value(),
	})
	if err != nil {
		return err
	}
	p, err := newTrianglePainter(ctx)
	if err != nil {
		return errors.Join(err, w.Close())
	}
	taskgroup.Go(ctx, "triangle", taskgroup.CPU, func(ctx context.Context, st *taskgroup.Status) error {
		defer w.Close()
		defer p.Close()
		var fps event.FPS
		return window.Animate(ctx, w, 0, func(dst *image.RGBA, elapsed time.Duration) error {
			st.Update(fmt.Sprintf("%.0f fps", fps.Get()))
			return p.Draw(ctx, dst, elapsed.Seconds())
		})
	})
	return nil
}

type perlinCmd struct {
	width  cmd.IntArg[int] `long:"width" default:"800" help:"window width"`
	height cmd.IntArg[int] `long:"height" default:"600" help:"window height"`
}

func (perlinCmd) Description() string {
	return "animate Perlin noise"
}

func runGUI(ctx context.Context, name string, options gui.Options, model gui.Model) error {
	return runDemo(ctx, func(ctx context.Context) error {
		taskgroup.Go(ctx, name, taskgroup.CPU, func(ctx context.Context, st *taskgroup.Status) error {
			return gui.Open(ctx, &statusModel{inner: model, status: st}, options)
		})
		return nil
	})
}

type statusModel struct {
	inner  gui.Model
	status *taskgroup.Status
}

func (model *statusModel) Init() gui.Cmd { return model.inner.Init() }

func (model *statusModel) Update(msg gui.Msg) (gui.Model, gui.Cmd) {
	if tick, ok := msg.(gui.TickMsg); ok && model.status != nil {
		model.status.Update(fmt.Sprintf("%.0f fps", tick.FPS))
	}
	next, cmd := model.inner.Update(msg)
	model.inner = next
	return model, cmd
}

func (model *statusModel) View() gui.Node {
	return model.inner.View()
}

func (c *perlinCmd) Run(ctx context.Context) error {
	return runDemo(ctx, c.run)
}

func (c *perlinCmd) run(ctx context.Context) error {
	w, err := window.Open(ctx, window.Config{
		Title:  "lewkit perlin",
		Width:  c.width.Value(),
		Height: c.height.Value(),
	})
	if err != nil {
		return err
	}
	p, err := newPerlinPainter(ctx)
	if err != nil {
		return errors.Join(err, w.Close())
	}
	taskgroup.Go(ctx, "perlin", taskgroup.CPU, func(ctx context.Context, st *taskgroup.Status) error {
		defer w.Close()
		defer p.Close()
		var fps event.FPS
		return window.Animate(ctx, w, 0, func(dst *image.RGBA, elapsed time.Duration) error {
			st.Update(fmt.Sprintf("%.0f fps", fps.Get()))
			return p.Draw(ctx, dst, elapsed.Seconds())
		})
	})
	return nil
}
