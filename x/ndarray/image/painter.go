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
	d      *vulkan.Device
	k      *ndarray.Kernel
	sess   *ndarray.Session
	pix    []float32
	turn   []float32
	width  []float32
	height []float32
	ins    [][]float32
}

// New compiles TriangleDyn once and attaches a reusable session.
func New(ctx context.Context) (*Painter, error) {
	st, err := ndarray.Of()
	if err != nil {
		return nil, err
	}
	expr, err := TriangleDyn(ndarray.In(0, st), ndarray.In(1, st), ndarray.In(2, st))
	if err != nil {
		return nil, err
	}
	k, err := ndarray.Compile(expr)
	if err != nil {
		return nil, err
	}
	d, _ := vulkan.Open(ctx)
	sess, err := k.Attach(ctx, d)
	if err != nil {
		e := k.Close()
		if d != nil {
			e = errors.Join(e, d.Close())
		}
		return nil, errors.Join(err, e)
	}
	p := &Painter{
		d: d, k: k, sess: sess,
		turn: []float32{0}, width: []float32{1}, height: []float32{1},
	}
	p.ins = [][]float32{p.turn, p.width, p.height}
	return p, nil
}

// Draw renders one frame into dst.
func (p *Painter) Draw(ctx context.Context, dst *stdimage.RGBA, turn float64) error {
	if p == nil || p.sess == nil || p.k == nil {
		return ndarray.ErrOp
	}
	h, w := dst.Rect.Dy(), dst.Rect.Dx()
	if h < 1 || w < 1 {
		return nil
	}
	if err := p.k.Resize([]int{h, w, 4}); err != nil {
		return err
	}
	n := h * w * 4
	if cap(p.pix) < n {
		p.pix = make([]float32, n)
	} else {
		p.pix = p.pix[:n]
	}
	p.turn[0] = float32(turn)
	p.width[0] = float32(w)
	p.height[0] = float32(h)
	if err := p.sess.Run(ctx, p.pix, p.ins); err != nil {
		return err
	}
	Write(dst, p.pix)
	return nil
}

// Close releases the session, kernel, and device.
func (p *Painter) Close() error {
	if p == nil {
		return nil
	}
	var err error
	if p.sess != nil {
		err = p.sess.Close()
		p.sess = nil
	}
	if p.k != nil {
		if e := p.k.Close(); err == nil {
			err = e
		}
		p.k = nil
	}
	if p.d != nil {
		if e := p.d.Close(); err == nil {
			err = e
		}
		p.d = nil
	}
	return err
}
