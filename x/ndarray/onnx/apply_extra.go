package onnx

import (
	"fmt"
	"math"

	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/ndarray/nn"
)

func applyAveragePool[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	input, inputShape, kernelShape, strideHeight, strideWidth, err := spatialPoolSetup(values, node)
	if err != nil {
		return nil, err
	}
	kernelHeight, kernelWidth := int(kernelShape[0]), int(kernelShape[1])
	pads, err := spatialPads(node.attributeString("auto_pad"), node.attributeIntegers("pads"), inputShape[2], inputShape[3], kernelHeight, kernelWidth, strideHeight, strideWidth)
	if err != nil {
		return nil, err
	}
	return nn.AveragePool2D(input, kernelHeight, kernelWidth, strideHeight, strideWidth, pads, node.attributeInteger("count_include_pad", 0) != 0)
}

func applyLpPool[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	input, inputShape, kernelShape, strideHeight, strideWidth, err := spatialPoolSetup(values, node)
	if err != nil {
		return nil, err
	}
	p := node.attributeInteger("p", 2)
	if p != 1 && p != 2 {
		return nil, fmt.Errorf("%w: p %d", ErrOp, p)
	}
	inner := absT(input)
	if p == 2 {
		inner = inner.Mul(inner)
	}
	kernelHeight, kernelWidth := int(kernelShape[0]), int(kernelShape[1])
	pads, err := spatialPads(node.attributeString("auto_pad"), node.attributeIntegers("pads"), inputShape[2], inputShape[3], kernelHeight, kernelWidth, strideHeight, strideWidth)
	if err != nil {
		return nil, err
	}
	mean, err := nn.AveragePool2D(inner, kernelHeight, kernelWidth, strideHeight, strideWidth, pads, true)
	if err != nil {
		return nil, err
	}
	sum := mean.Mul(ndarray.Const(T(kernelHeight * kernelWidth)))
	if p == 1 {
		return sum, nil
	}
	return sum.Sqrt(), nil
}

func applyGlobalLpPool[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	x, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	x, err = leaf(x)
	if err != nil {
		return nil, err
	}
	shape := x.Shape()
	if len(shape) < 3 {
		return nil, ndarray.ErrShape
	}
	p := node.attributeInteger("p", 2)
	if p != 1 && p != 2 {
		return nil, fmt.Errorf("%w: p %d", ErrOp, p)
	}
	acc := absT(x)
	if p != 1 {
		acc = acc.Mul(acc)
	}
	for axis := 2; axis < len(shape); axis++ {
		acc, err = reduceAxis(acc, axis, (*ndarray.Tensor[T]).Add)
		if err != nil {
			return nil, err
		}
	}
	if p == 1 {
		return acc, nil
	}
	return acc.Sqrt(), nil
}

func applyWindow[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, integerShapes map[string][]int64, kind string) (*ndarray.Tensor[T], error) {
	if len(node.Inputs) < 1 {
		return nil, ErrOp
	}
	n := 0
	if ints, ok := integerShapes[node.Inputs[0]]; ok && len(ints) > 0 {
		n = int(ints[0])
	} else if t, ok := values[node.Inputs[0]]; ok {
		d, err := t.Data()
		if err != nil || len(d) == 0 {
			return nil, fmt.Errorf("%w: size", ErrOp)
		}
		n = int(d[0])
	}
	if n < 1 {
		return nil, ndarray.ErrShape
	}
	den := n
	if node.attributeInteger("periodic", 1) == 0 {
		if n == 1 {
			return ndarray.New([]T{T(1)}, ndarray.Shape{1})
		}
		den = n - 1
	}
	idx := ndarray.Coord(0, ndarray.Shape{n}).Cast[T]()
	ang := idx.Mul(ndarray.Const(float32(2 * math.Pi / float64(den))).Cast[T]())
	c := cos(ang)
	switch kind {
	case "hamming":
		return ndarray.Const(float32(0.5434782608695652)).Cast[T]().Add(ndarray.Const(float32(0.45652173913043476)).Cast[T]().Mul(c).Neg()), nil
	case "blackman":
		c2 := cos(ang.Mul(ndarray.Const(T(2))))
		return ndarray.Const(float32(0.42)).Cast[T]().Add(ndarray.Const(float32(0.5)).Cast[T]().Mul(c).Neg()).Add(ndarray.Const(float32(0.08)).Cast[T]().Mul(c2)), nil
	default:
		return ndarray.Const(float32(0.5)).Cast[T]().Mul(ndarray.Const(T(1)).Add(c.Neg())), nil
	}
}

