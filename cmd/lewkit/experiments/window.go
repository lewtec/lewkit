package experiments

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/cmd"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
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
	Webview  *webviewCmd
	Spa      *spaCmd
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
	model, err := newFrameModel(1, triangleDynamic)
	if err != nil {
		return err
	}
	return runGUI(ctx, "triangle", gui.Options{
		Title:  "lewkit triangle",
		Width:  c.width.Value(),
		Height: c.height.Value(),
	}, model)
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
	model, err := newFrameModel(0.4, perlinDynamic)
	if err != nil {
		return err
	}
	return runGUI(ctx, "perlin", gui.Options{
		Title:  "lewkit perlin",
		Width:  c.width.Value(),
		Height: c.height.Value(),
	}, model)
}
