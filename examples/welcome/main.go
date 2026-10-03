// Welcome picks a folder from the start screen.
//
//	go run ./cmd/lewkit release run --config ./examples/welcome/eletrocromo.json
package main

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/app"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/ui/gui"
)

func init() { entry.Bind(runApp) }

func main() { entry.Main(runApp) }

func runApp(ctx context.Context) error {
	dirs, err := gui.Recent()
	if err != nil {
		return err
	}
	model := gui.NewWelcome(gui.WelcomeArgs{Title: release.Name(), Dirs: dirs})
	err = app.App{
		Title:   "Welcome",
		Width:   880,
		Height:  720,
		Handler: app.GUI(model),
	}.Run(ctx)
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