func applyCenterCropPad[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, integerShapes map[string][]int64) (*ndarray.Tensor[T], error) {
	x, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	if len(node.Inputs) < 2 {
		return nil, ndarray.ErrShape
	}
	target, ok := integerShapes[node.Inputs[1]]
	if !ok {
		return nil, fmt.Errorf("%w: shape", ErrOp)
	}
	x, err = leaf(x)
	if err != nil {
		return nil, err
	}
	shape := x.Shape()
	axes := node.attributeIntegers("axes")
	if len(axes) == 0 {
		axes = make([]int64, len(target))
		off := len(shape) - len(target)
		for i := range axes {
			axes[i] = int64(off + i)
		}
	}
	if len(axes) != len(target) {
		return nil, ndarray.ErrShape
	}
	shrink := make([][2]int, len(shape))
	pad := make([][2]int, len(shape))
	needPad := false
	for i, dim := range shape {
		shrink[i] = [2]int{0, dim}
	}
	for i, a := range axes {
		axis := int(a)
		if axis < 0 {
			axis += len(shape)
		}
		if axis < 0 || axis >= len(shape) {
			return nil, ndarray.ErrShape
		}
		want := int(target[i])
		have := shape[axis]
		if want == have {
			continue
		}
		if want < have {
			start := (have - want) / 2
			shrink[axis] = [2]int{start, start + want}
			continue
		}
		before := (want - have) / 2
		pad[axis] = [2]int{before, want - have - before}
		needPad = true
	}
	x, err = withView(x, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Shrink(shrink)
	})
	if err != nil {
		return nil, err
	}
	if !needPad {
		return x, nil
	}
	return withView(x, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Pad(pad)
	})
}

func applyGlobalPool[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, average bool) (*ndarray.Tensor[T], error) {
	input, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	input, err = leaf(input)
	if err != nil {
		return nil, err
	}
	shape := input.Shape()
	if len(shape) != 4 {
		return nil, ndarray.ErrShape
	}
	if average {
		return nn.AveragePool2D(input, shape[2], shape[3], shape[2], shape[3], nil, true)
	}
	return nn.MaximumPool2D(input, shape[2], shape[3], shape[2], shape[3], nil)
}

func applyConcat[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	if len(node.Inputs) == 0 {
		return nil, ErrOp
	}
	parts := make([]*ndarray.Tensor[T], len(node.Inputs))
	for i, name := range node.Inputs {
		t, ok := values[name]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrGraph, name)
		}
		t, err := leaf(t)
		if err != nil {
			return nil, err
		}
		parts[i] = t
	}
	return concatTensors(parts, int(node.attributeInteger("axis", 0)))
}

