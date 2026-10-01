// Scroll draws rounded translucent boxes in a loop.
//
//	go run ./cmd/lewkit release run --config ./examples/scroll/eletrocromo.json
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
	model, err := gui.NewMarquee()
	if err != nil {
		return err
	}
	return app.App{
		Title:   "Scroll",
		Width:   800,
		Height:  600,
		Handler: app.GUI(model),
	}.Run(ctx)
}
