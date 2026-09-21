package onnx

import (
	"fmt"
	"math"

	"github.com/lewtec/lewkit/x/ndarray"
)

func applyBatchNorm[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	if node.attributeInteger("training_mode", 0) != 0 {
		return nil, fmt.Errorf("%w: training_mode", ErrOp)
	}
	if len(node.Inputs) < 5 {
		return nil, ErrOp
	}
	x, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	scale, ok := values[node.Inputs[1]]
	if !ok {
		return nil, fmt.Errorf("%w: scale", ErrGraph)
	}
	bias, ok := values[node.Inputs[2]]
	if !ok {
		return nil, fmt.Errorf("%w: bias", ErrGraph)
	}
	mean, ok := values[node.Inputs[3]]
	if !ok {
		return nil, fmt.Errorf("%w: mean", ErrGraph)
	}
	v, ok := values[node.Inputs[4]]
	if !ok {
		return nil, fmt.Errorf("%w: var", ErrGraph)
	}
	eps := node.attributeFloat("epsilon", 1e-5)
	scale, err = channelLike(scale, x)
	if err != nil {
		return nil, err
	}
	bias, err = channelLike(bias, x)
	if err != nil {
		return nil, err
	}
	mean, err = channelLike(mean, x)
	if err != nil {
		return nil, err
	}
	v, err = channelLike(v, x)
	if err != nil {
		return nil, err
	}
	inv := v.Add(ndarray.Const(T(eps))).Sqrt().Reciprocal()
	return x.Add(mean.Neg()).Mul(inv).Mul(scale).Add(bias), nil
}

func applyInstanceNorm[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	if len(node.Inputs) < 3 {
		return nil, ErrOp
	}
	x, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	scale, ok := values[node.Inputs[1]]
	if !ok {
		return nil, fmt.Errorf("%w: scale", ErrGraph)
	}
	bias, ok := values[node.Inputs[2]]
	if !ok {
		return nil, fmt.Errorf("%w: bias", ErrGraph)
	}
	eps := node.attributeFloat("epsilon", 1e-5)
	x, err = leaf(x)
	if err != nil {
		return nil, err
	}
	shape := x.Shape()
	if len(shape) < 3 {
		return nil, ndarray.ErrShape
	}
	mean := x
	for axis := 2; axis < len(shape); axis++ {
		mean, err = reduceAxis(mean, axis, (*ndarray.Tensor[T]).Add)
		if err != nil {
			return nil, err
		}
	}
	spatial := 1
	for _, d := range shape[2:] {
		spatial *= d
	}
	mean = divide(mean, ndarray.Const(T(spatial)))
	mean, err = withView(mean, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Expand(shape)
	})
	if err != nil {
		return nil, err
	}
	delta := x.Add(mean.Neg())
	vr := delta.Mul(delta)
	for axis := 2; axis < len(shape); axis++ {
		vr, err = reduceAxis(vr, axis, (*ndarray.Tensor[T]).Add)
		if err != nil {
			return nil, err
		}
	}
	vr = divide(vr, ndarray.Const(T(spatial)))
	vr, err = withView(vr, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Expand(shape)
	})
	if err != nil {
		return nil, err
	}
	scale, err = channelLike(scale, x)
	if err != nil {
		return nil, err
	}
	bias, err = channelLike(bias, x)
	if err != nil {
		return nil, err
	}
	inv := vr.Add(ndarray.Const(T(eps))).Sqrt().Reciprocal()
	return delta.Mul(inv).Mul(scale).Add(bias), nil
}

