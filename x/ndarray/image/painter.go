package image

import (
	"context"
	"errors"
	stdimage "image"

	"github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
)

// Painter draws Triangle by resizing a Tensor. The kernel is compiled once.
type Painter struct {
	device   *vulkan.Device
	triangle *ndarray.Tensor
	turn     *ndarray.Tensor
	width    *ndarray.Tensor
	height   *ndarray.Tensor
}

// New compiles TriangleDynamic once and attaches a reusable session.
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
	device, _ := vulkan.Open(ctx)
	return &Painter{device: device, triangle: triangle, turn: turn, width: width, height: height}, nil
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
	if err := p.triangle.Resize(ndarray.Shape{h, w, 4}); err != nil {
		return err
	}
	var err error
	if p.device != nil {
		err = p.triangle.Exec(ctx, p.device)
	} else {
		err = p.triangle.Eval()
	}
	if err != nil {
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
	if p.device != nil {
		err = errors.Join(err, p.device.Close())
		p.device = nil
	}
	return err
}
