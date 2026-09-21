package gui

import (
	"image"
	"math"

	"github.com/lewtec/lewkit/x/ndarray"
)

type slot struct {
	x, y, width, height     *ndarray.Tensor[float32]
	red, green, blue, alpha *ndarray.Tensor[float32]
	radius                  *ndarray.Tensor[float32]
	clipX, clipY            *ndarray.Tensor[float32]
	clipWidth, clipHeight   *ndarray.Tensor[float32]
}

// Picture is one over-composite tensor: each fill layers onto acc, then ink.
// The graph grows with the tree; uniforms update each frame.
type Picture struct {
	slots     []slot
	outs      []*ndarray.Tensor[float32]
	base      *ndarray.Tensor[float32]
	acc       *ndarray.Tensor[float32]
	px, py    *ndarray.Tensor[float32]
	ch        *ndarray.Tensor[int32]
	params    *ndarray.Tensor[float32]
	ink       *ndarray.Tensor[uint8]
	pixels    *ndarray.Tensor[uint8]
	inkedFrom *ndarray.Tensor[float32]
	inkRGBA   *image.RGBA
	fills     []Draw
	texts     []textRun
	sig       uint64
	hadInk    bool
	used      int
}

func accOf(pic *Picture) *ndarray.Tensor[float32] {
	if pic == nil {
		return nil
	}
	return pic.acc
}

const slotFloats = 13

func (p *Picture) cell(i int) (*ndarray.Tensor[float32], error) {
	t, err := p.params.Shrink([][2]int{{i, i + 1}})
	if err != nil {
		return nil, err
	}
	return t.Splat()
}

func (p *Picture) newSlot(base int) (slot, error) {
	var s slot
	fs := [slotFloats]**ndarray.Tensor[float32]{
		&s.x, &s.y, &s.width, &s.height, &s.red, &s.green, &s.blue, &s.alpha, &s.radius, &s.clipX, &s.clipY, &s.clipWidth, &s.clipHeight,
	}
	for j, dst := range fs {
		t, err := p.cell(base + j)
		if err != nil {
			return s, err
		}
		*dst = t
	}
	return s, nil
}

func (s slot) write(d Draw) {
	put(s.x, d.X)
	put(s.y, d.Y)
	put(s.width, d.Width)
	put(s.height, d.Height)
	put(s.red, d.Red)
	put(s.green, d.Green)
	put(s.blue, d.Blue)
	put(s.alpha, d.Alpha)
	put(s.radius, d.Radius)
	put(s.clipX, d.ClipX)
	put(s.clipY, d.ClipY)
	put(s.clipWidth, d.ClipWidth)
	put(s.clipHeight, d.ClipHeight)
}

func put(t *ndarray.Tensor[float32], v float32) {
	if t == nil {
		return
	}
	buf := t.Buffer()
	off, ok, err := t.Tracker().At(0)
	if err != nil || !ok || off < 0 || off >= len(buf) {
		return
	}
	buf[off] = v
}

// NewPicture starts from opaque black. Fills grow the over-composite as Paint returns.
func NewPicture() (*Picture, error) {
	p := &Picture{}
	if err := p.init(); err != nil {
		return nil, err
	}
	p.pixels = p.withInk(p.base)
	p.inkedFrom = p.base
	return p, nil
}

func (p *Picture) init() error {
	ink, err := ndarray.New(make([]uint8, 4), ndarray.Shape{1, 1, 4})
	if err != nil {
		return err
	}
	shape := ndarray.Shape{1, 1, 4}
	p.px = ndarray.Coord(1, shape).Cast[float32]().Add(ndarray.Const(float32(0.5)))
	p.py = ndarray.Coord(0, shape).Cast[float32]().Add(ndarray.Const(float32(0.5)))
	p.ch = ndarray.Coord(2, shape)
	p.base = channelColor(p.ch, ndarray.Const(float32(0)), ndarray.Const(float32(0)), ndarray.Const(float32(0)), ndarray.Const(float32(255)))
	p.ink = ink
	p.acc = p.base
	return nil
}

func (p *Picture) withInk(acc *ndarray.Tensor[float32]) *ndarray.Tensor[uint8] {
	lit := p.ink.Cast[int32]().CmpNe(ndarray.Const(int32(0)))
	return lit.Where(p.ink.Cast[float32](), acc).Cast[uint8]()
}

func (p *Picture) glyph(run textRun) {
	p.texts = append(p.texts, run)
}

func (p *Picture) over(d Draw) *ndarray.Tensor[float32] {
	if p == nil {
		return nil
	}
	p.fills = append(p.fills, d)
	i := len(p.fills) - 1
	if i >= len(p.slots) {
		cap := len(p.slots)
		if cap < 1 {
			cap = 1
		}
		for cap <= i {
			cap *= 2
		}
		if err := p.compile(cap); err != nil {
			return p.acc
		}
		for j := 0; j < i; j++ {
			p.slots[j].write(p.fills[j])
		}
	}
	p.slots[i].write(d)
	p.used = i + 1
	p.acc = p.outs[i]
	return p.acc
}

