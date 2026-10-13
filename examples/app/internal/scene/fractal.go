package scene

import (
	"math"

	"github.com/lewtec/lewkit/x/ndarray"
)

// fractalTurnsPerSecond walks the Julia parameter once every twenty seconds.
const fractalTurnsPerSecond = 0.05

// fractalIterations is the bailout, unrolled into one kernel.
// ndarray has no loop op, so each step is another node in the same graph.
const fractalIterations = 32

// The parameter circle sits inside the Mandelbrot main cardioid.
// Center -0.1 and radius 0.34 stay at |μ| ≤ 0.8, so the filled Julia
// breathes without collapsing to the cusp.
const (
	juliaCenter = float32(-0.1)
	juliaRadius = float32(0.34)
	juliaSpan   = float32(3)
)

func fractalAt(h, w int, turn *ndarray.Tensor[float32]) (*ndarray.Tensor[float32], error) {
	args, err := rasterAt(h, w, turn)
	if err != nil {
		return nil, err
	}
	return fractal(args.seed, args.width, args.height, args.shape)
}

func fractalDynamic(turn, width, height *ndarray.Tensor[float32]) (*ndarray.Tensor[float32], error) {
	args := rasterDynamic(turn, width, height)
	return fractal(args.seed, args.width, args.height, args.shape)
}

func fractal(turn, width, height *ndarray.Tensor[float32], shape ndarray.Shape) (*ndarray.Tensor[float32], error) {
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
	scale := ndarray.Const(juliaSpan).Div(minFloat(width, height))
	zx := px.Add(width.Mul(ndarray.Const(float32(0.5))).Neg()).Mul(scale)
	zy := py.Add(height.Mul(ndarray.Const(float32(0.5))).Neg()).Mul(scale)
	cx, cy := juliaParameter(turn)
	count := ndarray.Const(float32(0))
	limit := ndarray.Const(float32(4))
	for range fractalIterations {
		mag2 := zx.Mul(zx).Add(zy.Mul(zy))
		inside := mag2.CmpLt(limit)
		nzx := zx.Mul(zx).Add(zy.Mul(zy).Neg()).Add(cx)
		nzy := zx.Mul(zy).Mul(ndarray.Const(float32(2))).Add(cy)
		zx = inside.Where(nzx, zx)
		zy = inside.Where(nzy, zy)
		count = inside.Where(count.Add(ndarray.Const(float32(1))), count)
	}
	escaped := count.CmpLt(ndarray.Const(float32(fractalIterations)))
	channel := ndarray.Coord(2, shape)
	rgb := juliaShade(count, channel)
	interior := channel.Equal(ndarray.Const(int32(3))).Where(ndarray.Const(float32(255)), ndarray.Const(float32(0)))
	out := escaped.Where(rgb, interior)
	if out.Shape() == nil {
		return nil, ndarray.ErrOp
	}
	return out, nil
}

func juliaParameter(turn *ndarray.Tensor[float32]) (cx, cy *ndarray.Tensor[float32]) {
	theta := turn.Mul(ndarray.Const(float32(2 * math.Pi)))
	cosine := theta.Add(ndarray.Const(float32(math.Pi / 2))).Sin()
	cx = ndarray.Const(juliaCenter).Add(cosine.Mul(ndarray.Const(juliaRadius)))
	cy = theta.Sin().Mul(ndarray.Const(juliaRadius))
	return cx, cy
}

func juliaShade(count *ndarray.Tensor[float32], channel *ndarray.Tensor[int32]) *ndarray.Tensor[float32] {
	// 0.4 rad per escape spreads one rainbow across the bailout.
	ang := count.Mul(ndarray.Const(float32(0.4)))
	amp := ndarray.Const(float32(127.5))
	bias := ndarray.Const(float32(127.5))
	red := ang.Sin().Mul(amp).Add(bias)
	green := ang.Add(ndarray.Const(float32(2 * math.Pi / 3))).Sin().Mul(amp).Add(bias)
	blue := ang.Add(ndarray.Const(float32(4 * math.Pi / 3))).Sin().Mul(amp).Add(bias)
	return channel.Equal(ndarray.Const(int32(0))).Where(red,
		channel.Equal(ndarray.Const(int32(1))).Where(green,
			channel.Equal(ndarray.Const(int32(2))).Where(blue, ndarray.Const(float32(255)))))
}
