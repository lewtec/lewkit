package onnx

import (
	"context"
	"math"

	"github.com/lewtec/lewkit/x/ndarray"
)

func concatTensors[T ndarray.Number](ctx context.Context, evaluator ndarray.Evaluator, parts []*ndarray.Tensor[T], axis int) (*ndarray.Tensor[T], error) {
	if len(parts) == 0 {
		return nil, ErrOp
	}
	for i, p := range parts {
		q, err := leaf(ctx, evaluator, p)
		if err != nil {
			return nil, err
		}
		parts[i] = q
	}
	rank := len(parts[0].Shape())
	if axis < 0 {
		axis += rank
	}
	if axis < 0 || axis >= rank {
		return nil, ndarray.ErrShape
	}
	outShape := parts[0].Shape().Clone()
	for _, p := range parts[1:] {
		ps := p.Shape()
		if len(ps) != rank {
			return nil, ndarray.ErrShape
		}
		for i, dim := range ps {
			if i == axis {
				outShape[i] += dim
				continue
			}
			if dim != outShape[i] {
				return nil, ndarray.ErrShape
			}
		}
	}
	acc, err := ndarray.Zeros[T](outShape)
	if err != nil {
		return nil, err
	}
	offset := 0
	for _, p := range parts {
		pad := make([][2]int, rank)
		pad[axis] = [2]int{offset, outShape[axis] - offset - p.Shape()[axis]}
		placed, err := p.Pad(pad)
		if err != nil {
			return nil, err
		}
		acc = acc.Add(placed)
		offset += p.Shape()[axis]
	}
	return acc, nil
}

func gatherTensors[T ndarray.Number](ctx context.Context, evaluator ndarray.Evaluator, data *ndarray.Tensor[T], idxs []int64, axis int) (*ndarray.Tensor[T], error) {
	data, err := leaf(ctx, evaluator, data)
	if err != nil {
		return nil, err
	}
	shape := data.Shape()
	if axis < 0 {
		axis += len(shape)
	}
	if axis < 0 || axis >= len(shape) {
		return nil, ndarray.ErrShape
	}
	dim := shape[axis]
	parts := make([]*ndarray.Tensor[T], len(idxs))
	for i, ix := range idxs {
		a := int(ix)
		if a < 0 {
			a += dim
		}
		if a < 0 || a >= dim {
			return nil, ndarray.ErrShape
		}
		parts[i], err = data.Shrink(axisShrinkRange(shape, axis, a, a+1))
		if err != nil {
			return nil, err
		}
	}
	return concatTensors(ctx, evaluator, parts, axis)
}

func scanTensor[T ndarray.Number](ctx context.Context, evaluator ndarray.Evaluator, x *ndarray.Tensor[T], axis int, reverse, exclusive bool, op func(*ndarray.Tensor[T], *ndarray.Tensor[T]) *ndarray.Tensor[T], identity T) (*ndarray.Tensor[T], error) {
	x, err := leaf(ctx, evaluator, x)
	if err != nil {
		return nil, err
	}
	shape := x.Shape()
	if axis < 0 {
		axis += len(shape)
	}
	if axis < 0 || axis >= len(shape) {
		return nil, ndarray.ErrShape
	}
	dim := shape[axis]
	if dim == 0 {
		return nil, ndarray.ErrOp
	}
	parts := make([]*ndarray.Tensor[T], dim)
	var acc *ndarray.Tensor[T]
	start, step, end := 0, 1, dim
	if reverse {
		start, step, end = dim-1, -1, -1
	}
	for i := start; i != end; i += step {
		cell, err := x.Shrink(axisShrinkRange(shape, axis, i, i+1))
		if err != nil {
			return nil, err
		}
		if exclusive {
			if acc == nil {
				z, err := ndarray.Full(identity, cell.Shape())
				if err != nil {
					return nil, err
				}
				parts[i] = z
			} else {
				parts[i] = acc
			}
		}
		if acc == nil {
			acc = cell
		} else {
			acc = op(acc, cell)
		}
		if !exclusive {
			parts[i] = acc
		}
	}
	return concatTensors(ctx, evaluator, parts, axis)
}