func applyLpNorm[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	p := node.attributeInteger("p", 2)
	if p != 1 && p != 2 {
		return nil, fmt.Errorf("%w: p", ErrOp)
	}
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
	var mag *ndarray.Tensor[T]
	if p == 2 {
		mag, err = reduceAxis(x.Mul(x), axis, (*ndarray.Tensor[T]).Add)
	} else {
		mag, err = reduceAxis(absT(x), axis, (*ndarray.Tensor[T]).Add)
	}
	if err != nil {
		return nil, err
	}
	mag, err = withView(mag, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Expand(shape)
	})
	if err != nil {
		return nil, err
	}
	if p == 2 {
		mag = mag.Sqrt()
	}
	return x.Mul(mag.Reciprocal()), nil
}

func applyIsInf[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
		pos := node.attributeInteger("detect_positive", 1) != 0
		neg := node.attributeInteger("detect_negative", 1) != 0
		inf := ndarray.Const(float32(math.Inf(1))).Cast[T]()
		ninf := ndarray.Const(float32(math.Inf(-1))).Cast[T]()
		var m *ndarray.Tensor[int32]
		if pos {
			m = x.Equal(inf)
		}
		if neg {
			n := x.Equal(ninf)
			if m == nil {
				m = n
			} else {
				m = m.Or(n)
			}
		}
		if m == nil {
			return x.Mul(ndarray.Const(T(0)))
		}
		return bool01[T](m)
	})
}

func applyLayerNorm[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
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
	eps := node.attributeFloat("epsilon", 1e-5)
	mean := x
	for a := axis; a < len(shape); a++ {
		mean, err = reduceAxis(mean, a, (*ndarray.Tensor[T]).Add)
		if err != nil {
			return nil, err
		}
	}
	n := 1
	for _, d := range shape[axis:] {
		n *= d
	}
	mean = divide(mean, ndarray.Const(T(n)))
	meanKeep := mean
	mean, err = withView(mean, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Expand(shape)
	})
	if err != nil {
		return nil, err
	}
	delta := x.Add(mean.Neg())
	vr := delta.Mul(delta)
	for a := axis; a < len(shape); a++ {
		vr, err = reduceAxis(vr, a, (*ndarray.Tensor[T]).Add)
		if err != nil {
			return nil, err
		}
	}
	vr = divide(vr, ndarray.Const(T(n)))
	inv := vr.Add(ndarray.Const(T(eps))).Sqrt().Reciprocal()
	inv, err = leaf(inv)
	if err != nil {
		return nil, err
	}
	invKeep := inv
	invE, err := withView(invKeep, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Expand(shape)
	})
	if err != nil {
		return nil, err
	}
	y := delta.Mul(invE)
	if len(node.Inputs) > 1 {
		w, ok := values[node.Inputs[1]]
		if ok {
			wt, err := expandLike(w, x)
			if err != nil {
				return nil, err
			}
			y = y.Mul(wt)
		}
	}
	if len(node.Inputs) > 2 {
		b, ok := values[node.Inputs[2]]
		if ok {
			bt, err := expandLike(b, x)
			if err != nil {
				return nil, err
			}
			y = y.Add(bt)
		}
	}
	if len(node.Outputs) > 1 {
		values[node.Outputs[1]] = meanKeep
	}
	if len(node.Outputs) > 2 {
		values[node.Outputs[2]] = invKeep
	}
	return y, nil
}