func applyPad[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, integerShapes map[string][]int64) (*ndarray.Tensor[T], error) {
	mode := node.attributeString("mode")
	if mode != "" && mode != "constant" {
		return nil, fmt.Errorf("%w: pad mode %s", ErrOp, mode)
	}
	input, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	input, err = leaf(input)
	if err != nil {
		return nil, err
	}
	if len(node.Inputs) < 2 {
		return nil, ndarray.ErrShape
	}
	pads, ok := integerShapes[node.Inputs[1]]
	if !ok {
		return nil, fmt.Errorf("%w: pads", ErrOp)
	}
	rank := len(input.Shape())
	if len(pads) != 2*rank {
		return nil, ndarray.ErrShape
	}
	arg := make([][2]int, rank)
	for i := range rank {
		arg[i] = [2]int{int(pads[i]), int(pads[rank+i])}
	}
	padded, err := input.Pad(arg)
	if err != nil {
		return nil, err
	}
	value, ok := optionalInput(values, node.Inputs, 2)
	if !ok {
		return padded, nil
	}
	ones := make([]T, input.Size())
	one := T(1)
	for i := range ones {
		ones[i] = one
	}
	maskSrc, err := ndarray.New(ones, input.Shape())
	if err != nil {
		return nil, err
	}
	mask, err := maskSrc.Pad(arg)
	if err != nil {
		return nil, err
	}
	value, err = padToRank(value, len(padded.Shape()))
	if err != nil {
		return nil, err
	}
	fill, err := value.Expand(padded.Shape())
	if err != nil {
		return nil, err
	}
	return mask.CmpNe(ndarray.Const(T(0))).Where(padded, fill), nil
}

func applySlice[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, integerShapes map[string][]int64) (*ndarray.Tensor[T], error) {
	input, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	if len(node.Inputs) < 3 {
		return nil, ndarray.ErrShape
	}
	starts, ok := integerShapes[node.Inputs[1]]
	if !ok {
		return nil, fmt.Errorf("%w: starts", ErrOp)
	}
	ends, ok := integerShapes[node.Inputs[2]]
	if !ok {
		return nil, fmt.Errorf("%w: ends", ErrOp)
	}
	if len(starts) != len(ends) {
		return nil, ndarray.ErrShape
	}
	axes := make([]int64, len(starts))
	for i := range axes {
		axes[i] = int64(i)
	}
	if len(node.Inputs) > 3 && node.Inputs[3] != "" {
		if a, ok := integerShapes[node.Inputs[3]]; ok {
			axes = a
		}
	}
	if len(node.Inputs) > 4 && node.Inputs[4] != "" {
		steps, ok := integerShapes[node.Inputs[4]]
		if !ok {
			return nil, fmt.Errorf("%w: steps", ErrOp)
		}
		for _, s := range steps {
			if s != 1 {
				return nil, fmt.Errorf("%w: slice step", ErrOp)
			}
		}
	}
	shape := input.Shape()
	ranges := make([][2]int, len(shape))
	for i, dim := range shape {
		ranges[i] = [2]int{0, dim}
	}
	for i, axis := range axes {
		a := int(axis)
		if a < 0 {
			a += len(shape)
		}
		if a < 0 || a >= len(shape) {
			return nil, ndarray.ErrShape
		}
		dim := shape[a]
		start, end := int(starts[i]), int(ends[i])
		if start < 0 {
			start += dim
		}
		if end < 0 {
			end += dim
		}
		if start < 0 {
			start = 0
		}
		if end > dim {
			end = dim
		}
		if start > end {
			start = end
		}
		ranges[a] = [2]int{start, end}
	}
	return withView(input, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Shrink(ranges)
	})
}

func applyConstant[T ndarray.Number](node Node) (*ndarray.Tensor[T], error) {
	p := node.attributeTensor("value")
	if p == nil {
		return nil, fmt.Errorf("%w: constant value", ErrOp)
	}
	return tensorFromProto[T](p)
}

func applyConstantOfShape[T ndarray.Number](node Node, integerShapes map[string][]int64) (*ndarray.Tensor[T], error) {
	if len(node.Inputs) < 1 {
		return nil, ndarray.ErrShape
	}
	shape, ok := integerShapes[node.Inputs[0]]
	if !ok {
		return nil, fmt.Errorf("%w: constantofshape", ErrOp)
	}
	var z T
	if p := node.attributeTensor("value"); p != nil {
		data, err := valuesFromProto[T](p)
		if err != nil {
			return nil, err
		}
		if len(data) > 0 {
			z = data[0]
		}
	}
	return ndarray.Full(z, shapeFromDimensions(shape))
}

