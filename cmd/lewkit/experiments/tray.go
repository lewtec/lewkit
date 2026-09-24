package experiments

import (
	"context"
	"image"
	"image/color"
	"log/slog"
	"os"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/driver/tray"
)

// Tray is `lewkit experiments tray`.
type Tray struct {
	icon cmd.StringArg `long:"icon" default:"" help:"PNG, JPEG, ICO, or ICNS file"`
	name cmd.StringArg `long:"name" default:"" help:"freedesktop icon name when --icon is omitted"`
}

func (Tray) Description() string {
	return "show a status item until Quit or interrupt"
}

func (c *Tray) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	icon, err := c.loadIcon()
	if err != nil {
		return err
	}
	item, err := tray.Open(ctx, tray.Config{
		Title:   "lewkit",
		Tooltip: "lewkit tray",
		Icon:    icon,
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

func (c *Tray) loadIcon() (tray.Icon, error) {
	if c.icon.Value() != "" {
		raw, err := os.ReadFile(c.icon.Value())
		if err != nil {
			return tray.Icon{}, err
		}
		return tray.Icon{Bytes: raw}, nil
	}
	if c.name.Value() != "" {
		return tray.Icon{Name: c.name.Value()}, nil
	}
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := range 32 {
		for x := range 32 {
			img.SetNRGBA(x, y, color.NRGBA{R: 32, G: 160, B: 96, A: 255})
		}
	}
	return tray.Icon{Image: img}, nil
}
