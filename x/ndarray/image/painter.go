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

// Painter draws Triangle frames from one compiled kernel. Size is a runtime
// push constant; glslang is instantiated once.
type Painter struct {
	d      *vulkan.Device
	k      *ndarray.Kernel
	pix    []float32
	turn   []float32
	width  []float32
	height []float32
	ins    [][]float32
	out    *vulkan.Buffer
	srcT   *vulkan.Buffer
	srcW   *vulkan.Buffer
	srcH   *vulkan.Buffer
}

// New compiles Triangle once and opens Vulkan when available.
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
	p := &Painter{
		k:      k,
		turn:   []float32{0},
		width:  []float32{1},
		height: []float32{1},
	}
	p.ins = [][]float32{p.turn, p.width, p.height}
	d, err := vulkan.Open(ctx)
	if err != nil {
		return p, nil
	}
	t, err := d.Buffer(4)
	if err != nil {
		d.Close()
		return p, nil
	}
	bw, err := d.Buffer(4)
	if err != nil {
		t.Close()
		d.Close()
		return p, nil
	}
	bh, err := d.Buffer(4)
	if err != nil {
		bw.Close()
		t.Close()
		d.Close()
		return p, nil
	}
	p.d, p.srcT, p.srcW, p.srcH = d, t, bw, bh
	return p, nil
}

// Draw renders one frame into dst.
func (p *Painter) Draw(ctx context.Context, dst *stdimage.RGBA, turn float64) error {
	if p == nil || p.k == nil {
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
	if p.d != nil && p.srcT != nil {
		if err := p.ensureOut(n * 4); err != nil {
			return err
		}
		if err := writeF32(p.srcT, p.turn[0]); err != nil {
			return err
		}
		if err := writeF32(p.srcW, p.width[0]); err != nil {
			return err
		}
		if err := writeF32(p.srcH, p.height[0]); err != nil {
			return err
		}
		if err := p.k.Run(ctx, p.d, p.out, p.srcT, p.srcW, p.srcH); err != nil {
			return err
		}
		if err := p.out.Read(asBytes(p.pix)); err != nil {
			return err
		}
	} else if err := p.k.EvalInto(p.pix, p.ins); err != nil {
		return err
	}
	Write(dst, p.pix)
	return nil
}

func (p *Painter) ensureOut(bytes int) error {
	if p.out != nil && p.out.Len() >= bytes {
		return nil
	}
	b, err := p.d.Buffer(bytes)
	if err != nil {
		return err
	}
	if p.out != nil {
		if err := p.out.Close(); err != nil {
			return errors.Join(err, b.Close())
		}
	}
	p.out = b
	return nil
}

func writeF32(b *vulkan.Buffer, v float32) error {
	var raw [4]byte
	binary.LittleEndian.PutUint32(raw[:], math.Float32bits(v))
	return b.Write(raw[:])
}

func asBytes(v []float32) []byte {
	if len(v) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(v))), len(v)*4)
}

// Close releases the kernel, buffers, and device.
func (p *Painter) Close() error {
	if p == nil {
		return nil
	}
	var err error
	for _, b := range []*vulkan.Buffer{p.srcT, p.srcW, p.srcH, p.out} {
		if b == nil {
			continue
		}
		if e := b.Close(); err == nil {
			err = e
		}
	}
	p.srcT, p.srcW, p.srcH, p.out = nil, nil, nil, nil
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
