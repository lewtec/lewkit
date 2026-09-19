package window

import (
	"context"
	"image"

	"github.com/lewtec/lewkit/x/ndarray"
)

// PixelShape is the (height, width, 4) layout Frame stores.
func PixelShape(dst *image.RGBA) (ndarray.Shape, error) {
	height, width, err := pixelSize(dst)
	if err != nil {
		return nil, err
	}
	return ndarray.Shape{height, width, 4}, nil
}

func pixelSize(dst *image.RGBA) (height, width int, err error) {
	if dst == nil {
		return 0, 0, ErrClosed
	}
	height, width = dst.Rect.Dy(), dst.Rect.Dx()
	if height < 1 || width < 1 {
		return 0, 0, ErrSize
	}
	return height, width, nil
}

func fit(t *ndarray.Tensor[uint8], dst *image.RGBA) (*ndarray.Tensor[uint8], error) {
	if t == nil {
		return nil, ndarray.ErrOp
	}
	shape, err := PixelShape(dst)
	if err != nil {
		return nil, err
	}
	if err := t.Resize(shape); err != nil {
		return nil, err
	}
	return t, nil
}

// Fit resizes t to dst (h, w, 4) and casts to uint8. That is what Frame
// stores (Go RGBA). If t is already uint8, the same tensor is resized
// and returned.
func Fit[T ndarray.Number](t *ndarray.Tensor[T], dst *image.RGBA) (*ndarray.Tensor[uint8], error) {
	if t == nil {
		return nil, ndarray.ErrOp
	}
	if pixels, ok := any(t).(*ndarray.Tensor[uint8]); ok {
		return fit(pixels, dst)
	}
	return fit(t.Cast[uint8](), dst)
}

// Fit resizes t to this buffer's Frame and casts to uint8.
func (b *Buffer) Fit[T ndarray.Number](t *ndarray.Tensor[T]) (*ndarray.Tensor[uint8], error) {
	if b == nil {
		return nil, ErrClosed
	}
	return Fit(t, b.Frame())
}

// Present fits t to dst and evals into dst's pixels.
func Present(ctx context.Context, t *ndarray.Tensor[uint8], evaluator ndarray.Evaluator, dst *image.RGBA) error {
	pixels, err := fit(t, dst)
	if err != nil {
		return err
	}
	width, height := dst.Rect.Dx(), dst.Rect.Dy()
	size := height * width * 4
	if dst.Stride == width*4 && len(dst.Pix) >= size {
		return pixels.Eval(ctx, evaluator, dst.Pix[:size])
	}
	buffer := make([]uint8, size)
	if err := pixels.Eval(ctx, evaluator, buffer); err != nil {
		return err
	}
	n := width * 4
	for y := 0; y < dst.Rect.Dy(); y++ {
		destIndex := dst.PixOffset(dst.Rect.Min.X, dst.Rect.Min.Y+y)
		copy(dst.Pix[destIndex:destIndex+n], buffer[y*n:y*n+n])
	}
	return nil
}

// Present fits t to this buffer's Frame and evals into it.
func (b *Buffer) Present(ctx context.Context, t *ndarray.Tensor[uint8], evaluator ndarray.Evaluator) error {
	if b == nil {
		return ErrClosed
	}
	return Present(ctx, t, evaluator, b.Frame())
}
