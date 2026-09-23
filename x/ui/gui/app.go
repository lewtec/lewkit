package gui

import (
	"context"
	"errors"
	"log/slog"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ndarray"
)

// vulkanWindowID is x/driver/window/vulkanwindow.ID. gui does not import that package.
const vulkanWindowID = "vulkan_window"

// Options configures [Open].
type Options struct {
	window.Config
	Evaluator ndarray.Evaluator
}

// Open creates a host window and evaluator, then [Run]s model until close.
// Frame rate arrives on the model as [TickMsg.FPS].
func Open(ctx context.Context, model Model, options Options) error {
	host, err := openWindow(ctx, options.Config)
	if err != nil {
		return err
	}
	defer host.Close()
	evaluator := options.Evaluator
	if evaluator == nil {
		if source, ok := host.(window.EvaluatorSource); ok {
			evaluator, err = source.Evaluator(ctx)
		} else {
			evaluator, err = ndarray.Open(ctx)
		}
		if err != nil {
			return errors.Join(err, host.Close())
		}
		defer evaluator.Close()
	}
	return run(ctx, host, evaluator, model)
}

// openWindow prefers the vulkan_window driver. Any other host stays on window.Open.
func openWindow(ctx context.Context, cfg window.Config) (window.Window, error) {
	handles, err := driver.List[window.Driver](ctx)
	if err == nil {
		for _, handle := range handles {
			if handle.ID != vulkanWindowID {
				continue
			}
			impl, openErr := handle.Open(ctx)
			if openErr != nil {
				slog.Debug("vulkan_window", "err", openErr)
				break
			}
			host, openErr := impl.Open(ctx, cfg)
			if openErr != nil {
				slog.Debug("vulkan_window", "err", openErr)
				break
			}
			return host, nil
		}
	}
	return window.Open(ctx, cfg)
}
