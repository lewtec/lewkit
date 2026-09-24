package experiments

import (
	"context"
	"errors"
	"fmt"

	"github.com/lewtec/lewkit/x/ui/gui"
)

// Welcome is `lewkit experiments welcome`.
type welcomeCmd struct{}

func (welcomeCmd) Description() string {
	return "pick a folder from the start screen"
}

func (*welcomeCmd) Run(ctx context.Context) error {
	dirs, err := gui.Recent()
	if err != nil {
		return err
	}
	model := gui.NewWelcome("lewkit", dirs)
	model.Use(ctx)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	model.OnDone(cancel)
	err = runGUI(runCtx, "welcome", gui.Options{
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
	if errors.Is(err, context.Canceled) && ctx.Err() == nil {
		return nil
	}
	return err
}
