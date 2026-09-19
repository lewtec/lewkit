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
	channel := ndarray.Coord(2, ndarray.Shape{h, w, 4})
	v := ndarray.Equal(channel, ndarray.ConstInt(0)).Where(ndarray.Const(r),
		ndarray.Equal(channel, ndarray.ConstInt(1)).Where(ndarray.Const(g),
			ndarray.Equal(channel, ndarray.ConstInt(2)).Where(ndarray.Const(b), ndarray.Const(a))))
	if v.Shape() == nil {
		return nil, ndarray.ErrOp
	}
	return v, nil
}

func requireFloatSplat(n *ndarray.Node, name string) error {
	if n == nil || n.DType() != ndarray.F32 {
		return ndarray.ErrType
	}
	if shape := n.Shape(); len(shape) != 0 {
		return fmt.Errorf("%w: %s must be a splat", ndarray.ErrShape, name)
	}
	return nil
}

func minFloat(a, b *ndarray.Node) *ndarray.Node {
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
	return triangle(turn, ndarray.Const(float32(w)), ndarray.Const(float32(h)), ndarray.Shape{h, w, 4})
}

// TriangleDynamic is Triangle with runtime width/height splats. Coord rank is (1,1,4);
// the caller sets the real (h,w,4) with Kernel.Resize before Eval/Run.
func TriangleDynamic(turn, width, height *ndarray.Node) (*ndarray.Node, error) {
	return triangle(turn, width, height, ndarray.Shape{1, 1, 4})
}

func triangle(turn, width, height *ndarray.Node, shape ndarray.Shape) (*ndarray.Node, error) {
	if err := requireFloatSplat(turn, "turn"); err != nil {
		return nil, err
	}
	if err := requireFloatSplat(width, "width"); err != nil {
		return nil, err
	}
	if err := requireFloatSplat(height, "height"); err != nil {
		return nil, err
	}
	px := ndarray.Coord(1, shape).Cast(ndarray.F32).Add(ndarray.Const(0.5))
	py := ndarray.Coord(0, shape).Cast(ndarray.F32).Add(ndarray.Const(0.5))
	tau := turn.Mul(ndarray.Const(2 * math.Pi))
	sine := tau.Sin()
	cosine := tau.Add(ndarray.Const(math.Pi / 2)).Sin()
	scale := minFloat(width, height).Mul(ndarray.Const(0.5))
	originX := width.Mul(ndarray.Const(0.5))
	originY := height.Mul(ndarray.Const(0.5))
	fr := frame{sine, cosine, scale, originX, originY}
	ax, ay := fr.rotate(triAX, triAY)
	bx, by := fr.rotate(triBX, triBY)
	cx, cy := fr.rotate(triCX, triCY)
	den := by.Add(cy.Neg()).Mul(ax.Add(cx.Neg())).Add(cx.Add(bx.Neg()).Mul(ay.Add(cy.Neg())))
	u := ndarray.Div(
		by.Add(cy.Neg()).Mul(px.Add(cx.Neg())).Add(cx.Add(bx.Neg()).Mul(py.Add(cy.Neg()))),
		den,
	)
	v := ndarray.Div(
		cy.Add(ay.Neg()).Mul(px.Add(cx.Neg())).Add(ax.Add(cx.Neg()).Mul(py.Add(cy.Neg()))),
		den,
	)
	weight := ndarray.Const(1).Add(u.Neg()).Add(v.Neg())
	inside := ndarray.GreaterEqual(u, ndarray.Const(0)).And(ndarray.GreaterEqual(v, ndarray.Const(0))).And(ndarray.GreaterEqual(weight, ndarray.Const(0)))
	channel := ndarray.Coord(2, shape)
	rgb := ndarray.Equal(channel, ndarray.ConstInt(0)).Where(u.Mul(ndarray.Const(255)),
		ndarray.Equal(channel, ndarray.ConstInt(1)).Where(v.Mul(ndarray.Const(255)),
			ndarray.Equal(channel, ndarray.ConstInt(2)).Where(weight.Mul(ndarray.Const(255)), ndarray.Const(255))))
	out := inside.Where(rgb, ndarray.Equal(channel, ndarray.ConstInt(3)).Where(ndarray.Const(255), ndarray.Const(0)))
	if out.Shape() == nil {
		return nil, ndarray.ErrOp
	}
	return out, nil
}

type frame struct {
	sine, cosine, scale, originX, originY *ndarray.Node
}

func (f frame) rotate(x, y float32) (px, py *ndarray.Node) {
	px = ndarray.Const(x).Mul(f.cosine).Add(ndarray.Const(y).Mul(f.sine).Neg()).Mul(f.scale).Add(f.originX)
	py = ndarray.Const(x).Mul(f.sine).Add(ndarray.Const(y).Mul(f.cosine)).Mul(f.scale).Add(f.originY)
	return px, py
}

// RGBA packs a dense (h, w, 4) float32 buffer into a new image.
func RGBA(h, w int, pixels []float32) *stdimage.RGBA {
	dst := stdimage.NewRGBA(stdimage.Rect(0, 0, w, h))
	Write(dst, pixels)
	return dst
}

// Write packs pixels (h, w, 4) float32 into dst. dst's bounds set h and w.
func Write(dst *stdimage.RGBA, pixels []float32) {
	if dst == nil {
		return
	}
	w, h := dst.Rect.Dx(), dst.Rect.Dy()
	if w < 1 || h < 1 || len(pixels) < h*w*4 {
		return
	}
	for y := range h {
		di := dst.PixOffset(dst.Rect.Min.X, dst.Rect.Min.Y+y)
		si := y * w * 4
		for range w {
			dst.Pix[di] = toUint8(pixels[si])
			dst.Pix[di+1] = toUint8(pixels[si+1])
			dst.Pix[di+2] = toUint8(pixels[si+2])
			dst.Pix[di+3] = toUint8(pixels[si+3])
			di += 4
			si += 4
		}
	}
}

func toUint8(v float32) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