func rangeTensor[T ndarray.Number](start, limit, delta T) (*ndarray.Tensor[T], error) {
	n := rangeLength(start, limit, delta)
	if n < 0 {
		return nil, ndarray.ErrShape
	}
	idx := ndarray.Coord(0, ndarray.Shape{n})
	return idx.Cast[T]().Mul(ndarray.Const(delta)).Add(ndarray.Const(start)), nil
}

func rangeLength[T ndarray.Number](start, limit, delta T) int {
	step := numberAsFloat64(delta)
	if step == 0 {
		return -1
	}
	n := int(math.Ceil((numberAsFloat64(limit) - numberAsFloat64(start)) / step))
	return max(n, 0)
}

func numberAsFloat64[T ndarray.Number](v T) float64 {
	switch x := any(v).(type) {
	case float32:
		return float64(x)
	case int32:
		return float64(x)
	case uint8:
		return float64(x)
	default:
		return 0
	}
}

func reduceTensors[T ndarray.Number](ctx context.Context, evaluator ndarray.Evaluator, x *ndarray.Tensor[T], axes []int64, keepdims bool, op func(*ndarray.Tensor[T], *ndarray.Tensor[T]) *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
	x, err := leaf(ctx, evaluator, x)
	if err != nil {
		return nil, err
	}
	shape := x.Shape()
	if len(axes) == 0 {
		axes = make([]int64, len(shape))
		for i := range axes {
			axes[i] = int64(i)
		}
	}
	seen := map[int]struct{}{}
	for _, a := range axes {
		axis := int(a)
		if axis < 0 {
			axis += len(shape)
		}
		if axis < 0 || axis >= len(shape) {
			return nil, ndarray.ErrShape
		}
		if _, ok := seen[axis]; ok {
			continue
		}
		seen[axis] = struct{}{}
		x, err = reduceAxis(ctx, evaluator, x, axis, op)
		if err != nil {
			return nil, err
		}
	}
	if keepdims {
		return x, nil
	}
	out := make(ndarray.Shape, 0, len(x.Shape()))
	cur := x.Shape()
	for i, dim := range cur {
		if _, ok := seen[i]; ok {
			continue
		}
		out = append(out, dim)
	}
	return withView(ctx, evaluator, x, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Reshape(out)
	})
}

func reduceAxis[T ndarray.Number](ctx context.Context, evaluator ndarray.Evaluator, x *ndarray.Tensor[T], axis int, op func(*ndarray.Tensor[T], *ndarray.Tensor[T]) *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
	x, err := leaf(ctx, evaluator, x)
	if err != nil {
		return nil, err
	}
	shape := x.Shape()
	dim := shape[axis]
	if dim == 0 {
		return nil, ndarray.ErrOp
	}
	var acc *ndarray.Tensor[T]
	for i := range dim {
		cell, err := x.Shrink(axisShrinkRange(shape, axis, i, i+1))
		if err != nil {
			return nil, err
		}
		if acc == nil {
			acc = cell
		} else {
			acc = op(acc, cell)
		}
	}
	return acc, nil
}

func expandLike[T ndarray.Number](ctx context.Context, evaluator ndarray.Evaluator, t, like *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
	want, have := like.Shape(), t.Shape()
	if len(have) > len(want) {
		return nil, ndarray.ErrShape
	}
	shaped := make(ndarray.Shape, len(want))
	copy(shaped[len(want)-len(have):], have)
	for i := 0; i < len(want)-len(have); i++ {
		shaped[i] = 1
	}
	t, err := withView(ctx, evaluator, t, func(x *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return x.Reshape(shaped)
	})
	if err != nil {
		return nil, err
	}
	return withView(ctx, evaluator, t, func(x *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return x.Expand(want)
	})
}

func channelLike[T ndarray.Number](ctx context.Context, evaluator ndarray.Evaluator, t, like *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
	rank := len(like.Shape())
	c := t.Shape()
	if len(c) != 1 || rank < 2 {
		return nil, ndarray.ErrShape
	}
	shaped := make(ndarray.Shape, rank)
	for i := range shaped {
		shaped[i] = 1
	}
	shaped[1] = c[0]
	t, err := withView(ctx, evaluator, t, func(x *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return x.Reshape(shaped)
	})
	if err != nil {
		return nil, err
	}
	return withView(ctx, evaluator, t, func(x *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return x.Expand(like.Shape())
	})
}
