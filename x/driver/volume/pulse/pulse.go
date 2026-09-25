package pulse

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

const sink = "@DEFAULT_SINK@"

type backend struct{}

func (backend) SetVolume(ctx context.Context, level float64) error {
	percent := fmt.Sprintf("%d%%", int(level*100))
	if err := run(ctx, "pactl", "set-sink-volume", sink, percent); err != nil {
		return fmt.Errorf("set volume: %w", err)
	}
	return nil
}

func (backend) GetVolume(ctx context.Context) (float64, error) {
	out, err := output(ctx, "pactl", "get-sink-volume", sink)
	if err != nil {
		return 0, err
	}
	return parseVolume(out)
}

func (backend) GetMute(ctx context.Context) (bool, error) {
	out, err := output(ctx, "pactl", "get-sink-mute", sink)
	if err != nil {
		return false, err
	}
	return strings.Contains(out, "yes"), nil
}

func (backend) SinkName(ctx context.Context) (string, error) {
	out, err := output(ctx, "pactl", "get-default-sink")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (b backend) ToggleMute(ctx context.Context) error {
	muted, err := b.GetMute(ctx)
	if err != nil {
		return err
	}
	state := "yes"
	if muted {
		state = "no"
	}
	return run(ctx, "pactl", "set-sink-mute", sink, state)
}

func parseVolume(output string) (float64, error) {
	for _, item := range strings.Split(strings.TrimSpace(output), " ") {
		before, ok := strings.CutSuffix(item, "%")
		if !ok {
			continue
		}
		volume, err := strconv.Atoi(before)
		if err != nil {
			return 0, err
		}
		return float64(volume) / 100, nil
	}
	return 0, nil
}

func run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func output(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return string(out), nil
}
