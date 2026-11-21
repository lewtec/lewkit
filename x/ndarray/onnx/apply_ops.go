package onnx

import (
	"fmt"
	"math"

	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/ndarray/nn"
	"github.com/lewtec/lewkit/x/ndarray/onnx/internal/proto"
)

const (
	ln2    = float32(math.Ln2)
	log2e  = float32(1 / math.Ln2)
	piHalf = float32(math.Pi / 2)
)

func exp[T ndarray.Number](x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
	return x.Mul(ndarray.Const(log2e).Cast[T]()).Exp2()
}

func log[T ndarray.Number](x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
	return x.Log2().Mul(ndarray.Const(ln2).Cast[T]())
}

func minimum[T ndarray.Number](a, b *ndarray.Tensor[T]) *ndarray.Tensor[T] {
	return a.Neg().Max(b.Neg()).Neg()
}

func unary[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, fn func(*ndarray.Tensor[T]) *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
	value, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	return fn(value), nil
}

func binaryBroadcast[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, fn func(*ndarray.Tensor[T], *ndarray.Tensor[T]) *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
	left, right, err := twoInputs(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	left, right, err = broadcast(left, right)
	if err != nil {
		return nil, err
	}
	return fn(left, right), nil
}

func nary[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, fn func(*ndarray.Tensor[T], *ndarray.Tensor[T]) *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
	if len(node.Inputs) == 0 {
		return nil, ErrOp
	}
	acc, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	for i := 1; i < len(node.Inputs); i++ {
		next, ok := values[node.Inputs[i]]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrGraph, node.Inputs[i])
		}
		acc, next, err = broadcast(acc, next)
		if err != nil {
			return nil, err
		}
		acc = fn(acc, next)
	}
	return acc, nil
}

func optionalInput[T ndarray.Number](values map[string]*ndarray.Tensor[T], names []string, i int) (*ndarray.Tensor[T], bool) {
	if i >= len(names) || names[i] == "" {
		return nil, false
	}
	t, ok := values[names[i]]
	return t, ok
}

func applyClip[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	value, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	if min, ok := optionalInput(values, node.Inputs, 1); ok {
		value, min, err = broadcast(value, min)
		if err != nil {
			return nil, err
		}
		value = value.Max(min)
	}
	if max, ok := optionalInput(values, node.Inputs, 2); ok {
		value, max, err = broadcast(value, max)
		if err != nil {
			return nil, err
		}
		value = minimum(value, max)
	}
	return value, nil
}

func applyTranspose[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	value, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	value, err = leaf(value)
	if err != nil {
		return nil, err
	}
	perm := node.attributeIntegers("perm")
	axes := make([]int, len(value.Shape()))
	if len(perm) == 0 {
		for i := range axes {
			axes[i] = len(axes) - 1 - i
		}
	} else {
		if len(perm) != len(axes) {
			return nil, ndarray.ErrShape
		}
		for i, p := range perm {
			axes[i] = int(p)
		}
	}
	return value.Permute(axes...)
}

func applyFlatten[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	value, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	shape := value.Shape()
	axis := int(node.attributeInteger("axis", 1))
	if axis < 0 {
		axis += len(shape)
	}
	if axis < 0 || axis > len(shape) {
		return nil, ndarray.ErrShape
	}
	outer, inner := 1, 1
	for i, dim := range shape {
		if i < axis {
			outer *= dim
		} else {
			inner *= dim
		}
	}
	return withView(value, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Reshape(ndarray.Shape{outer, inner})
	})
}

func applySqueeze[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, integerShapes map[string][]int64) (*ndarray.Tensor[T], error) {
	value, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	var axes []int64
	if len(node.Inputs) > 1 && node.Inputs[1] != "" {
		var ok bool
		axes, ok = integerShapes[node.Inputs[1]]
		if !ok {
			return nil, fmt.Errorf("%w: squeeze axes", ErrOp)
		}
	} else {
		axes = node.attributeIntegers("axes")
	}
	out, err := squeezeShape(value.Shape(), axes)
	if err != nil {
		return nil, err
	}
	return withView(value, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Reshape(out)
	})
}

