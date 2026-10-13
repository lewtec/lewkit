package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/battery"
	"github.com/lewtec/lewkit/x/driver/brightness"
	"github.com/lewtec/lewkit/x/driver/clipboard"
	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/dirs"
	"github.com/lewtec/lewkit/x/driver/screen"
	"github.com/lewtec/lewkit/x/driver/volume"
	"github.com/lewtec/lewkit/x/release"
)

var errUnknown = errors.New("unknown action")

type result struct {
	Name string
	Text string
}

func probe(ctx context.Context, appID string) []result {
	selected := selectedIDs(ctx)
	rows := make([]result, 0, 8)
	rows = append(rows, read("battery", selected, func() (string, error) {
		status, serr := battery.BatteryStatus(ctx)
		level, lerr := battery.BatteryLevel(ctx)
		switch {
		case serr != nil && lerr != nil:
			return "", serr
		case serr != nil:
			return fmt.Sprintf("%d%%", level), serr
		case lerr != nil:
			return string(status), lerr
		default:
			return fmt.Sprintf("%s %d%%", status, level), nil
		}
	}))
	rows = append(rows, read("brightness", selected, func() (string, error) {
		dev, err := brightness.Status(ctx)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s %.0f%%", dev.Name, dev.Brightness*100), nil
	}))
	rows = append(rows, read("volume", selected, func() (string, error) {
		level, err := volume.GetVolume(ctx)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%.0f%%", level*100), nil
	}))
	rows = append(rows, read("mute", selected, func() (string, error) {
		muted, err := volume.GetMute(ctx)
		if err != nil {
			return "", err
		}
		if muted {
			return "muted", nil
		}
		return "on", nil
	}))
	rows = append(rows, read("sink", selected, func() (string, error) {
		return volume.SinkName(ctx)
	}))
	rows = append(rows, read("screen", selected, func() (string, error) {
		on, err := screen.IsDPMSOn(ctx)
		if err != nil {
			return "", err
		}
		if on {
			return "on", nil
		}
		return "off", nil
	}))
	rows = append(rows, read("night", selected, func() (string, error) {
		mode, err := daynight.Current(ctx)
		return mode.String(), err
	}))
	rows = append(rows, read("dirs", selected, func() (string, error) {
		got, err := dirs.Resolve(ctx, appID)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("data %s\ncache %s\nconfig %s", got.Data, got.Cache, got.Config), nil
	}))
	return rows
}

func read(name string, selected map[string]string, fn func() (string, error)) result {
	text, err := fn()
	if id := selected[name]; id != "" {
		if text != "" {
			text = id + ": " + text
		} else {
			text = id
		}
	}
	if err != nil {
		if text != "" {
			text = text + ": " + err.Error()
		} else {
			text = err.Error()
		}
	}
	return result{Name: name, Text: text}
}

func selectedIDs(ctx context.Context) map[string]string {
	out := map[string]string{}
	for _, iface := range driver.Doctor(ctx) {
		name := iface.Name
		if i := strings.LastIndex(name, "/"); i >= 0 {
			name = name[i+1:]
		}
		name = strings.TrimSuffix(name, ".Driver")
		for _, status := range iface.Drivers {
			if status.Selected {
				out[name] = status.ID
			}
		}
	}
	return out
}

type action struct {
	Name string
	Text string
}

func (a action) run(ctx context.Context) error {
	switch a.Name {
	case "brightness-up":
		return brightness.Increase(ctx)
	case "brightness-down":
		return brightness.Decrease(ctx)
	case "clipboard":
		text := strings.TrimSpace(a.Text)
		if text == "" {
			text = release.Name()
		}
		return clipboard.WriteText(ctx, text)
	default:
		return fmt.Errorf("%w: %s", errUnknown, a.Name)
	}
}
