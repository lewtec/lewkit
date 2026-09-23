package gui

import (
	"context"
	"errors"
	"log/slog"

	"github.com/lewtec/lewkit/x/driver/ndeval"
	"github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ndarray"
)

// Options configures [Open].
type Options struct {
	window.Config
	Evaluator ndarray.Evaluator
}

// Open creates a host window and evaluator, then [Run]s model until close.
// Frame rate arrives on the model as [TickMsg.FPS].
func Open(ctx context.Context, model Model, options Options) error {
	host, err := window.Open(ctx, options.Config)
	if err != nil {
		return err
	}
	defer host.Close()
	if screen, evaluator, err := bridge(ctx, host); err == nil {
		defer screen.Close()
		defer evaluator.Close()
		return run(ctx, bridgeDisplay{Window: host, screen: screen, evaluator: evaluator}, nil, model)
	} else {
		slog.Debug("swapchain", "err", err)
	}
	evaluator := options.Evaluator
	if evaluator == nil {
		evaluator, err = ndarray.Open(ctx)
		if err != nil {
			return errors.Join(err, host.Close())
		}
		defer evaluator.Close()
	}
	return run(ctx, imageDisplay{host}, evaluator, model)
}

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