func applyUnsqueeze[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, integerShapes map[string][]int64) (*ndarray.Tensor[T], error) {
	value, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	if len(node.Inputs) < 2 {
		return nil, ndarray.ErrShape
	}
	axes, ok := integerShapes[node.Inputs[1]]
	if !ok {
		axes = node.attributeIntegers("axes")
	}
	out, err := unsqueezeShape(value.Shape(), axes)
	if err != nil {
		return nil, err
	}
	return withView(value, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Reshape(out)
	})
}

func applyExpand[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, integerShapes map[string][]int64) (*ndarray.Tensor[T], error) {
	value, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	if len(node.Inputs) < 2 {
		return nil, ndarray.ErrShape
	}
	target, ok := integerShapes[node.Inputs[1]]
	if !ok {
		return nil, fmt.Errorf("%w: expand shape", ErrOp)
	}
	out, err := expandShape(value.Shape(), target)
	if err != nil {
		return nil, err
	}
	value, err = ndarray.PadRank(value, len(out))
	if err != nil {
		return nil, err
	}
	return withView(value, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Expand(out)
	})
}

func applyGemm[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	a, b, err := twoInputs(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	a, err = leaf(a)
	if err != nil {
		return nil, err
	}
	b, err = leaf(b)
	if err != nil {
		return nil, err
	}
	if node.attributeInteger("transA", 0) != 0 {
		a, err = transposeLast2(a)
		if err != nil {
			return nil, err
		}
	}
	if node.attributeInteger("transB", 0) != 0 {
		b, err = transposeLast2(b)
		if err != nil {
			return nil, err
		}
	}
	y, err := nn.MatrixMultiply(a, b)
	if err != nil {
		return nil, err
	}
	alpha := node.attributeFloat("alpha", 1)
	if alpha != 1 {
		y = y.Mul(ndarray.Const(T(alpha)))
	}
	if c, ok := optionalInput(values, node.Inputs, 2); ok {
		c, err = leaf(c)
		if err != nil {
			return nil, err
		}
		beta := node.attributeFloat("beta", 1)
		if beta != 1 {
			c = c.Mul(ndarray.Const(T(beta)))
		}
		y, c, err = broadcast(y, c)
		if err != nil {
			return nil, err
		}
		y = y.Add(c)
	}
	return y, nil
}

func transposeLast2[T ndarray.Number](t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
	shape := t.Shape()
	if len(shape) < 2 {
		return nil, ndarray.ErrShape
	}
	axes := make([]int, len(shape))
	for i := range axes {
		axes[i] = i
	}
	axes[len(axes)-2], axes[len(axes)-1] = axes[len(axes)-1], axes[len(axes)-2]
	return t.Permute(axes...)
}

func squeezeShape(in ndarray.Shape, axes []int64) (ndarray.Shape, error) {
	drop := map[int]struct{}{}
	if len(axes) == 0 {
		for i, dim := range in {
			if dim == 1 {
				drop[i] = struct{}{}
			}
		}
	} else {
		for _, axis := range axes {
			a := int(axis)
			if a < 0 {
				a += len(in)
			}
			if a < 0 || a >= len(in) || in[a] != 1 {
				return nil, ndarray.ErrShape
			}
			drop[a] = struct{}{}
		}
	}
	out := make(ndarray.Shape, 0, len(in))
	for i, dim := range in {
		if _, ok := drop[i]; !ok {
			out = append(out, dim)
		}
	}
	return out, nil
}

func unsqueezeShape(in ndarray.Shape, axes []int64) (ndarray.Shape, error) {
	if len(axes) == 0 {
		return nil, ndarray.ErrShape
	}
	outRank := len(in) + len(axes)
	insert := map[int]struct{}{}
	for _, axis := range axes {
		a := int(axis)
		if a < 0 {
			a += outRank
		}
		if a < 0 || a >= outRank {
			return nil, ndarray.ErrShape
		}
		if _, ok := insert[a]; ok {
			return nil, ndarray.ErrShape
		}
		insert[a] = struct{}{}
	}
	out := make(ndarray.Shape, outRank)
	i := 0
	for d := 0; d < outRank; d++ {
		if _, ok := insert[d]; ok {
			out[d] = 1
			continue
		}
		if i >= len(in) {
			return nil, ndarray.ErrShape
		}
		out[d] = in[i]
		i++
	}
	return out, nil
}

func expandShape(input ndarray.Shape, target []int64) (ndarray.Shape, error) {
	want := shapeFromDimensions(target)
	rank := max(len(input), len(want))
	left := make(ndarray.Shape, rank)
	right := make(ndarray.Shape, rank)
	copy(left[rank-len(input):], input)
	copy(right[rank-len(want):], want)
	for i := 0; i < rank-len(input); i++ {
		left[i] = 1
	}
	for i := 0; i < rank-len(want); i++ {
		right[i] = 1
	}
	out := make(ndarray.Shape, rank)
	for i := range rank {
		switch {
		case left[i] == right[i]:
			out[i] = left[i]
		case left[i] == 1:
			out[i] = right[i]
		case right[i] == 1:
			out[i] = left[i]
		default:
			return nil, ndarray.ErrShape
		}
	}
	return out, nil
}

func leaky[T ndarray.Number](x *ndarray.Tensor[T], alpha float32) *ndarray.Tensor[T] {
	return x.GreaterEqual(ndarray.Const(T(0))).Where(x, x.Mul(ndarray.Const(T(alpha))))
}

func cos[T ndarray.Number](x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
	return x.Add(ndarray.Const(piHalf).Cast[T]()).Sin()
}

func trunc[T ndarray.Number](x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
	return x.Cast[int32]().Cast[T]()
}

func floor[T ndarray.Number](x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
	t := trunc(x)
	neg := x.CmpLt(ndarray.Const(T(0)))
	frac := x.CmpNe(t)
	return neg.And(frac).Where(t.Add(ndarray.Const(float32(-1)).Cast[T]()), t)
}

func ceil[T ndarray.Number](x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
	return floor(x.Neg()).Neg()
}

func roundEven[T ndarray.Number](x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
	t := trunc(x)
	frac := absT(x.Add(t.Neg()))
	half := ndarray.Const(float32(0.5)).Cast[T]()
	one := ndarray.Const(T(1))
	away := x.GreaterEqual(ndarray.Const(T(0))).Where(t.Add(one), t.Add(one.Neg()))
	odd := t.Cast[int32]().And(ndarray.Const(int32(1))).CmpNe(ndarray.Const(int32(0)))
	return frac.CmpLt(half).Where(t, half.CmpLt(frac).Where(away, odd.Where(away, t)))
}

func applyCast[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	x, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	return castTo[T](x, node.attributeInteger("to", 0))
}

func applyCastLike[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	x, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	return x, nil
}

func castTo[T ndarray.Number](x *ndarray.Tensor[T], to int64) (*ndarray.Tensor[T], error) {
	switch proto.TensorProto_DataType(to) {
	case proto.TensorProto_UNDEFINED, proto.TensorProto_FLOAT, proto.TensorProto_INT32, proto.TensorProto_UINT8, proto.TensorProto_BOOL:
		if to == int64(proto.TensorProto_INT32) {
			return x.Cast[int32]().Cast[T](), nil
		}
		if to == int64(proto.TensorProto_UINT8) || to == int64(proto.TensorProto_BOOL) {
			return x.Cast[uint8]().Cast[T](), nil
		}
		return x, nil
	default:
		return nil, fmt.Errorf("%w: cast %d", ErrOp, to)
	}
}

func applyShape[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	x, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	shape := x.Shape()
	rank := len(shape)
	start := int(node.attributeInteger("start", 0))
	end := int(node.attributeInteger("end", int64(rank)))
	if start < 0 {
		start += rank
	}
	if end < 0 {
		end += rank
	}
	if start < 0 {
		start = 0
	}
	if end > rank {
		end = rank
	}
	if start > end {
		start = end
	}
	out := make([]T, end-start)
	for i, dim := range shape[start:end] {
		out[i] = T(dim)
	}
	return ndarray.New(out, ndarray.Shape{len(out)})
}

func applySize[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	x, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	return ndarray.New([]T{T(x.Size())}, ndarray.Shape{1})
}

func applyMod[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	a, b, err := twoInputs(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	a, b, err = broadcast(a, b)
	if err != nil {
		return nil, err
	}
	q := trunc(divide(a, b))
	r := a.Add(q.Mul(b).Neg())
	if node.attributeInteger("fmod", 0) != 0 {
		return r, nil
	}
	flip := r.CmpLt(ndarray.Const(T(0))).Xor(b.CmpLt(ndarray.Const(T(0)))).And(r.CmpNe(ndarray.Const(T(0))))
	return flip.Where(r.Add(b), r), nil
}
