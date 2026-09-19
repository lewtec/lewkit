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

func splatF32(n *ndarray.Node, name string) error {
	if n == nil || n.DType() != ndarray.F32 {
		return ndarray.ErrType
	}
	if sh := n.Shape(); len(sh) != 0 {
		return fmt.Errorf("%w: %s must be a splat", ndarray.ErrShape, name)
	}
	return nil
}

func minF32(a, b *ndarray.Node) *ndarray.Node {
	return ndarray.CmpLt(a, b).Where(a, b)
}

// Triangle is the RGB vulkan-tutorial triangle. turn is revolutions (nil is 0).
// Output shape is (h, w, 4).
func Triangle(h, w int, turn *ndarray.Node) (*ndarray.Node, error) {
	if h < 1 || w < 1 {
		return nil, ndarray.ErrShape
	}
	if turn == nil {
		turn = ndarray.Const(0)
	}
	return triangle(turn, ndarray.Const(float32(w)), ndarray.Const(float32(h)), []int{h, w, 4})
}

// TriangleDyn is Triangle with runtime width/height splats. Coord rank is (1,1,4);
// the caller sets the real (h,w,4) with Kernel.Resize before Eval/Run.
func TriangleDyn(turn, width, height *ndarray.Node) (*ndarray.Node, error) {
	return triangle(turn, width, height, []int{1, 1, 4})
}

func triangle(turn, width, height *ndarray.Node, shape []int) (*ndarray.Node, error) {
	if err := splatF32(turn, "turn"); err != nil {
		return nil, err
	}
	if err := splatF32(width, "width"); err != nil {
		return nil, err
	}
	if err := splatF32(height, "height"); err != nil {
		return nil, err
	}
	px := ndarray.Coord(1, shape...).Cast(ndarray.F32).Add(ndarray.Const(0.5))
	py := ndarray.Coord(0, shape...).Cast(ndarray.F32).Add(ndarray.Const(0.5))
	tau := turn.Mul(ndarray.Const(2 * math.Pi))
	sn := tau.Sin()
	cs := tau.Add(ndarray.Const(math.Pi / 2)).Sin()
	scale := minF32(width, height).Mul(ndarray.Const(0.5))
	ox := width.Mul(ndarray.Const(0.5))
	oy := height.Mul(ndarray.Const(0.5))
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

// RGBA packs a dense (h, w, 4) float32 buffer into a new image.
func RGBA(h, w int, pix []float32) *stdimage.RGBA {
	dst := stdimage.NewRGBA(stdimage.Rect(0, 0, w, h))
	Write(dst, pix)
	return dst
}

// Write packs pix (h, w, 4) float32 into dst. dst's bounds set h and w.
func Write(dst *stdimage.RGBA, pix []float32) {
	if dst == nil {
		return
	}
	w, h := dst.Rect.Dx(), dst.Rect.Dy()
	if w < 1 || h < 1 || len(pix) < h*w*4 {
		return
	}
	for y := range h {
		di := dst.PixOffset(dst.Rect.Min.X, dst.Rect.Min.Y+y)
		si := y * w * 4
		for range w {
			dst.Pix[di] = u8(pix[si])
			dst.Pix[di+1] = u8(pix[si+1])
			dst.Pix[di+2] = u8(pix[si+2])
			dst.Pix[di+3] = u8(pix[si+3])
			di += 4
			si += 4
		}
	}
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
