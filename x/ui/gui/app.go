package gui

import (
	"context"
	"errors"
	"log/slog"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/vulkanwindow"
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
	if screen, err := openSwap(ctx, options.Config); err == nil {
		defer screen.Close()
		return run(ctx, swapDisplay{screen}, nil, model)
	} else {
		slog.Debug("vulkan_window", "err", err)
	}
	host, err := window.Open(ctx, options.Config)
	if err != nil {
		return err
	}
	defer host.Close()
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

func openSwap(ctx context.Context, cfg window.Config) (vulkanwindow.Screen, error) {
	handles, err := driver.List[vulkanwindow.Driver](ctx)
	if err != nil {
		return nil, err
	}
	var last error
	for _, handle := range handles {
		if handle.ID != vulkanwindow.ID {
			continue
		}
		impl, err := handle.Open(ctx)
		if err != nil {
			last = err
			continue
		}
		screen, err := impl.Open(ctx, cfg)
		if err != nil {
			last = err
			continue
		}
		return screen, nil
	}
	if last == nil {
		last = driver.ErrUnavailable
	}
	return nil, last
}
