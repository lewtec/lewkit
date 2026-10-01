// Notepad is an editor. The buffer stays in memory.
//
//	go run ./cmd/lewkit release run --config ./examples/notepad/eletrocromo.json
package main

import (
	"context"

	"github.com/lewtec/lewkit/x/app"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/ui/gui"
)

func init() { entry.Bind(runApp) }

func main() { entry.Main(runApp) }

func runApp(ctx context.Context) error {
	model, err := gui.NewNotepad()
	if err != nil {
		return err
	}
	return app.App{
		Title:   "Notepad",
		Width:   640,
		Height:  480,
		Handler: app.GUI(model),
	}.Run(ctx)
}
