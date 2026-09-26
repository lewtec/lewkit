package onnx

import (
	"fmt"

	"github.com/lewtec/lewkit/x/ndarray"
)

func applyReduce[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, integerShapes map[string][]int64, op func(*ndarray.Tensor[T], *ndarray.Tensor[T]) *ndarray.Tensor[T], mean bool) (*ndarray.Tensor[T], error) {
	x, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	return applyReduceOn(x, node, integerShapes, op, mean)
}

func applyReduceOn[T ndarray.Number](x *ndarray.Tensor[T], node Node, integerShapes map[string][]int64, op func(*ndarray.Tensor[T], *ndarray.Tensor[T]) *ndarray.Tensor[T], mean bool) (*ndarray.Tensor[T], error) {
	x, err := leaf(x)
	if err != nil {
		return nil, err
	}
	keepdims := node.attributeInteger("keepdims", 1) != 0
	noop := node.attributeInteger("noop_with_empty_axes", 0) != 0
	axes := node.attributeIntegers("axes")
	if len(node.Inputs) > 1 && node.Inputs[1] != "" {
		if a, ok := integerShapes[node.Inputs[1]]; ok {
			axes = a
		}
	}
	shape := x.Shape()
	if len(axes) == 0 && noop {
		return x, nil
	}
	reduced := 1
	seen := map[int]struct{}{}
	use := axes
	if len(use) == 0 {
		use = make([]int64, len(shape))
		for i := range use {
			use[i] = int64(i)
		}
	}
	for _, a := range use {
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
		reduced *= shape[axis]
	}
	x, err = reduceTensors(x, axes, keepdims, op)
	if err != nil {
		return nil, err
	}
	if mean {
		x = divide(x, ndarray.Const(T(reduced)))
	}
	return x, nil
}

func applyLogSoftmax[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	s, err := applySoftmax(values, node)
	if err != nil {
		return nil, err
	}
	return log(s), nil
}

func applyTile[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, integerShapes map[string][]int64) (*ndarray.Tensor[T], error) {
	x, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	if len(node.Inputs) < 2 {
		return nil, ndarray.ErrShape
	}
	reps, ok := integerShapes[node.Inputs[1]]
	if !ok {
		return nil, fmt.Errorf("%w: repeats", ErrOp)
	}
	x, err = leaf(x)
	if err != nil {
		return nil, err
	}
	shape := x.Shape()
	if len(reps) != len(shape) {
		return nil, ndarray.ErrShape
	}
	for axis := len(shape) - 1; axis >= 0; axis-- {
		n := int(reps[axis])
		if n < 1 {
			return nil, ndarray.ErrShape
		}
		if n == 1 {
			continue
		}
		parts := make([]*ndarray.Tensor[T], n)
		for i := range n {
			parts[i] = x
		}
		x, err = concatTensors(parts, axis)
		if err != nil {
			return nil, err
		}
	}
	return x, nil
}

func applyTrilu[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, integerShapes map[string][]int64) (*ndarray.Tensor[T], error) {
	x, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	x, err = leaf(x)
	if err != nil {
		return nil, err
	}
	shape := x.Shape()
	if len(shape) < 2 {
		return nil, ndarray.ErrShape
	}
	k := int32(0)
	if len(node.Inputs) > 1 && node.Inputs[1] != "" {
		got, ok, err := optionalInt(values, integerShapes, node.Inputs[1], "k")
		if err != nil {
			return nil, err
		}
		if ok {
			k = int32(got)
		}
	}
	upper := node.attributeInteger("upper", 1) != 0
	i := ndarray.Coord(len(shape)-2, shape)
	j := ndarray.Coord(len(shape)-1, shape)
	diag := j.Add(ndarray.Const(-k))
	var mask *ndarray.Tensor[int32]
	if upper {
		mask = i.CmpLt(diag).Or(i.Equal(diag))
	} else {
		mask = diag.CmpLt(i).Or(i.Equal(diag))
	}
	return mask.Where(x, ndarray.Const(T(0))), nil
}

func applyEyeLike[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	x, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	shape := x.Shape()
	if len(shape) != 2 {
		return nil, ndarray.ErrShape
	}
	k := int32(node.attributeInteger("k", 0))
	i := ndarray.Coord(0, shape)
	j := ndarray.Coord(1, shape)
	diag := i.Equal(j.Add(ndarray.Const(-k)))
	return bool01[T](diag), nil
}
