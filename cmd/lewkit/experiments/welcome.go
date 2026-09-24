package experiments

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/ui/gui"
)

// Welcome is `lewkit experiments welcome`.
type welcomeCmd struct {
	logo cmd.StringArg `long:"logo" default:"" help:"image to show instead of the LEWTEC lockup"`
}

func (welcomeCmd) Description() string {
	return "pick a folder from the start screen"
}

func (c *welcomeCmd) Run(ctx context.Context) error {
	dirs, err := gui.Recent()
	if err != nil {
		return err
	}
	model := gui.NewWelcome("lewkit", dirs)
	if path := c.logo.Value(); path != "" {
		img, err := loadLogo(path)
		if err != nil {
			return err
		}
		model.Logo(img)
	}
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

func loadLogo(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	return img, err
}