func applySpaceToDepth[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	block := int(node.attributeInteger("blocksize", 0))
	if block <= 0 {
		return nil, ErrOp
	}
	input, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	input, err = leaf(input)
	if err != nil {
		return nil, err
	}
	shape := input.Shape()
	if len(shape) != 4 || shape[2]%block != 0 || shape[3]%block != 0 {
		return nil, ndarray.ErrShape
	}
	n, c, h, w := shape[0], shape[1], shape[2], shape[3]
	hs, ws := h/block, w/block
	shaped, err := input.Reshape(ndarray.Shape{n, c, hs, block, ws, block})
	if err != nil {
		return nil, err
	}
	perm, err := shaped.Permute(0, 3, 5, 1, 2, 4)
	if err != nil {
		return nil, err
	}
	return perm.Reshape(ndarray.Shape{n, c * block * block, hs, ws})
}

func applyDepthToSpace[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	block := int(node.attributeInteger("blocksize", 0))
	if block <= 0 {
		return nil, ErrOp
	}
	mode := node.attributeString("mode")
	if mode == "" {
		mode = "DCR"
	}
	input, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	input, err = leaf(input)
	if err != nil {
		return nil, err
	}
	shape := input.Shape()
	if len(shape) != 4 || shape[1]%(block*block) != 0 {
		return nil, ndarray.ErrShape
	}
	n, c, h, w := shape[0], shape[1], shape[2], shape[3]
	cout := c / (block * block)
	var shaped *ndarray.Tensor[T]
	if mode == "CRD" {
		shaped, err = input.Reshape(ndarray.Shape{n, cout, block, block, h, w})
		if err != nil {
			return nil, err
		}
		shaped, err = shaped.Permute(0, 1, 4, 2, 5, 3)
	} else {
		shaped, err = input.Reshape(ndarray.Shape{n, block, block, cout, h, w})
		if err != nil {
			return nil, err
		}
		shaped, err = shaped.Permute(0, 3, 4, 1, 5, 2)
	}
	if err != nil {
		return nil, err
	}
	return shaped.Reshape(ndarray.Shape{n, cout, h * block, w * block})
}

func applySoftmax[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	x, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	x, err = leaf(x)
	if err != nil {
		return nil, err
	}
	shape := x.Shape()
	axis := int(node.attributeInteger("axis", -1))
	if axis < 0 {
		axis += len(shape)
	}
	if axis < 0 || axis >= len(shape) {
		return nil, ndarray.ErrShape
	}
	m, err := reduceAxis(x, axis, (*ndarray.Tensor[T]).Max)
	if err != nil {
		return nil, err
	}
	m, err = expandLike(m, x)
	if err != nil {
		return nil, err
	}
	e := exp(x.Add(m.Neg()))
	s, err := reduceAxis(e, axis, (*ndarray.Tensor[T]).Add)
	if err != nil {
		return nil, err
	}
	s, err = expandLike(s, e)
	if err != nil {
		return nil, err
	}
	return e.Mul(s.Reciprocal()), nil
}

func absT[T ndarray.Number](x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
	return x.Max(x.Neg())
}

func sigmoid[T ndarray.Number](x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
	return ndarray.Const(T(1)).Add(exp(x.Neg())).Reciprocal()
}

func tanh[T ndarray.Number](x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
	e := exp(x.Mul(ndarray.Const(T(2))))
	return e.Add(ndarray.Const(float32(-1)).Cast[T]()).Mul(e.Add(ndarray.Const(T(1))).Reciprocal())
}

func hardSigmoid[T ndarray.Number](x *ndarray.Tensor[T], alpha, beta float32) *ndarray.Tensor[T] {
	y := x.Mul(ndarray.Const(T(alpha))).Add(ndarray.Const(T(beta)))
	y = y.Max(ndarray.Const(T(0)))
	return minimum(y, ndarray.Const(T(1)))
}
