// Package screenshot captures a screen rectangle and selects an area.
//
//	img, err := screenshot.Capture(ctx, rect)
//	area, err := screenshot.SelectArea(ctx)
//
// Import grim or maim. Grim is Wayland and uses slurp. Maim is X11 and uses slop.
// Capture returns the image. The caller saves it.
package screenshot

import (
	"context"
	"errors"
	"fmt"
	"image"
	"strconv"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/wm"
)

var (
	// ErrSelectionToolNotFound means slurp or slop is not on PATH.
	ErrSelectionToolNotFound = errors.New("selection tool not found")
	// ErrEmptySelection means the selection tool returned no geometry.
	ErrEmptySelection = errors.New("empty selection")
	// ErrUnknownTargetType means the target is not all, output, window, or selection.
	ErrUnknownTargetType = errors.New("unknown target type")
)

// TargetType selects what Capture should frame.
type TargetType int

const (
	// TargetAll captures every output. The rectangle is nil.
	TargetAll TargetType = iota
	// TargetOutput captures the focused output.
	TargetOutput
	// TargetWindow captures the focused window.
	TargetWindow
	// TargetSelection captures an interactive selection.
	TargetSelection
)

// Driver captures pixels and reads a selection rectangle.
type Driver interface {
	Capture(ctx context.Context, rect *wm.Rect) (image.Image, error)
	SelectArea(ctx context.Context) (*wm.Rect, error)
}

// ParseRectParts turns four decimal integer strings into a Rect.
func ParseRectParts(parts []string) (*wm.Rect, error) {
	if len(parts) != 4 {
		return nil, fmt.Errorf("expected 4 geometry fields, got %d", len(parts))
	}
	vals := make([]int, 4)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("parse geometry field %d %q: %w", i, p, err)
		}
		vals[i] = n
	}
	return &wm.Rect{X: vals[0], Y: vals[1], Width: vals[2], Height: vals[3]}, nil
}

// ResolveRect maps target to a capture rectangle. TargetAll returns nil.
func ResolveRect(ctx context.Context, targetType TargetType) (*wm.Rect, error) {
	switch targetType {
	case TargetAll:
		return nil, nil
	case TargetOutput:
		_, rect, err := wm.GetFocusedOutput(ctx)
		return rect, err
	case TargetWindow:
		return wm.GetFocusedWindowRect(ctx)
	case TargetSelection:
		return SelectArea(ctx)
	default:
		return nil, fmt.Errorf("%w: %v", ErrUnknownTargetType, targetType)
	}
}

// Capture returns the image for rect. A nil rect captures the whole screen.
func Capture(ctx context.Context, rect *wm.Rect) (image.Image, error) {
	return driver.WithResult(ctx, func(source Driver) (image.Image, error) {
		return source.Capture(ctx, rect)
	})
}

// SelectArea asks the user for a rectangle.
func SelectArea(ctx context.Context) (*wm.Rect, error) {
	return driver.WithResult(ctx, func(source Driver) (*wm.Rect, error) {
		return source.SelectArea(ctx)
	})
}