func applyGroupNorm[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	groups := int(node.attributeInteger("num_groups", 0))
	if groups <= 0 {
		return nil, ErrOp
	}
	x, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	x, err = leaf(x)
	if err != nil {
		return nil, err
	}
	shape := x.Shape()
	if len(shape) < 2 || shape[1]%groups != 0 {
		return nil, ndarray.ErrShape
	}
	n, c := shape[0], shape[1]
	spatial := 1
	for _, d := range shape[2:] {
		spatial *= d
	}
	g := groups
	cg := c / g
	grouped, err := x.Reshape(append(ndarray.Shape{n, g, cg}, shape[2:]...))
	if err != nil {
		return nil, err
	}
	mean := grouped
	for a := 2; a < len(grouped.Shape()); a++ {
		mean, err = reduceAxis(mean, a, (*ndarray.Tensor[T]).Add)
		if err != nil {
			return nil, err
		}
	}
	count := cg * spatial
	mean = divide(mean, ndarray.Const(T(count)))
	mean, err = withView(mean, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Expand(grouped.Shape())
	})
	if err != nil {
		return nil, err
	}
	delta := grouped.Add(mean.Neg())
	vr := delta.Mul(delta)
	for a := 2; a < len(grouped.Shape()); a++ {
		vr, err = reduceAxis(vr, a, (*ndarray.Tensor[T]).Add)
		if err != nil {
			return nil, err
		}
	}
	vr = divide(vr, ndarray.Const(T(count)))
	vr, err = withView(vr, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Expand(grouped.Shape())
	})
	if err != nil {
		return nil, err
	}
	eps := node.attributeFloat("epsilon", 1e-5)
	y := delta.Mul(vr.Add(ndarray.Const(T(eps))).Sqrt().Reciprocal())
	yt, err := withView(y, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Reshape(shape)
	})
	if err != nil {
		return nil, err
	}
	if len(node.Inputs) > 1 {
		scale, err := channelLike(values[node.Inputs[1]], x)
		if err != nil {
			return nil, err
		}
		yt = yt.Mul(scale)
	}
	if len(node.Inputs) > 2 {
		bias, err := channelLike(values[node.Inputs[2]], x)
		if err != nil {
			return nil, err
		}
		yt = yt.Add(bias)
	}
	return yt, nil
}

func applyMVN[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	x, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	x, err = leaf(x)
	if err != nil {
		return nil, err
	}
	axes := node.attributeIntegers("axes")
	if len(axes) == 0 {
		if len(x.Shape()) == 4 {
			axes = []int64{0, 2, 3}
		} else {
			axes = []int64{0}
		}
	}
	shape := x.Shape()
	n := 1
	for _, a := range axes {
		axis := int(a)
		if axis < 0 {
			axis += len(shape)
		}
		if axis < 0 || axis >= len(shape) {
			return nil, ndarray.ErrShape
		}
		n *= shape[axis]
	}
	mean, err := reduceTensors(x, axes, true, (*ndarray.Tensor[T]).Add)
	if err != nil {
		return nil, err
	}
	mean = divide(mean, ndarray.Const(T(n)))
	mean, err = expandLike(mean, x)
	if err != nil {
		return nil, err
	}
	delta := x.Add(mean.Neg())
	vr, err := reduceTensors(delta.Mul(delta), axes, true, (*ndarray.Tensor[T]).Add)
	if err != nil {
		return nil, err
	}
	vr = divide(vr, ndarray.Const(T(n)))
	vr, err = expandLike(vr, x)
	if err != nil {
		return nil, err
	}
	return delta.Mul(vr.Add(ndarray.Const(float32(1e-9)).Cast[T]()).Sqrt().Reciprocal()), nil
}

func applyIsNaN[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
		return bool01[T](x.CmpNe(x))
	})
}

func applyRMSNorm[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
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
	eps := node.attributeFloat("epsilon", 1e-5)
	ms := x.Mul(x)
	n := 1
	for a := axis; a < len(shape); a++ {
		n *= shape[a]
		ms, err = reduceAxis(ms, a, (*ndarray.Tensor[T]).Add)
		if err != nil {
			return nil, err
		}
	}
	ms = divide(ms, ndarray.Const(T(n)))
	inv := ms.Add(ndarray.Const(T(eps))).Sqrt().Reciprocal()
	inv, err = expandLike(inv, x)
	if err != nil {
		return nil, err
	}
	y := x.Mul(inv)
	if len(node.Inputs) > 1 {
		if w, ok := values[node.Inputs[1]]; ok {
			wt, err := expandLike(w, x)
			if err != nil {
				return nil, err
			}
			y = y.Mul(wt)
		}
	}
	return y, nil
}
