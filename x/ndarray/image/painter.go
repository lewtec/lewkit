package image

import (
	"context"
	"encoding/binary"
	stdimage "image"
	"math"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
)

// Painter draws Triangle frames without reallocating the kernel or pixel buffer.
type Painter struct {
	h, w int
	k    *ndarray.Kernel
	pix  []float32
	turn []float32
	ins  [][]float32
	d    *vulkan.Device
	dst  *vulkan.Buffer
	src  *vulkan.Buffer
}

// Open compiles Triangle once. Vulkan is used when Open succeeds.
func Open(ctx context.Context, h, w int) (*Painter, error) {
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
	p := &Painter{h: h, w: w, k: k, pix: make([]float32, h*w*4), turn: []float32{0}}
	p.ins = [][]float32{p.turn}
	d, err := vulkan.Open(ctx)
	if err != nil {
		return p, nil
	}
	dst, err := d.Buffer(max(len(p.pix), 1) * 4)
	if err != nil {
		d.Close()
		return p, nil
	}
	src, err := d.Buffer(4)
	if err != nil {
		dst.Close()
		d.Close()
		return p, nil
	}
	p.d, p.dst, p.src = d, dst, src
	return p, nil
}

// Matches reports that p was compiled for h×w.
func (p *Painter) Matches(h, w int) bool {
	return p != nil && p.h == h && p.w == w
}

// Draw renders one frame into dst. dst must be h×w.
func (p *Painter) Draw(ctx context.Context, dst *stdimage.RGBA, turn float64) error {
	if p == nil || p.k == nil {
		return ndarray.ErrOp
	}
	p.turn[0] = float32(turn)
	if p.d != nil {
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], math.Float32bits(p.turn[0]))
		if err := p.src.Write(b[:]); err != nil {
			return err
		}
		if err := p.k.Run(ctx, p.d, p.dst, p.src); err != nil {
			return err
		}
		if err := p.dst.Read(f32bytes(p.pix)); err != nil {
			return err
		}
	} else if err := p.k.EvalInto(p.pix, p.ins); err != nil {
		return err
	}
	Write(dst, p.pix)
	return nil
}

func f32bytes(v []float32) []byte {
	if len(v) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(v))), len(v)*4)
}

// Close releases the kernel and any Vulkan buffers.
func (p *Painter) Close() error {
	if p == nil {
		return nil
	}
	var first error
	if p.src != nil {
		first = p.src.Close()
		p.src = nil
	}
	if p.dst != nil {
		if err := p.dst.Close(); first == nil {
			first = err
		}
		p.dst = nil
	}
	if p.k != nil {
		if err := p.k.Close(); first == nil {
			first = err
		}
		p.k = nil
	}
	if p.d != nil {
		if err := p.d.Close(); first == nil {
			first = err
		}
		p.d = nil
	}
	return first
}
