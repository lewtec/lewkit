// Elm is two buttons that add and subtract an integer.
//
//	go run ./cmd/lewkit release run --config ./examples/elm/eletrocromo.json
package main

import (
	"context"

	"github.com/lewtec/lewkit/x/app"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/ui/gui"
)

func init() { entry.Bind(runApp) }

func main() { entry.Main(context.Background(), runApp) }

func runApp(ctx context.Context) error {
	model, err := gui.NewCounter()
	if err != nil {
		return err
	}
	return app.App{
		Title:   "Elm",
		Width:   400,
		Height:  240,
		Handler: app.GUI(model),
	}.Run(ctx)
}
