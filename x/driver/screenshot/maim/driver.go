package maim

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
	if !execdriver.IsBinaryAvailable(ctx, "slop") {
		return nil, screenshot.ErrSelectionToolNotFound
	}
	out, err := execdriver.MustRun(ctx, "slop", "-f", "%x %y %w %h").Output()
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

func requireBinary(ctx context.Context, name string) error {
	return execdriver.RequireBinary(ctx, name)
}

var _ screenshot.Driver = backend{}
