package experiments

import (
	"context"
	"errors"
	"fmt"
	stdimage "image"
	"math"

	"github.com/lewtec/lewkit/x/ndarray"
	ndimage "github.com/lewtec/lewkit/x/ndarray/image"
)

const (
	triAX, triAY = 0, -2.0 / 3.0
	triBX, triBY = 0.5, 1.0 / 3.0
	triCX, triCY = -0.5, 1.0 / 3.0
)

func triangleAt(h, w int, turn *ndarray.Tensor) (*ndarray.Tensor, error) {
	if h < 1 || w < 1 {
		return nil, ndarray.ErrShape
	}
	if turn == nil {
		turn = ndarray.Const(0)
	}
	return triangle(turn, ndarray.Const(float32(w)), ndarray.Const(float32(h)), ndarray.Shape{h, w, 4})
}

func triangleDynamic(turn, width, height *ndarray.Tensor) (*ndarray.Tensor, error) {
	return triangle(turn, width, height, ndarray.Shape{1, 1, 4})
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
	fr := triangleFrame{sine, cosine, scale, originX, originY}
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

type triangleFrame struct {
	sine, cosine, scale, originX, originY *ndarray.Tensor
}

func (f triangleFrame) rotate(x, y float32) (px, py *ndarray.Tensor) {
	px = ndarray.Const(x).Mul(f.cosine).Add(ndarray.Const(y).Mul(f.sine).Neg()).Mul(f.scale).Add(f.originX)
	py = ndarray.Const(x).Mul(f.sine).Add(ndarray.Const(y).Mul(f.cosine)).Mul(f.scale).Add(f.originY)
	return px, py
}

type trianglePainter struct {
	evaluator   ndarray.Evaluator
	triangle    *ndarray.Tensor
	turn        *ndarray.Tensor
	width       *ndarray.Tensor
	height      *ndarray.Tensor
	buffer      []float32
	frameHeight int
	frameWidth  int
}

func newTrianglePainter(ctx context.Context) (*trianglePainter, error) {
	turn, err := ndarray.New([]float32{0}, nil)
	if err != nil {
		return nil, err
	}
	width, err := ndarray.New([]float32{1}, nil)
	if err != nil {
		return nil, err
	}
	height, err := ndarray.New([]float32{1}, nil)
	if err != nil {
		return nil, err
	}
	tri, err := triangleDynamic(turn, width, height)
	if err != nil {
		return nil, err
	}
	tri = tri.Cast(ndarray.U8)
	evaluator, err := ndarray.Open(ctx)
	if err != nil {
		return nil, err
	}
	return &trianglePainter{
		evaluator: evaluator,
		triangle:  tri,
		turn:      turn,
		width:     width,
		height:    height,
	}, nil
}

func (p *trianglePainter) Draw(ctx context.Context, destination *stdimage.RGBA, turn float64) error {
	if p == nil || p.triangle == nil {
		return ndarray.ErrOp
	}
	frameHeight, frameWidth := destination.Rect.Dy(), destination.Rect.Dx()
	if frameHeight < 1 || frameWidth < 1 {
		return nil
	}
	if buf := p.turn.Buffer(); len(buf) > 0 {
		buf[0] = float32(turn)
	}
	if buf := p.width.Buffer(); len(buf) > 0 {
		buf[0] = float32(frameWidth)
	}
	if buf := p.height.Buffer(); len(buf) > 0 {
		buf[0] = float32(frameHeight)
	}
	if frameHeight != p.frameHeight || frameWidth != p.frameWidth {
		if err := p.triangle.Resize(ndarray.Shape{frameHeight, frameWidth, 4}); err != nil {
			return err
		}
		p.frameHeight, p.frameWidth = frameHeight, frameWidth
	}
	size := frameHeight * frameWidth * 4
	if cap(p.buffer) < size {
		p.buffer = make([]float32, size)
	} else {
		p.buffer = p.buffer[:size]
	}
	if err := p.triangle.Eval(ctx, p.evaluator, p.buffer); err != nil {
		return err
	}
	ndimage.Write(destination, p.buffer)
	return nil
}

func (p *trianglePainter) Close() error {
	if p == nil {
		return nil
	}
	var err error
	if p.triangle != nil {
		err = p.triangle.Close()
		p.triangle = nil
	}
	if p.evaluator != nil {
		err = errors.Join(err, p.evaluator.Close())
		p.evaluator = nil
	}
	return err
}
