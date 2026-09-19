package image

import (
	"context"
	"errors"
	stdimage "image"

	"github.com/lewtec/lewkit/x/ndarray"
)

// Painter draws Triangle by resizing a Tensor. The kernel is compiled once.
type Painter struct {
	eval     ndarray.Evaluator
	triangle *ndarray.Tensor
	turn     *ndarray.Tensor
	width    *ndarray.Tensor
	height   *ndarray.Tensor
	h, w     int
}

// New compiles TriangleDynamic once. Open picks Vulkan if it can, else CPU.
func New(ctx context.Context) (*Painter, error) {
	turn, err := ndarray.New([]float32{0}, nil)
	if err != nil {
		return nil, err
	}
	width, err := ndarray.New([]float32{1}, nil)
	if err != nil {
		return nil, err
	}
	height, err := ndarray.New([]float32{1}, nil)
	if err != nil {
		return nil, err
	}
	triangle, err := TriangleDynamic(turn, width, height)
	if err != nil {
		return nil, err
	}
	eval, err := ndarray.Open(ctx)
	if err != nil {
		return nil, err
	}
	return &Painter{eval: eval, triangle: triangle, turn: turn, width: width, height: height}, nil
}

// Draw renders one frame into dst.
func (p *Painter) Draw(ctx context.Context, dst *stdimage.RGBA, turn float64) error {
	if p == nil || p.triangle == nil {
		return ndarray.ErrOp
	}
	h, w := dst.Rect.Dy(), dst.Rect.Dx()
	if h < 1 || w < 1 {
		return nil
	}
	if buf := p.turn.Buffer(); len(buf) > 0 {
		buf[0] = float32(turn)
	}
	if buf := p.width.Buffer(); len(buf) > 0 {
		buf[0] = float32(w)
	}
	if buf := p.height.Buffer(); len(buf) > 0 {
		buf[0] = float32(h)
	}
	if h != p.h || w != p.w {
		if err := p.triangle.Resize(ndarray.Shape{h, w, 4}); err != nil {
			return err
		}
		p.h, p.w = h, w
	}
	if err := p.triangle.Eval(ctx, p.eval); err != nil {
		return err
	}
	pixels, err := p.triangle.Data()
	if err != nil {
		return err
	}
	Write(dst, pixels)
	return nil
}

// Close releases the triangle kernel/session and device.
func (p *Painter) Close() error {
	if p == nil {
		return nil
	}
	var err error
	if p.triangle != nil {
		err = p.triangle.Close()
		p.triangle = nil
	}
	if p.eval != nil {
		err = errors.Join(err, p.eval.Close())
		p.eval = nil
	}
	return err
}
