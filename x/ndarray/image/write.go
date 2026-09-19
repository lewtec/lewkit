package image

import (
	"context"
	"fmt"
	stdimage "image"

	"github.com/lewtec/lewkit/x/ndarray"
)

// Eval writes tensor into destination as packed 0..255 RGBA.
// destination bounds must match the tensor size (h×w×4).
func Eval(ctx context.Context, tensor *ndarray.Tensor, evaluator ndarray.Evaluator, destination *stdimage.RGBA) error {
	if tensor == nil || destination == nil {
		return ndarray.ErrOp
	}
	size := tensor.Size()
	if destination.Rect.Dx()*destination.Rect.Dy()*4 != size {
		return fmt.Errorf("%w: image %d×%d×4 != %d", ndarray.ErrSize, destination.Rect.Dx(), destination.Rect.Dy(), size)
	}
	buffer := make([]float32, size)
	if err := tensor.Eval(ctx, evaluator, buffer); err != nil {
		return err
	}
	Write(destination, buffer)
	return nil
}
