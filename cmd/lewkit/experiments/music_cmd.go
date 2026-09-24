package experiments

import (
	"context"
	"errors"

	"github.com/lewtec/lewkit/cmd/lewkit/experiments/music"
	"github.com/lewtec/lewkit/x/ui/gui"
)

// musicCmd hosts the music package in the window demo session.
type musicCmd struct {
	music.Cmd
}

func (c *musicCmd) Run(ctx context.Context) error {
	model, closeLibrary, err := c.Open(ctx)
	if err != nil {
		return err
	}
	return errors.Join(runGUI(ctx, "music", gui.Options{
		Title:  "lewkit music",
		Width:  c.Width.Value(),
		Height: c.Height.Value(),
	}, model), closeLibrary())
}
