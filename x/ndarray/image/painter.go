package image

import (
	"context"
	"encoding/binary"
	"errors"
	stdimage "image"
	"math"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
)

// Painter draws Triangle frames. One Vulkan device is opened for the
// lifetime; kernels are rebuilt only when the pixel size changes.
type Painter struct {
	d    *vulkan.Device
	cur  *slot
	prev *slot
	turn []float32
	ins  [][]float32
}

type slot struct {
	h, w int
	k    *ndarray.Kernel
	pix  []float32
	dst  *vulkan.Buffer
	src  *vulkan.Buffer
}

// New opens an optional Vulkan device. Draw compiles the first size on demand.
func New(ctx context.Context) (*Painter, error) {
	p := &Painter{turn: []float32{0}}
	p.ins = [][]float32{p.turn}
	d, err := vulkan.Open(ctx)
	if err == nil {
		p.d = d
	}
	return p, nil
}

// Draw renders one frame into dst.
func (p *Painter) Draw(ctx context.Context, dst *stdimage.RGBA, turn float64) error {
	if p == nil {
		return ndarray.ErrOp
	}
	h, w := dst.Rect.Dy(), dst.Rect.Dx()
	if h < 1 || w < 1 {
		return nil
	}
	if err := p.ensure(ctx, h, w); err != nil {
		return err
	}
	s := p.cur
	p.turn[0] = float32(turn)
	if p.d != nil && s.dst != nil {
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], math.Float32bits(p.turn[0]))
		if err := s.src.Write(b[:]); err != nil {
			return err
		}
		if err := s.k.Run(ctx, p.d, s.dst, s.src); err != nil {
			return err
		}
		if err := s.dst.Read(asBytes(s.pix)); err != nil {
			return err
		}
	} else if err := s.k.EvalInto(s.pix, p.ins); err != nil {
		return err
	}
	Write(dst, s.pix)
	return nil
}

func (p *Painter) ensure(ctx context.Context, h, w int) error {
	if p.cur != nil && p.cur.h == h && p.cur.w == w {
		return nil
	}
	if p.prev != nil && p.prev.h == h && p.prev.w == w {
		p.cur, p.prev = p.prev, p.cur
		return nil
	}
	s, err := p.build(ctx, h, w)
	if err != nil {
		return err
	}
	if p.prev != nil {
		if err := p.prev.close(); err != nil {
			return errors.Join(err, s.close())
		}
	}
	p.prev = p.cur
	p.cur = s
	return nil
}

func (p *Painter) build(ctx context.Context, h, w int) (*slot, error) {
	st, err := ndarray.Of()
	if err != nil {
		return nil, err
	}
	expr, err := Triangle(h, w, ndarray.In(0, st))
	if err != nil {
		return nil, err
	}
	k, err := ndarray.Compile(expr)
	if err != nil {
		return nil, err
	}
	s := &slot{h: h, w: w, k: k, pix: make([]float32, h*w*4)}
	if p.d == nil {
		return s, nil
	}
	dst, err := p.d.Buffer(max(len(s.pix), 1) * 4)
	if err != nil {
		return nil, errors.Join(err, k.Close())
	}
	src, err := p.d.Buffer(4)
	if err != nil {
		return nil, errors.Join(err, dst.Close(), k.Close())
	}
	s.dst, s.src = dst, src
	return s, nil
}

func (s *slot) close() error {
	if s == nil {
		return nil
	}
	var err error
	if s.src != nil {
		err = s.src.Close()
		s.src = nil
	}
	if s.dst != nil {
		if e := s.dst.Close(); err == nil {
			err = e
		}
		s.dst = nil
	}
	if s.k != nil {
		if e := s.k.Close(); err == nil {
			err = e
		}
		s.k = nil
	}
	return err
}

func asBytes(v []float32) []byte {
	if len(v) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(v))), len(v)*4)
}

// Close releases kernels, buffers, and the device.
func (p *Painter) Close() error {
	if p == nil {
		return nil
	}
	var err error
	if p.prev != nil {
		err = p.prev.close()
		p.prev = nil
	}
	if p.cur != nil {
		if e := p.cur.close(); err == nil {
			err = e
		}
		p.cur = nil
	}
	if p.d != nil {
		if e := p.d.Close(); err == nil {
			err = e
		}
		p.d = nil
	}
	return err
}
