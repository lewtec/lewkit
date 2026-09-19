package image

import (
	"context"
	"errors"
	stdimage "image"

	"github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
)

// Painter draws Triangle by resizing a Session. The kernel is compiled once.
type Painter struct {
	device  *vulkan.Device
	kernel  *ndarray.Kernel
	session *ndarray.Session
	pixels  []float32
	turn    []float32
	width   []float32
	height  []float32
	inputs  [][]float32
}

// New compiles TriangleDynamic once and attaches a reusable session.
func New(ctx context.Context) (*Painter, error) {
	tracker, err := ndarray.Of(nil)
	if err != nil {
		return nil, err
	}
	expr, err := TriangleDynamic(ndarray.In(0, tracker), ndarray.In(1, tracker), ndarray.In(2, tracker))
	if err != nil {
		return nil, err
	}
	kernel, err := ndarray.Compile(expr)
	if err != nil {
		return nil, err
	}
	device, _ := vulkan.Open(ctx)
	session, err := kernel.Attach(ctx, device)
	if err != nil {
		e := kernel.Close()
		if device != nil {
			e = errors.Join(e, device.Close())
		}
		return nil, errors.Join(err, e)
	}
	p := &Painter{
		device: device, kernel: kernel, session: session,
		turn: []float32{0}, width: []float32{1}, height: []float32{1},
	}
	p.inputs = [][]float32{p.turn, p.width, p.height}
	return p, nil
}

// Draw renders one frame into dst.
func (p *Painter) Draw(ctx context.Context, dst *stdimage.RGBA, turn float64) error {
	if p == nil || p.session == nil || p.kernel == nil {
		return ndarray.ErrOp
	}
	h, w := dst.Rect.Dy(), dst.Rect.Dx()
	if h < 1 || w < 1 {
		return nil
	}
	if err := p.kernel.Resize(ndarray.Shape{h, w, 4}); err != nil {
		return err
	}
	n := h * w * 4
	if cap(p.pixels) < n {
		p.pixels = make([]float32, n)
	} else {
		p.pixels = p.pixels[:n]
	}
	p.turn[0] = float32(turn)
	p.width[0] = float32(w)
	p.height[0] = float32(h)
	if err := p.session.Run(ctx, p.pixels, p.inputs); err != nil {
		return err
	}
	Write(dst, p.pixels)
	return nil
}

// Close releases the session, kernel, and device.
func (p *Painter) Close() error {
	if p == nil {
		return nil
	}
	var err error
	if p.session != nil {
		err = p.session.Close()
		p.session = nil
	}
	if p.kernel != nil {
		if e := p.kernel.Close(); err == nil {
			err = e
		}
		p.kernel = nil
	}
	if p.device != nil {
		if e := p.device.Close(); err == nil {
			err = e
		}
		p.device = nil
	}
	return err
}
