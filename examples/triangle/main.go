// Triangle draws an RGB triangle, one turn every four seconds.
// Plus and minus step the rate by 0.05.
//
//	go run ./cmd/lewkit release run --config ./examples/triangle/eletrocromo.json
package main

import (
	"context"

	"github.com/lewtec/lewkit/examples/internal/scene"
	"github.com/lewtec/lewkit/x/app"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/entry"
)

func init() { entry.Bind(runApp) }

func main() { entry.Main(runApp) }

func runApp(ctx context.Context) error {
	model, err := scene.TriangleModel(800, 600)
	if err != nil {
		return err
	}
	return app.App{
		Title:   "Triangle",
		Width:   800,
		Height:  600,
		Handler: app.GUI(model),
	}.Run(ctx)
}
