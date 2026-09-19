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
func Fill(h, w int, r, g, b, a float32) (*ndarray.Tensor, error) {
	if h < 1 || w < 1 {
		return nil, ndarray.ErrShape
	}
	channel := ndarray.Coord(2, ndarray.Shape{h, w, 4})
	v := channel.Equal(ndarray.ConstInt(0)).Where(ndarray.Const(r),
		channel.Equal(ndarray.ConstInt(1)).Where(ndarray.Const(g),
			channel.Equal(ndarray.ConstInt(2)).Where(ndarray.Const(b), ndarray.Const(a))))
	if v.Shape() == nil {
		return nil, ndarray.ErrOp
	}
	return v, nil
}

func requireFloatSplat(n *ndarray.Tensor, name string) error {
	if n == nil || n.DType() != ndarray.F32 {
		return ndarray.ErrType
	}
	if shape := n.Shape(); len(shape) != 0 {
		return fmt.Errorf("%w: %s must be a splat", ndarray.ErrShape, name)
	}
	return nil
}

func minFloat(a, b *ndarray.Tensor) *ndarray.Tensor {
	return a.CmpLt(b).Where(a, b)
}

// Triangle is the RGB vulkan-tutorial triangle. turn is revolutions (nil is 0).
// Output shape is (h, w, 4).
func Triangle(h, w int, turn *ndarray.Tensor) (*ndarray.Tensor, error) {
	if h < 1 || w < 1 {
		return nil, ndarray.ErrShape
	}
	if turn == nil {
		turn = ndarray.Const(0)
	}
	return triangle(turn, ndarray.Const(float32(w)), ndarray.Const(float32(h)), ndarray.Shape{h, w, 4})
}

// TriangleDynamic is Triangle with runtime width/height splats. Coord rank is (1,1,4);
// the caller sets the real (h,w,4) with Tensor.Resize before Eval/Exec.
func TriangleDynamic(turn, width, height *ndarray.Tensor) (*ndarray.Tensor, error) {
	return triangle(turn, width, height, ndarray.Shape{1, 1, 4})
}

func triangle(turn, width, height *ndarray.Tensor, shape ndarray.Shape) (*ndarray.Tensor, error) {
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
	u := by.Add(cy.Neg()).Mul(px.Add(cx.Neg())).Add(cx.Add(bx.Neg()).Mul(py.Add(cy.Neg()))).Div(den)
	v := cy.Add(ay.Neg()).Mul(px.Add(cx.Neg())).Add(ax.Add(cx.Neg()).Mul(py.Add(cy.Neg()))).Div(den)
	weight := ndarray.Const(1).Add(u.Neg()).Add(v.Neg())
	inside := u.GreaterEqual(ndarray.Const(0)).And(v.GreaterEqual(ndarray.Const(0))).And(weight.GreaterEqual(ndarray.Const(0)))
	channel := ndarray.Coord(2, shape)
	rgb := channel.Equal(ndarray.ConstInt(0)).Where(u.Mul(ndarray.Const(255)),
		channel.Equal(ndarray.ConstInt(1)).Where(v.Mul(ndarray.Const(255)),
			channel.Equal(ndarray.ConstInt(2)).Where(weight.Mul(ndarray.Const(255)), ndarray.Const(255))))
	out := inside.Where(rgb, channel.Equal(ndarray.ConstInt(3)).Where(ndarray.Const(255), ndarray.Const(0)))
	if out.Shape() == nil {
		return nil, ndarray.ErrOp
	}
	return out, nil
}

type frame struct {
	sine, cosine, scale, originX, originY *ndarray.Tensor
}

func (f frame) rotate(x, y float32) (px, py *ndarray.Tensor) {
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
		destIndex := dst.PixOffset(dst.Rect.Min.X, dst.Rect.Min.Y+y)
		sourceIndex := y * w * 4
		for range w {
			dst.Pix[destIndex] = toUint8(pixels[sourceIndex])
			dst.Pix[destIndex+1] = toUint8(pixels[sourceIndex+1])
			dst.Pix[destIndex+2] = toUint8(pixels[sourceIndex+2])
			dst.Pix[destIndex+3] = toUint8(pixels[sourceIndex+3])
			destIndex += 4
			sourceIndex += 4
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
