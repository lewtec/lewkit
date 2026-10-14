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
	"fmt"
	"image"
	"strconv"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/wm"
	"github.com/lewtec/lewkit/x/errs"
)

var (
	// ErrSelectionToolNotFound means slurp or slop is not on PATH.
	ErrSelectionToolNotFound = errs.Wrap(errs.ErrNotFound, "selection tool not found")
	// ErrEmptySelection means the selection tool returned no geometry.
	ErrEmptySelection = errs.Wrap(errs.ErrEmpty, "empty selection")
)

//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate enum screenshot.go

// TargetType selects what Capture should frame.
//
//lewkit:enum
type TargetType int

const (
	// TargetAll captures every output. The rectangle is nil.
	TargetAll TargetType = iota // all
	// TargetOutput captures the focused output.
	TargetOutput // output
	// TargetWindow captures the focused window.
	TargetWindow // window
	// TargetSelection captures an interactive selection.
	TargetSelection // selection
)

// Driver captures pixels and reads a selection rectangle.
type Driver interface {
	Capture(ctx context.Context, rect *wm.Rect) (image.Image, error)
	SelectArea(ctx context.Context) (*wm.Rect, error)
}

// ParseRectParts turns four decimal integer strings into a Rect.
func ParseRectParts(parts []string) (*wm.Rect, error) {
	if len(parts) != 4 {
		return nil, fmt.Errorf("%w %d", ErrExpected4GeometryFieldsGot, len(parts))
	}
	vals := make([]int, 4)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("%w %d %q: %w", ErrParseGeometryField, i, p, err)
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
		return nil, fmt.Errorf("%w: %d", ErrTargetType, int(targetType))
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
