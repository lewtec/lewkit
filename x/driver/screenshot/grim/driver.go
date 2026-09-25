package grim

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
	if _, err := exec.LookPath("slurp"); err != nil {
		return nil, screenshot.ErrSelectionToolNotFound
	}
	out, err := exec.CommandContext(ctx, "slurp").Output()
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, screenshot.ErrEmptySelection
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == 'x'
	})
	rect, err := screenshot.ParseRectParts(parts)
	if err != nil {
		return nil, fmt.Errorf("invalid slurp output %q: %w", raw, err)
	}
	return rect, nil
}

func (backend) Capture(ctx context.Context, rect *wm.Rect) (image.Image, error) {
	args := []string{}
	if rect != nil {
		args = append(args, "-g", fmt.Sprintf("%d,%d %dx%d", rect.X, rect.Y, rect.Width, rect.Height))
	}
	args = append(args, "-")
	return screenshot.CaptureViaCmd(ctx, "grim", args...)
}

func requireBinary(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%w: %s not found", driver.ErrIncompatible, name)
	}
	return nil
}

var _ screenshot.Driver = backend{}
