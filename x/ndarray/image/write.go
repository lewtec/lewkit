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

// RGBA packs a dense (h, w, 4) float32 buffer into a new image.
func RGBA(h, w int, pixels []float32) *stdimage.RGBA {
	dst := stdimage.NewRGBA(stdimage.Rect(0, 0, w, h))
	Write(dst, pixels)
	return dst
}

// Write packs pixels (h, w, 4) float32 into dst. dst's bounds set h and w.
func Write(dst *stdimage.RGBA, pixels []float32) {
	if dst == nil {
		return
	}
	w, h := dst.Rect.Dx(), dst.Rect.Dy()
	if w < 1 || h < 1 || len(pixels) < h*w*4 {
		return
	}
	for y := range h {
		destIndex := dst.PixOffset(dst.Rect.Min.X, dst.Rect.Min.Y+y)
		sourceIndex := y * w * 4
		for range w {
			dst.Pix[destIndex] = toUint8(pixels[sourceIndex])
			dst.Pix[destIndex+1] = toUint8(pixels[sourceIndex+1])
			dst.Pix[destIndex+2] = toUint8(pixels[sourceIndex+2])
			dst.Pix[destIndex+3] = toUint8(pixels[sourceIndex+3])
			destIndex += 4
			sourceIndex += 4
		}
	}
}

func toUint8(v float32) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
