package gui

import (
	"context"

	"github.com/lewtec/lewkit/x/driver/present"
	_ "github.com/lewtec/lewkit/x/driver/present/prelude"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ndarray"
)

func bridge(ctx context.Context, host window.Window) (present.Screen, ndarray.Evaluator, error) {
	surfacer, ok := host.(window.Surfacer)
	if !ok {
		return nil, nil, window.ErrPresent
	}
	surface := surfacer.Surface()
	if surface.A == 0 {
		return nil, nil, window.ErrPresent
	}
	size := host.Size()
	screen, err := present.Open(ctx, surface.Kind, surface.A, surface.B, size.X, size.Y)
	if err != nil {
		return nil, nil, err
	}
	if painter, ok := screen.(present.Painter); ok {
		return screen, painter.Evaluator(), nil
	}
	return screen, nil, nil
}
