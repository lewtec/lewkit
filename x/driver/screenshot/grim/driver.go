package grim

import (
	"context"
	"fmt"
	"image"
	"strings"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/driver/screenshot"
	"github.com/lewtec/lewkit/x/driver/wm"
)

type backend struct{}

func (backend) SelectArea(ctx context.Context) (*wm.Rect, error) {
	if !execdriver.IsBinaryAvailable(ctx, "slurp") {
		return nil, screenshot.ErrSelectionToolNotFound
	}
	out, err := execdriver.MustRun(ctx, "slurp").Output()
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

func requireBinary(ctx context.Context, name string) error {
	return execdriver.RequireBinary(ctx, name)
}

var _ screenshot.Driver = backend{}
