package image

import (
	"context"
	"fmt"
	stdimage "image"

	"github.com/lewtec/lewkit/x/ndarray"
)

// Eval writes tensor into destination as packed 0..255 RGBA.
// destination bounds must match the tensor size (h×w×4).
func Eval[T ndarray.Number](ctx context.Context, tensor *ndarray.Tensor[T], evaluator ndarray.Evaluator, destination *stdimage.RGBA) error {
	if tensor == nil || destination == nil {
		return ndarray.ErrOp
	}
	size := tensor.Size()
	if destination.Rect.Dx()*destination.Rect.Dy()*4 != size {
		return fmt.Errorf("%w: image %d×%d×4 != %d", ndarray.ErrSize, destination.Rect.Dx(), destination.Rect.Dy(), size)
	}
	buffer := make([]uint8, size)
	if err := ndarray.Cast[uint8](tensor).Eval(ctx, evaluator, buffer); err != nil {
		return err
	}
	Write(destination, buffer)
	return nil
}

// Raster evals tensor into a new image sized from Shape (h, w, 4).
func Raster[T ndarray.Number](ctx context.Context, tensor *ndarray.Tensor[T], evaluator ndarray.Evaluator) (*stdimage.RGBA, error) {
	if tensor == nil {
		return nil, ndarray.ErrOp
	}
	shape := tensor.Shape()
	if len(shape) < 2 {
		return nil, ndarray.ErrShape
	}
	dst := stdimage.NewRGBA(stdimage.Rect(0, 0, shape[1], shape[0]))
	if err := Eval(ctx, tensor, evaluator, dst); err != nil {
		return nil, err
	}
	return dst, nil
}

// RGBA packs a dense (h, w, 4) uint8 buffer into a new image.
func RGBA(h, w int, pixels []uint8) *stdimage.RGBA {
	dst := stdimage.NewRGBA(stdimage.Rect(0, 0, w, h))
	Write(dst, pixels)
	return dst
}

// Write packs pixels (h, w, 4) uint8 into dst. dst's bounds set h and w.
func Write(dst *stdimage.RGBA, pixels []uint8) {
	if dst == nil {
		return
	}
	w, h := dst.Rect.Dx(), dst.Rect.Dy()
	if w < 1 || h < 1 || len(pixels) < h*w*4 {
		return
	}
	n := w * 4
	for y := range h {
		destIndex := dst.PixOffset(dst.Rect.Min.X, dst.Rect.Min.Y+y)
		sourceIndex := y * n
		copy(dst.Pix[destIndex:destIndex+n], pixels[sourceIndex:sourceIndex+n])
	}
}
