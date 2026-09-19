package window

import (
	"context"
	"image"

	"github.com/lewtec/lewkit/x/ndarray"
)

// PixelShape is the (height, width, 4) layout Frame stores.
func PixelShape(dst *image.RGBA) (ndarray.Shape, error) {
	if dst == nil {
		return nil, ErrClosed
	}
	return Shape(dst.Rect.Size())
}

// Shape is the (height, width, 4) tensor layout for a window Size.
func Shape(size image.Point) (ndarray.Shape, error) {
	if size.X < 1 || size.Y < 1 {
		return nil, ErrSize
	}
	return ndarray.Shape{size.Y, size.X, 4}, nil
}

func fit(t *ndarray.Tensor[uint8], size image.Point) (*ndarray.Tensor[uint8], error) {
	if t == nil {
		return nil, ndarray.ErrOp
	}
	shape, err := Shape(size)
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
		return fit(pixels, dst.Rect.Size())
	}
	return fit(t.Cast[uint8](), dst.Rect.Size())
}

// Fit resizes t to this window's Size and casts to uint8.
func (b *Buffer) Fit[T ndarray.Number](t *ndarray.Tensor[T]) (*ndarray.Tensor[uint8], error) {
	if b == nil {
		return nil, ErrClosed
	}
	if pixels, ok := any(t).(*ndarray.Tensor[uint8]); ok {
		return fit(pixels, b.Size())
	}
	return fit(t.Cast[uint8](), b.Size())
}

// Present fits t to dst and evals into dst's pixels.
func Present(ctx context.Context, t *ndarray.Tensor[uint8], evaluator ndarray.Evaluator, dst *image.RGBA) error {
	if dst == nil {
		return ErrClosed
	}
	pixels, err := fit(t, dst.Rect.Size())
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
