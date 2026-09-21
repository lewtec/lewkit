package experiments

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/ui/gui"
)

type counterCmd struct {
	width  cmd.IntArg[int] `long:"width" default:"400" help:"window width"`
	height cmd.IntArg[int] `long:"height" default:"240" help:"window height"`
}

func (counterCmd) Description() string {
	return "elm counter, two buttons"
}

func (c *counterCmd) Run(ctx context.Context) error {
	model, err := gui.NewCounter()
	if err != nil {
		return err
	}
	return runGUI(ctx, "counter", gui.Options{
		Title:  "lewkit counter",
		Width:  c.width.Value(),
		Height: c.height.Value(),
	}, model)
}
