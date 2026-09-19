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
	"github.com/lewtec/lewkit/x/taskgroup"
)

// Window is `lewkit experiments window`.
type Window struct {
	Triangle *triangleCmd
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
		var (
			last time.Time
			fps  float64
		)
		return window.Animate(ctx, w, time.Second/60, func(dst *image.RGBA, elapsed time.Duration) error {
			if err := p.Draw(ctx, dst, elapsed.Seconds()); err != nil {
				return err
			}
			now := time.Now()
			if !last.IsZero() {
				dt := now.Sub(last).Seconds()
				if dt > 0 {
					inst := 1 / dt
					if fps == 0 {
						fps = inst
					} else {
						fps = fps*0.85 + inst*0.15
					}
					st.Update(fmt.Sprintf("%.0f fps", fps))
				}
			}
			last = now
			return nil
		})
	})
	return nil
}
