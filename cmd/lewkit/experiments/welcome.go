package experiments

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/image/convert"
	"github.com/lewtec/lewkit/x/ui/gui"
)

// Welcome is `lewkit experiments welcome`.
type welcomeCmd struct {
	logo   cmd.StringArg `long:"logo" default:"" help:"image to show instead of the LEWTEC lockup"`
	accent cmd.ColorArg  `long:"accent" default:"" help:"accent #RRGGBB; the logo average when omitted"`
}

func (welcomeCmd) Description() string {
	return "pick a folder from the start screen"
}

func (c *welcomeCmd) Run(ctx context.Context) error {
	dirs, err := gui.Recent()
	if err != nil {
		return err
	}
	args := gui.WelcomeArgs{Title: "lewkit", Dirs: dirs}
	if path := c.logo.Value(); path != "" {
		img, err := convert.Open(path)
		if err != nil {
			return err
		}
		args.Logo = img
	}
	if c.accent.IsSet() {
		color := c.accent.Value()
		args.Accent = &color
	}
	model := gui.NewWelcome(args)
	err = runGUI(ctx, "welcome", gui.Options{
		Title:  "lewkit",
		Width:  880,
		Height: 720,
	}, model)
	if path := model.Picked(); path != "" {
		if saveErr := gui.Remember(path); saveErr != nil {
			return saveErr
		}
		fmt.Println(path)
		return nil
	}
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	return err
}