func (p *Picture) compile(n int) error {
	if p == nil || n < 1 {
		return ndarray.ErrShape
	}
	params, err := ndarray.New(make([]float32, n*slotFloats), ndarray.Shape{n * slotFloats})
	if err != nil {
		return err
	}
	p.params = params
	p.slots = make([]slot, n)
	p.outs = make([]*ndarray.Tensor[float32], n)
	acc := p.base
	for i := 0; i < n; i++ {
		sl, err := p.newSlot(i * slotFloats)
		if err != nil {
			return err
		}
		p.slots[i] = sl
		cov := sl.coverage(p.px, p.py)
		alpha := cov.Mul(sl.alpha.Mul(ndarray.Const(float32(1.0 / 255))))
		color := channelColor(p.ch, sl.red, sl.green, sl.blue, ndarray.Const(float32(255)))
		acc = acc.Add(alpha.Mul(color.Add(acc.Neg())))
		p.outs[i] = acc
	}
	p.inkedFrom = nil
	return nil
}

func channelColor(ch *ndarray.Tensor[int32], r, g, b, a *ndarray.Tensor[float32]) *ndarray.Tensor[float32] {
	return ch.Equal(ndarray.Const(int32(0))).Where(r,
		ch.Equal(ndarray.Const(int32(1))).Where(g,
			ch.Equal(ndarray.Const(int32(2))).Where(b, a)))
}

func (s slot) coverage(px, py *ndarray.Tensor[float32]) *ndarray.Tensor[float32] {
	half := ndarray.Const(float32(0.5))
	zero := ndarray.Const(float32(0))
	lx := px.Add(s.x.Neg()).Add(half).Add(s.width.Mul(half).Neg())
	ly := py.Add(s.y.Neg()).Add(half).Add(s.height.Mul(half).Neg())
	bx := s.width.Mul(half)
	by := s.height.Mul(half)
	rad := s.radius.CmpLt(bx).Where(s.radius, bx)
	rad = rad.CmpLt(by).Where(rad, by)
	qx := lx.Max(lx.Neg()).Add(bx.Neg()).Add(rad)
	qy := ly.Max(ly.Neg()).Add(by.Neg()).Add(rad)
	outsideX := qx.Max(zero)
	outsideY := qy.Max(zero)
	qmax := qx.Max(qy)
	inside := qmax.CmpLt(zero).Where(qmax, zero)
	dist := inside.Add(outsideX.Mul(outsideX).Add(outsideY.Mul(outsideY)).Sqrt()).Add(rad.Neg())
	cover := dist.CmpLt(half).Where(ndarray.Const(float32(1)), zero)
	inX := px.GreaterEqual(s.clipX).And(px.CmpLt(s.clipX.Add(s.clipWidth)))
	inY := py.GreaterEqual(s.clipY).And(py.CmpLt(s.clipY.Add(s.clipHeight)))
	clipOn := s.clipWidth.GreaterEqual(half)
	clipMask := clipOn.Where(inX.And(inY), ndarray.Const(int32(1)))
	return clipMask.Where(cover, zero)
}

// Render layouts root, then Paint returns the over-composite tensor.
func (p *Picture) Render(root Node, size Size) (*ndarray.Tensor[uint8], error) {
	if p == nil || p.base == nil {
		return nil, ErrView
	}
	if root == nil || size.Width < 1 || size.Height < 1 {
		return nil, ndarray.ErrShape
	}
	root.Layout(Tight(size.Width, size.Height))
	p.used = 0
	p.fills = p.fills[:0]
	p.texts = p.texts[:0]
	p.acc = p.base
	acc := root.Paint(Offset{}, Rect{0, 0, size.Width, size.Height}, p)
	if acc == nil {
		acc = p.base
	}
	if p.pixels == nil || p.inkedFrom != acc {
		if p.pixels != nil {
			_ = p.pixels.Close()
		}
		p.pixels = p.withInk(acc)
		p.inkedFrom = acc
	}
	h, w := int(size.Height), int(size.Width)
	if err := p.pixels.Resize(ndarray.Shape{h, w, 4}); err != nil {
		return nil, err
	}
	if err := p.ensureInk(h, w, len(p.texts)); err != nil {
		return nil, err
	}
	for _, run := range p.texts {
		run.stamp(p.inkRGBA)
	}
	p.stamp(len(p.texts) > 0)
	return p.pixels, nil
}

func (p *Picture) frameSig() uint64 {
	if p == nil {
		return 0
	}
	return p.sig
}

func (p *Picture) stamp(ink bool) {
	if p == nil {
		return
	}
	h := uint64(14695981039346656037)
	mix := func(v uint64) {
		h ^= v
		h *= 1099511628211
	}
	if p.params != nil {
		for _, f := range p.params.Buffer() {
			mix(uint64(math.Float32bits(f)))
		}
	}
	if p.pixels != nil {
		for _, d := range p.pixels.Shape() {
			mix(uint64(d))
		}
	}
	if ink && p.inkRGBA != nil {
		for _, b := range p.inkRGBA.Pix {
			mix(uint64(b))
		}
	}
	p.sig = h
}

func (p *Picture) ensureInk(h, w, texts int) error {
	if p == nil || p.ink == nil || h < 1 || w < 1 {
		return ndarray.ErrShape
	}
	need := h * w * 4
	if err := p.ink.EnsureCells(need); err != nil {
		return err
	}
	pix := p.ink.Buffer()[:need]
	same := p.inkRGBA != nil && p.inkRGBA.Rect.Dx() == w && p.inkRGBA.Rect.Dy() == h
	if texts > 0 || p.hadInk || !same {
		clear(pix)
	}
	p.hadInk = texts > 0
	p.inkRGBA = &image.RGBA{Pix: pix, Stride: w * 4, Rect: image.Rect(0, 0, w, h)}
	return nil
}

// Ink is the CPU glyph overlay for this frame. Call after Render.
func (p *Picture) Ink() *image.RGBA {
	if p == nil {
		return nil
	}
	return p.inkRGBA
}
