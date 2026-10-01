// Music browses a dropped music directory and plays it.
//
//	go run ./cmd/lewkit release run --config ./examples/music/eletrocromo.json
package main

import (
	"context"
	"errors"
	"os"

	"github.com/lewtec/lewkit/x/app"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/entry"
)

func init() { entry.Bind(runApp) }

func main() { entry.Main(runApp) }

func runApp(ctx context.Context) error {
	dir := ""
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	model, closeLibrary, err := openMusic(ctx, dir)
	if err != nil {
		return err
	}
	err = app.App{
		Title:   "Music",
		Width:   900,
		Height:  700,
		Handler: app.GUI(model),
	}.Run(ctx)
	return errors.Join(err, closeLibrary())
}
