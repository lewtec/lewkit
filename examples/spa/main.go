// Spa serves a templ page as a single-page app.
//
//	go run ./cmd/lewkit release run --config ./examples/spa/eletrocromo.json
package main

import (
	"context"

	"github.com/lewtec/lewkit/x/app"
	"github.com/lewtec/lewkit/x/entry"
)

func init() { entry.Bind(runApp) }

func main() { entry.Main(runApp) }

func runApp(ctx context.Context) error {
	handler, err := spaHandler(ctx)
	if err != nil {
		return err
	}
	return app.App{
		Title:   "Spa",
		Width:   800,
		Height:  600,
		Handler: app.Web(handler),
	}.Run(ctx)
}
