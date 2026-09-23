package window

import (
	"context"

	"github.com/lewtec/lewkit/x/ndarray"
)

// EvaluatorSource is a window that owns the evaluator for its swapchain.
type EvaluatorSource interface {
	Evaluator(ctx context.Context) (ndarray.Evaluator, error)
}

// DevicePainter presents a tensor without copying it into Frame.
type DevicePainter interface {
	PaintDevice(ctx context.Context, tensor *ndarray.Tensor[uint8], evaluator ndarray.Evaluator) error
}

// Show presents tensor. A [DevicePainter] keeps the canvas on the GPU.
// Every other window evaluates into Frame and Draw blits that page.
func Show(ctx context.Context, host Window, tensor *ndarray.Tensor[uint8], evaluator ndarray.Evaluator) error {
	if painter, ok := host.(DevicePainter); ok {
		return painter.PaintDevice(ctx, tensor, evaluator)
	}
	if err := Present(ctx, tensor, evaluator, host.Frame()); err != nil {
		return err
	}
	return host.Draw()
}
