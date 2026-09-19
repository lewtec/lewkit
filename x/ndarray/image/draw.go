package image

import (
	"fmt"
	stdimage "image"
	"math"

	"github.com/lewtec/lewkit/x/ndarray"
)

const (
	triAX, triAY = 0, -2.0 / 3.0
	triBX, triBY = 0.5, 1.0 / 3.0
	triCX, triCY = -0.5, 1.0 / 3.0
)

// Fill is a constant RGBA picture of size h×w.
func Fill(h, w int, r, g, b, a float32) (*ndarray.Node, error) {
	if h < 1 || w < 1 {
		return nil, ndarray.ErrShape
	}
	ch := ndarray.Coord(2, h, w, 4)
	v := ndarray.Eq(ch, ndarray.ConstI(0)).Where(ndarray.Const(r),
		ndarray.Eq(ch, ndarray.ConstI(1)).Where(ndarray.Const(g),
			ndarray.Eq(ch, ndarray.ConstI(2)).Where(ndarray.Const(b), ndarray.Const(a))))
	if v.Shape() == nil {
		return nil, ndarray.ErrOp
	}
	return v, nil
}

// Triangle is the RGB vulkan-tutorial triangle, optionally rotated by turn
// revolutions. turn is a splat (nil is 0). Output shape is (h, w, 4).
func Triangle(h, w int, turn *ndarray.Node) (*ndarray.Node, error) {
	if h < 1 || w < 1 {
		return nil, ndarray.ErrShape
	}
	if turn == nil {
		turn = ndarray.Const(0)
	}
	if turn.DType() != ndarray.F32 {
		return nil, ndarray.ErrType
	}
	if sh := turn.Shape(); sh != nil {
		return nil, fmt.Errorf("%w: turn must be a splat", ndarray.ErrShape)
	}
	shape := []int{h, w, 4}
	px := ndarray.Coord(1, shape...).Cast(ndarray.F32).Add(ndarray.Const(0.5))
	py := ndarray.Coord(0, shape...).Cast(ndarray.F32).Add(ndarray.Const(0.5))
	tau := turn.Mul(ndarray.Const(2 * math.Pi))
	sn := tau.Sin()
	cs := tau.Add(ndarray.Const(math.Pi / 2)).Sin()
	scale := ndarray.Const(float32(min(h, w)) / 2)
	ox := ndarray.Const(float32(w) / 2)
	oy := ndarray.Const(float32(h) / 2)
	fr := frame{sn, cs, scale, ox, oy}
	ax, ay := fr.rot(triAX, triAY)
	bx, by := fr.rot(triBX, triBY)
	cx, cy := fr.rot(triCX, triCY)
	den := by.Add(cy.Neg()).Mul(ax.Add(cx.Neg())).Add(cx.Add(bx.Neg()).Mul(ay.Add(cy.Neg())))
	u := ndarray.Div(
		by.Add(cy.Neg()).Mul(px.Add(cx.Neg())).Add(cx.Add(bx.Neg()).Mul(py.Add(cy.Neg()))),
		den,
	)
	v := ndarray.Div(
		cy.Add(ay.Neg()).Mul(px.Add(cx.Neg())).Add(ax.Add(cx.Neg()).Mul(py.Add(cy.Neg()))),
		den,
	)
	wt := ndarray.Const(1).Add(u.Neg()).Add(v.Neg())
	inside := ndarray.Ge(u, ndarray.Const(0)).And(ndarray.Ge(v, ndarray.Const(0))).And(ndarray.Ge(wt, ndarray.Const(0)))
	ch := ndarray.Coord(2, shape...)
	rgb := ndarray.Eq(ch, ndarray.ConstI(0)).Where(u.Mul(ndarray.Const(255)),
		ndarray.Eq(ch, ndarray.ConstI(1)).Where(v.Mul(ndarray.Const(255)),
			ndarray.Eq(ch, ndarray.ConstI(2)).Where(wt.Mul(ndarray.Const(255)), ndarray.Const(255))))
	out := inside.Where(rgb, ndarray.Eq(ch, ndarray.ConstI(3)).Where(ndarray.Const(255), ndarray.Const(0)))
	if out.Shape() == nil {
		return nil, ndarray.ErrOp
	}
	return out, nil
}

type frame struct {
	sn, cs, scale, ox, oy *ndarray.Node
}

func (f frame) rot(x, y float32) (px, py *ndarray.Node) {
	px = ndarray.Const(x).Mul(f.cs).Add(ndarray.Const(y).Mul(f.sn).Neg()).Mul(f.scale).Add(f.ox)
	py = ndarray.Const(x).Mul(f.sn).Add(ndarray.Const(y).Mul(f.cs)).Mul(f.scale).Add(f.oy)
	return px, py
}

// RGBA packs a dense (h, w, 4) float32 buffer into an image.
func RGBA(h, w int, pix []float32) *stdimage.RGBA {
	dst := stdimage.NewRGBA(stdimage.Rect(0, 0, w, h))
	if h < 1 || w < 1 || len(pix) < h*w*4 {
		return dst
	}
	for i := range h * w {
		o := i * 4
		dst.Pix[o] = u8(pix[o])
		dst.Pix[o+1] = u8(pix[o+1])
		dst.Pix[o+2] = u8(pix[o+2])
		dst.Pix[o+3] = u8(pix[o+3])
	}
	return dst
}

func u8(v float32) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
