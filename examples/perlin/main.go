// Perlin animates Perlin noise.
//
//	go run ./cmd/lewkit release run --config ./examples/perlin/eletrocromo.json
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
	model, err := scene.PerlinModel(800, 600)
	if err != nil {
		return err
	}
	return app.App{
		Title:   "Perlin",
		Width:   800,
		Height:  600,
		Handler: app.GUI(model),
	}.Run(ctx)
}
