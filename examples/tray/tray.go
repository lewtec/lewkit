// Tray shows a status item until Quit or interrupt.
//
//	go run ./cmd/lewkit release run --config ./examples/tray/eletrocromo.json
package main

import (
	"context"
	"image"
	"image/color"
	"log/slog"

	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/driver/tray"
	"github.com/lewtec/lewkit/x/entry"
)

func init() { entry.Bind(run) }

func main() { entry.Main(run) }

func run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	item, err := tray.Open(ctx, tray.Config{
		Title:   "lewkit",
		Tooltip: "lewkit tray",
		Icon:    greenIcon(),
		Menu: []tray.Item{
			{Label: "Ping", OnClick: func() { slog.Info("tray ping") }},
			{Label: "More", Children: []tray.Item{
				{Label: "About", OnClick: func() { slog.Info("tray about") }},
			}},
			{Separator: true},
			{Label: "Quit", OnClick: cancel},
		},
	})
	if err != nil {
		return err
	}
	defer item.Close()
	slog.Info("tray open", "title", "lewkit")
	<-ctx.Done()
	return nil
}

func greenIcon() tray.Icon {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := range 32 {
		for x := range 32 {
			img.SetNRGBA(x, y, color.NRGBA{R: 32, G: 160, B: 96, A: 255})
		}
	}
	return tray.Icon{Image: img}
}
