package experiments

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/ui/gui"
)

type scrollCmd struct {
	width  cmd.IntArg[int] `long:"width" default:"800" help:"window width"`
	height cmd.IntArg[int] `long:"height" default:"600" help:"window height"`
}

func (scrollCmd) Description() string {
	return "rounded translucent boxes scrolling in a loop"
}

func (c *scrollCmd) Run(ctx context.Context) error {
	model, err := gui.NewMarquee()
	if err != nil {
		return err
	}
	return runGUI(ctx, "scroll", gui.Options{
		Title:  "lewkit scroll",
		Width:  c.width.Value(),
		Height: c.height.Value(),
	}, model)
}
