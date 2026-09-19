package experiments

import (
	"context"
	"errors"
	stdimage "image"

	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ndarray"
)

func perlinAt(h, w int, time *ndarray.Tensor[float32]) (*ndarray.Tensor[float32], error) {
	if h < 1 || w < 1 {
		return nil, ndarray.ErrShape
	}
	if time == nil {
		time = ndarray.Const(float32(0))
	}
	return perlin(time, ndarray.Const(float32(w)), ndarray.Const(float32(h)), ndarray.Shape{h, w, 4})
}

func perlinDynamic(time, width, height *ndarray.Tensor[float32]) (*ndarray.Tensor[float32], error) {
	return perlin(time, width, height, ndarray.Shape{1, 1, 4})
}

func perlin(time, width, height *ndarray.Tensor[float32], shape ndarray.Shape) (*ndarray.Tensor[float32], error) {
	if err := requireFloatSplat(time, "time"); err != nil {
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
	u := px.Div(width).Mul(ndarray.Const(float32(8))).Add(time)
	v := py.Div(height).Mul(ndarray.Const(float32(8)))
	n := perlinNoise(u, v)
	n = n.Add(perlinNoise(u.Mul(ndarray.Const(float32(2))), v.Mul(ndarray.Const(float32(2)))).Mul(ndarray.Const(float32(0.5))))
	n = n.Add(perlinNoise(u.Mul(ndarray.Const(float32(4))), v.Mul(ndarray.Const(float32(4)))).Mul(ndarray.Const(float32(0.25))))
	gray := minFloat(n.Mul(ndarray.Const(float32(72))).Add(ndarray.Const(float32(127.5))).Max(ndarray.Const(float32(0))), ndarray.Const(float32(255)))
	channel := ndarray.Coord(2, shape)
	out := channel.Equal(ndarray.Const(int32(3))).Where(ndarray.Const(float32(255)), gray)
	if out.Shape() == nil {
		return nil, ndarray.ErrOp
	}
	return out, nil
}

func perlinNoise(u, v *ndarray.Tensor[float32]) *ndarray.Tensor[float32] {
	ix := u.Cast[int32]()
	iy := v.Cast[int32]()
	fx := u.Add(ix.Cast[float32]().Neg())
	fy := v.Add(iy.Cast[float32]().Neg())
	fadeX := perlinFade(fx)
	fadeY := perlinFade(fy)
	x1 := ix.Add(ndarray.Const(int32(1)))
	y1 := iy.Add(ndarray.Const(int32(1)))
	n00 := perlinGradient(perlinHash(ix, iy), fx, fy)
	n10 := perlinGradient(perlinHash(x1, iy), fx.Add(ndarray.Const(float32(-1))), fy)
	n01 := perlinGradient(perlinHash(ix, y1), fx, fy.Add(ndarray.Const(float32(-1))))
	n11 := perlinGradient(perlinHash(x1, y1), fx.Add(ndarray.Const(float32(-1))), fy.Add(ndarray.Const(float32(-1))))
	return perlinInterpolate(perlinInterpolate(n00, n10, fadeX), perlinInterpolate(n01, n11, fadeX), fadeY)
}

func perlinHash(x, y *ndarray.Tensor[int32]) *ndarray.Tensor[int32] {
	n := x.Mul(ndarray.Const(int32(374761393))).Xor(y.Mul(ndarray.Const(int32(668265263))))
	n = n.Xor(n.Shr(ndarray.Const(int32(13))))
	n = n.Mul(ndarray.Const(int32(1274126177)))
	return n.Xor(n.Shr(ndarray.Const(int32(16))))
}

func perlinGradient(h *ndarray.Tensor[int32], dx, dy *ndarray.Tensor[float32]) *ndarray.Tensor[float32] {
	gx := h.And(ndarray.Const(int32(1))).Cast[float32]().Mul(ndarray.Const(float32(2))).Add(ndarray.Const(float32(-1)))
	gy := h.Shr(ndarray.Const(int32(1))).And(ndarray.Const(int32(1))).Cast[float32]().Mul(ndarray.Const(float32(2))).Add(ndarray.Const(float32(-1)))
	return gx.Mul(dx).Add(gy.Mul(dy))
}

func perlinFade(t *ndarray.Tensor[float32]) *ndarray.Tensor[float32] {
	inner := t.Mul(t.Mul(ndarray.Const(float32(6))).Add(ndarray.Const(float32(-15)))).Add(ndarray.Const(float32(10)))
	return t.Mul(t).Mul(t).Mul(inner)
}

func perlinInterpolate(a, b, t *ndarray.Tensor[float32]) *ndarray.Tensor[float32] {
	return a.Add(t.Mul(b.Add(a.Neg())))
}

type perlinPainter struct {
	evaluator ndarray.Evaluator
	pixels    *ndarray.Tensor[uint8]
	time      *ndarray.Tensor[float32]
	width     *ndarray.Tensor[float32]
	height    *ndarray.Tensor[float32]
}

func newPerlinPainter(ctx context.Context) (*perlinPainter, error) {
	time, err := ndarray.New([]float32{0}, nil)
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
	noise, err := perlinDynamic(time, width, height)
	if err != nil {
		return nil, err
	}
	evaluator, err := ndarray.Open(ctx)
	if err != nil {
		return nil, err
	}
	return &perlinPainter{
		evaluator: evaluator,
		pixels:    noise.Cast[uint8](),
		time:      time,
		width:     width,
		height:    height,
	}, nil
}

func (p *perlinPainter) Draw(ctx context.Context, destination *stdimage.RGBA, elapsed float64) error {
	if p == nil || p.pixels == nil {
		return ndarray.ErrOp
	}
	frameHeight, frameWidth := destination.Rect.Dy(), destination.Rect.Dx()
	if frameHeight < 1 || frameWidth < 1 {
		return nil
	}
	if buf := p.time.Buffer(); len(buf) > 0 {
		buf[0] = float32(elapsed * 0.4)
	}
	if buf := p.width.Buffer(); len(buf) > 0 {
		buf[0] = float32(frameWidth)
	}
	if buf := p.height.Buffer(); len(buf) > 0 {
		buf[0] = float32(frameHeight)
	}
	return window.Present(ctx, p.pixels, p.evaluator, destination)
}

func (p *perlinPainter) Close() error {
	if p == nil {
		return nil
	}
	var err error
	if p.pixels != nil {
		err = p.pixels.Close()
		p.pixels = nil
	}
	if p.evaluator != nil {
		err = errors.Join(err, p.evaluator.Close())
		p.evaluator = nil
	}
	return err
}
