package experiments

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/ui/gui"
)

type notepadCmd struct {
	width  cmd.IntArg[int] `long:"width" default:"640" help:"window width"`
	height cmd.IntArg[int] `long:"height" default:"480" help:"window height"`
}

func (notepadCmd) Description() string {
	return "unsaved notepad"
}

func (c *notepadCmd) Run(ctx context.Context) error {
	model, err := gui.NewNotepad()
	if err != nil {
		return err
	}
	return runGUI(ctx, "notepad", gui.Options{
		Title:  "lewkit notepad",
		Width:  c.width.Value(),
		Height: c.height.Value(),
	}, model)
}
