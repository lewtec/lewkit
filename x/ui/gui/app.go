package gui

import (
	"context"
	"errors"

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
	evaluator := options.Evaluator
	if evaluator == nil {
		evaluator, err = ndarray.Open(ctx)
		if err != nil {
			return errors.Join(err, host.Close())
		}
		defer evaluator.Close()
	}
	return run(ctx, host, evaluator, model)
}
