// Gemini browses gemini:// capsules and built-in gemtext pages with Flutter-style animations.
//
//	go run ./cmd/lewkit release run --config ./examples/app/gemini/eletrocromo.json
package main

import (
	"context"

	"github.com/lewtec/lewkit/x/app"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/entry"
)

func init() { entry.Bind(runApp) }

func main() { entry.Main(context.Background(), runApp) }

func runApp(ctx context.Context) error {
	browser, err := NewBrowser(ctx)
	if err != nil {
		return err
	}
	return app.App{
		Title:   "Gemini",
		Width:   960,
		Height:  700,
		Handler: app.GUI(browser),
	}.Run(ctx)
}
