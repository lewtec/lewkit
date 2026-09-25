package maim

import (
	"context"
	"fmt"
	"image"
	"os/exec"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/screenshot"
	"github.com/lewtec/lewkit/x/driver/wm"
)

type backend struct{}

func (backend) SelectArea(ctx context.Context) (*wm.Rect, error) {
	if _, err := exec.LookPath("slop"); err != nil {
		return nil, screenshot.ErrSelectionToolNotFound
	}
	out, err := exec.CommandContext(ctx, "slop", "-f", "%x %y %w %h").Output()
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(string(out))
	parts := strings.Fields(raw)
	rect, err := screenshot.ParseRectParts(parts)
	if err != nil {
		return nil, fmt.Errorf("invalid slop output %q: %w", raw, err)
	}
	return rect, nil
}

func (backend) Capture(ctx context.Context, rect *wm.Rect) (image.Image, error) {
	args := []string{}
	if rect != nil {
		args = append(args, "-g", fmt.Sprintf("%dx%d+%d+%d", rect.Width, rect.Height, rect.X, rect.Y))
	}
	return screenshot.CaptureViaCmd(ctx, "maim", args...)
}

func requireBinary(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%w: %s not found", driver.ErrIncompatible, name)
	}
	return nil
}

var _ screenshot.Driver = backend{}
