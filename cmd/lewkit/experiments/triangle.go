package experiments

import (
	"context"
	"fmt"
	"math"

	"github.com/lewtec/lewkit/x/ndarray"
)

const (
	triAX, triAY = 0, -2.0 / 3.0
	triBX, triBY = 0.5, 1.0 / 3.0
	triCX, triCY = -0.5, 1.0 / 3.0
)

func triangleAt(h, w int, turn *ndarray.Tensor[float32]) (*ndarray.Tensor[float32], error) {
	args, err := rasterAt(h, w, turn)
	if err != nil {
		return nil, err
	}
	return triangle(args.seed, args.width, args.height, args.shape)
}

func triangleDynamic(turn, width, height *ndarray.Tensor[float32]) (*ndarray.Tensor[float32], error) {
	args := rasterDynamic(turn, width, height)
	return triangle(args.seed, args.width, args.height, args.shape)
}

func requireFloatSplat(n *ndarray.Tensor[float32], name string) error {
	if n == nil {
		return ndarray.ErrType
	}
	if shape := n.Shape(); len(shape) != 0 {
		return fmt.Errorf("%w: %s must be a splat", ndarray.ErrShape, name)
	}
	return nil
}

func minFloat(a, b *ndarray.Tensor[float32]) *ndarray.Tensor[float32] {
	return a.CmpLt(b).Where(a, b)
}

func triangle(turn, width, height *ndarray.Tensor[float32], shape ndarray.Shape) (*ndarray.Tensor[float32], error) {
	if err := requireFloatSplat(turn, "turn"); err != nil {
		return nil, err
	}
	if err := requireFloatSplat(width, "width"); err != nil {
		return nil, err
	}
	if err := requireFloatSplat(height, "height"); err != nil {
		return nil, err
	}
	px := ndarray.Coord(1, shape).Cast[float32]().Add(ndarray.Const(float32(0.5)))
	py := ndarray.Coord(0, shape).Cast[float32]().Add(ndarray.Const(float32(0.5)))
	tau := turn.Mul(ndarray.Const(float32(2 * math.Pi)))
	sine := tau.Sin()
	cosine := tau.Add(ndarray.Const(float32(math.Pi / 2))).Sin()
	scale := minFloat(width, height).Mul(ndarray.Const(float32(0.5)))
	originX := width.Mul(ndarray.Const(float32(0.5)))
	originY := height.Mul(ndarray.Const(float32(0.5)))
	fr := triangleFrame{sine, cosine, scale, originX, originY}
	ax, ay := fr.rotate(triAX, triAY)
	bx, by := fr.rotate(triBX, triBY)
	cx, cy := fr.rotate(triCX, triCY)
	den := by.Add(cy.Neg()).Mul(ax.Add(cx.Neg())).Add(cx.Add(bx.Neg()).Mul(ay.Add(cy.Neg())))
	u := by.Add(cy.Neg()).Mul(px.Add(cx.Neg())).Add(cx.Add(bx.Neg()).Mul(py.Add(cy.Neg()))).Div(den)
	v := cy.Add(ay.Neg()).Mul(px.Add(cx.Neg())).Add(ax.Add(cx.Neg()).Mul(py.Add(cy.Neg()))).Div(den)
	weight := ndarray.Const(float32(1)).Add(u.Neg()).Add(v.Neg())
	inside := u.GreaterEqual(ndarray.Const(float32(0))).And(v.GreaterEqual(ndarray.Const(float32(0)))).And(weight.GreaterEqual(ndarray.Const(float32(0))))
	channel := ndarray.Coord(2, shape)
	rgb := channel.Equal(ndarray.Const(int32(0))).Where(u.Mul(ndarray.Const(float32(255))),
		channel.Equal(ndarray.Const(int32(1))).Where(v.Mul(ndarray.Const(float32(255))),
			channel.Equal(ndarray.Const(int32(2))).Where(weight.Mul(ndarray.Const(float32(255))), ndarray.Const(float32(255)))))
	out := inside.Where(rgb, channel.Equal(ndarray.Const(int32(3))).Where(ndarray.Const(float32(255)), ndarray.Const(float32(0))))
	if out.Shape() == nil {
		return nil, ndarray.ErrOp
	}
	return out, nil
}

type triangleFrame struct {
	sine, cosine, scale, originX, originY *ndarray.Tensor[float32]
}

func (f triangleFrame) rotate(x, y float32) (px, py *ndarray.Tensor[float32]) {
	px = ndarray.Const(x).Mul(f.cosine).Add(ndarray.Const(y).Mul(f.sine).Neg()).Mul(f.scale).Add(f.originX)
	py = ndarray.Const(x).Mul(f.sine).Add(ndarray.Const(y).Mul(f.cosine)).Mul(f.scale).Add(f.originY)
	return px, py
}

func newTrianglePainter(ctx context.Context) (*framePainter, error) {
	return newFramePainter(ctx, 1, triangleDynamic)
}
