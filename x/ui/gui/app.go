package gui

import (
	"context"

	"github.com/lewtec/lewkit/x/driver/ndeval"
	"github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ndarray"
)

func bridge(ctx context.Context, host window.Window) (vulkan.Screen, ndarray.Evaluator, error) {
	surfacer, ok := host.(window.Surfacer)
	if !ok {
		return nil, nil, window.ErrPresent
	}
	surface := surfacer.Surface()
	if surface.A == 0 {
		return nil, nil, window.ErrPresent
	}
	size := host.Size()
	screen, err := vulkan.OpenNative(ctx, surface.Kind, surface.A, surface.B, size.X, size.Y)
	if err != nil {
		return nil, nil, err
	}
	evaluator := ndeval.Bind(screen.Device())
	return screen, evaluator, nil
}
