package onnx

import (
	"fmt"

	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/ndarray/nn"
)

func applyGather[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, integerShapes map[string][]int64) (*ndarray.Tensor[T], error) {
	data, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	if len(node.Inputs) < 2 {
		return nil, ErrOp
	}
	idxs, ok := integerShapes[node.Inputs[1]]
	if !ok {
		return nil, fmt.Errorf("%w: indices", ErrOp)
	}
	return gatherTensors(data, idxs, int(node.attributeInteger("axis", 0)))
}

func applyCumSum[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, integerShapes map[string][]int64) (*ndarray.Tensor[T], error) {
	return applyScan(values, node, integerShapes, (*ndarray.Tensor[T]).Add, 0)
}

func applyCumProd[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, integerShapes map[string][]int64) (*ndarray.Tensor[T], error) {
	return applyScan(values, node, integerShapes, (*ndarray.Tensor[T]).Mul, 1)
}

func applyScan[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, integerShapes map[string][]int64, op func(*ndarray.Tensor[T], *ndarray.Tensor[T]) *ndarray.Tensor[T], identity T) (*ndarray.Tensor[T], error) {
	x, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	axis := int(node.attributeInteger("axis", 0))
	if len(node.Inputs) > 1 && node.Inputs[1] != "" {
		got, ok, err := optionalInt(values, integerShapes, node.Inputs[1], "axis")
		if err != nil {
			return nil, err
		}
		if ok {
			axis = int(got)
		}
	}
	return scanTensor(x, axis, node.attributeInteger("reverse", 0) != 0, node.attributeInteger("exclusive", 0) != 0, op, identity)
}

// optionalInt reads one scalar from an integer shape or a tensor input.
// Missing names are not an error. label is the ErrOp text when the tensor is empty.
func optionalInt[T ndarray.Number](values map[string]*ndarray.Tensor[T], integerShapes map[string][]int64, name, label string) (int64, bool, error) {
	if ints, ok := integerShapes[name]; ok && len(ints) > 0 {
		return ints[0], true, nil
	}
	tensor, ok := values[name]
	if !ok {
		return 0, false, nil
	}
	tensor, err := leaf(tensor)
	if err != nil {
		return 0, false, err
	}
	data, err := tensor.Data()
	if err != nil || len(data) == 0 {
		return 0, false, fmt.Errorf("%w: %s", ErrOp, label)
	}
	return int64(data[0]), true, nil
}

func applyHardmax[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
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
	m, err := reduceAxis(x, axis, (*ndarray.Tensor[T]).Max)
	if err != nil {
		return nil, err
	}
	m, err = withView(m, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Expand(shape)
	})
	if err != nil {
		return nil, err
	}
	eq := bool01[T](x.Equal(m))
	cs, err := scanTensor(eq, axis, false, false, (*ndarray.Tensor[T]).Add, 0)
	if err != nil {
		return nil, err
	}
	first := bool01[T](cs.Equal(ndarray.Const(T(1))))
	return eq.Mul(first), nil
}

func applyRange[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, integerShapes map[string][]int64) (*ndarray.Tensor[T], error) {
	if len(node.Inputs) < 3 {
		return nil, ErrOp
	}
	start, err := rangeScalar(values, integerShapes, node.Inputs[0])
	if err != nil {
		return nil, err
	}
	limit, err := rangeScalar(values, integerShapes, node.Inputs[1])
	if err != nil {
		return nil, err
	}
	delta, err := rangeScalar(values, integerShapes, node.Inputs[2])
	if err != nil {
		return nil, err
	}
	return rangeTensor(start, limit, delta)
}

func rangeScalar[T ndarray.Number](values map[string]*ndarray.Tensor[T], integerShapes map[string][]int64, name string) (T, error) {
	var zero T
	if ints, ok := integerShapes[name]; ok && len(ints) > 0 {
		return T(ints[0]), nil
	}
	t, ok := values[name]
	if !ok {
		return zero, fmt.Errorf("%w: %s", ErrGraph, name)
	}
	t, err := leaf(t)
	if err != nil {
		return zero, err
	}
	d, err := t.Data()
	if err != nil || len(d) == 0 {
		return zero, fmt.Errorf("%w: %s", ErrOp, name)
	}
	return d[0], nil
}

func applyGatherElements[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	data, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	if len(node.Inputs) < 2 {
		return nil, ErrOp
	}
	idx, ok := values[node.Inputs[1]]
	if !ok {
		return nil, fmt.Errorf("%w: indices", ErrGraph)
	}
	data, err = leaf(data)
	if err != nil {
		return nil, err
	}
	idx, err = leaf(idx)
	if err != nil {
		return nil, err
	}
	shape := data.Shape()
	axis := int(node.attributeInteger("axis", 0))
	if axis < 0 {
		axis += len(shape)
	}
	dim := shape[axis]
	neg := idx.CmpLt(ndarray.Const(T(0)))
	idx = neg.Where(idx.Add(ndarray.Const(T(dim))), idx)
	acc, err := ndarray.Zeros[T](idx.Shape())
	if err != nil {
		return nil, err
	}
	for k := range dim {
		cell, err := data.Shrink(nn.AxisShrink(shape, axis, k, k+1))
		if err != nil {
			return nil, err
		}
		cell, err = withView(cell, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
			return t.Expand(idx.Shape())
		})
		if err != nil {
			return nil, err
		}
		mk := idx.Equal(ndarray.Const(T(k)))
		acc = mk.Where(cell, acc)
	}
	return acc, nil
}

func applyCompress[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	x, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	if len(node.Inputs) < 2 {
		return nil, ErrOp
	}
	cond, ok := values[node.Inputs[1]]
	if !ok {
		return nil, fmt.Errorf("%w: condition", ErrGraph)
	}
	x, err = leaf(x)
	if err != nil {
		return nil, err
	}
	cond, err = leaf(cond)
	if err != nil {
		return nil, err
	}
	bits, err := cond.Data()
	if err != nil {
		return nil, err
	}
	var idxs []int64
	for i, v := range bits {
		if v != 0 {
			idxs = append(idxs, int64(i))
		}
	}
	axis := int(node.attributeInteger("axis", 0))
	return gatherTensors(x, idxs, axis)
}

func applySplit[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, integerShapes map[string][]int64) (*ndarray.Tensor[T], error) {
	x, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	x, err = leaf(x)
	if err != nil {
		return nil, err
	}
	n := len(node.Outputs)
	if n < 1 {
		return nil, ErrOp
	}
	shape := x.Shape()
	axis := int(node.attributeInteger("axis", 0))
	if axis < 0 {
		axis += len(shape)
	}
	dim := shape[axis]
	splits := integerShapes[""]
	if len(node.Inputs) > 1 {
		splits = integerShapes[node.Inputs[1]]
	}
	if len(splits) == 0 {
		if dim%n != 0 {
			return nil, ndarray.ErrShape
		}
		part := dim / n
		splits = make([]int64, n)
		for i := range splits {
			splits[i] = int64(part)
		}
	}
	if len(splits) != n {
		return nil, ndarray.ErrShape
	}
	off := 0
	var first *ndarray.Tensor[T]
	for i, s := range splits {
		end := off + int(s)
		part, err := x.Shrink(nn.AxisShrink(shape, axis, off, end))
		if err != nil {
			return nil, err
		}
		if i == 0 {
			first = part
		} else if i < len(node.Outputs) {
			values[node.Outputs[i]] = part
		}
		off = end
	}
	return first, nil
}

func applyArgMinMax[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, min bool) (*ndarray.Tensor[T], error) {
	x, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	x, err = leaf(x)
	if err != nil {
		return nil, err
	}
	shape := x.Shape()
	axis := int(node.attributeInteger("axis", 0))
	if axis < 0 {
		axis += len(shape)
	}
	keepdims := node.attributeInteger("keepdims", 1) != 0
	selectLast := node.attributeInteger("select_last_index", 0) != 0
	dim := shape[axis]
	best, err := x.Shrink(nn.AxisShrink(shape, axis, 0, 1))
	if err != nil {
		return nil, err
	}
	idx, err := ndarray.Zeros[T](best.Shape())
	if err != nil {
		return nil, err
	}
	for i := 1; i < dim; i++ {
		cell, err := x.Shrink(nn.AxisShrink(shape, axis, i, i+1))
		if err != nil {
			return nil, err
		}
		var better *ndarray.Tensor[int32]
		if min {
			better = cell.CmpLt(best)
			if selectLast {
				better = better.Or(cell.Equal(best))
			}
		} else {
			better = best.CmpLt(cell)
			if selectLast {
				better = better.Or(cell.Equal(best))
			}
		}
		best = better.Where(cell, best)
		idx = better.Where(ndarray.Const(T(i)), idx)
	}
	if keepdims {
		return idx, nil
	}
	out := make(ndarray.Shape, 0, len(idx.Shape())-1)
	for i, d := range idx.Shape() {
		if i == axis {
			continue
		}
		out = append(out, d)
	}
	return withView(idx, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Reshape(out)
	})
}

func applyOneHot[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, integerShapes map[string][]int64) (*ndarray.Tensor[T], error) {
	if len(node.Inputs) < 3 {
		return nil, ErrOp
	}
	indices, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	depthT, ok := values[node.Inputs[1]]
	if !ok {
		return nil, fmt.Errorf("%w: depth", ErrGraph)
	}
	vals, ok := values[node.Inputs[2]]
	if !ok {
		return nil, fmt.Errorf("%w: values", ErrGraph)
	}
	depthT, err = leaf(depthT)
	if err != nil {
		return nil, err
	}
	vals, err = leaf(vals)
	if err != nil {
		return nil, err
	}
	dd, err := depthT.Data()
	if err != nil || len(dd) == 0 {
		return nil, fmt.Errorf("%w: depth", ErrOp)
	}
	depth := int(dd[0])
	if depth < 1 {
		return nil, ndarray.ErrShape
	}
	vd, err := vals.Data()
	if err != nil || len(vd) < 2 {
		return nil, fmt.Errorf("%w: values", ErrOp)
	}
	off, on := vd[0], vd[1]
	indices, err = leaf(indices)
	if err != nil {
		return nil, err
	}
	inShape := indices.Shape()
	axis := int(node.attributeInteger("axis", -1))
	outRank := len(inShape) + 1
	if axis < 0 {
		axis += outRank
	}
	if axis < 0 || axis >= outRank {
		return nil, ndarray.ErrShape
	}
	outShape := make(ndarray.Shape, outRank)
	copy(outShape[:axis], inShape[:axis])
	outShape[axis] = depth
	copy(outShape[axis+1:], inShape[axis:])
	idx, err := unsqueezeShape(inShape, []int64{int64(axis)})
	if err != nil {
		return nil, err
	}
	indices, err = withView(indices, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Reshape(idx)
	})
	if err != nil {
		return nil, err
	}
	indices, err = withView(indices, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Expand(outShape)
	})
	if err != nil {
		return nil, err
	}
	neg := indices.CmpLt(ndarray.Const(T(0)))
	indices = neg.Where(indices.Add(ndarray.Const(T(depth))), indices)
	c := ndarray.Coord(axis, outShape).Cast[T]()
	mask := c.Equal(indices)
	return mask.Where(ndarray.Const(on), ndarray.Const(off)), nil
}

func applyScatterElements[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	data, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	if len(node.Inputs) < 3 {
		return nil, ErrOp
	}
	idx, ok := values[node.Inputs[1]]
	if !ok {
		return nil, fmt.Errorf("%w: indices", ErrGraph)
	}
	updates, ok := values[node.Inputs[2]]
	if !ok {
		return nil, fmt.Errorf("%w: updates", ErrGraph)
	}
	data, err = leaf(data)
	if err != nil {
		return nil, err
	}
	idx, err = leaf(idx)
	if err != nil {
		return nil, err
	}
	updates, err = leaf(updates)
	if err != nil {
		return nil, err
	}
	shape := data.Shape()
	axis := int(node.attributeInteger("axis", 0))
	if axis < 0 {
		axis += len(shape)
	}
	if axis < 0 || axis >= len(shape) {
		return nil, ndarray.ErrShape
	}
	dim := shape[axis]
	neg := idx.CmpLt(ndarray.Const(T(0)))
	idx = neg.Where(idx.Add(ndarray.Const(T(dim))), idx)
	reduction := node.attributeString("reduction")
	parts := make([]*ndarray.Tensor[T], dim)
	zero := ndarray.Const(T(0))
	one := ndarray.Const(T(1))
	for k := range dim {
		slice, err := data.Shrink(nn.AxisShrink(shape, axis, k, k+1))
		if err != nil {
			return nil, err
		}
		mk := idx.Equal(ndarray.Const(T(k)))
		hit, err := reduceAxis(bool01[T](mk), axis, (*ndarray.Tensor[T]).Max)
		if err != nil {
			return nil, err
		}
		expanded, err := expandLike(slice, updates)
		if err != nil {
			return nil, err
		}
		switch reduction {
		case "add":
			summed, err := reduceAxis(mk.Where(updates, zero), axis, (*ndarray.Tensor[T]).Add)
			if err != nil {
				return nil, err
			}
			parts[k] = slice.Add(summed)
		case "mul":
			prod, err := reduceAxis(mk.Where(updates, one), axis, (*ndarray.Tensor[T]).Mul)
			if err != nil {
				return nil, err
			}
			parts[k] = slice.Mul(prod)
		case "max":
			mx, err := reduceAxis(mk.Where(updates, expanded), axis, (*ndarray.Tensor[T]).Max)
			if err != nil {
				return nil, err
			}
			parts[k] = mx
		case "min":
			mn, err := reduceAxis(mk.Where(updates, expanded), axis, minimum[T])
			if err != nil {
				return nil, err
			}
			parts[k] = mn
		default:
			summed, err := reduceAxis(mk.Where(updates, zero), axis, (*ndarray.Tensor[T]).Add)
			if err != nil {
				return nil, err
			}
			parts[k] = hit.CmpNe(zero).Where(summed, slice)
		}
	}
	return concatTensors(parts, axis)
}

func applyGatherND[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, integerShapes map[string][]int64) (*ndarray.Tensor[T], error) {
	if node.attributeInteger("batch_dims", 0) != 0 {
		return nil, fmt.Errorf("%w: batch_dims", ErrOp)
	}
	data, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	if len(node.Inputs) < 2 {
		return nil, ErrOp
	}
	idxs, ok := integerShapes[node.Inputs[1]]
	if !ok {
		return nil, fmt.Errorf("%w: indices", ErrOp)
	}
	idx, ok := values[node.Inputs[1]]
	if !ok {
		return nil, fmt.Errorf("%w: indices", ErrGraph)
	}
	data, err = leaf(data)
	if err != nil {
		return nil, err
	}
	idxShape := idx.Shape()
	if len(idxShape) < 1 {
		return nil, ndarray.ErrShape
	}
	k := idxShape[len(idxShape)-1]
	dataShape := data.Shape()
	if k > len(dataShape) {
		return nil, ndarray.ErrShape
	}
	n := 1
	for _, d := range idxShape[:len(idxShape)-1] {
		n *= d
	}
	if n*k != len(idxs) {
		return nil, ndarray.ErrShape
	}
	rest := dataShape[k:]
	outRank := len(idxShape) - 1 + len(rest)
	parts := make([]*ndarray.Tensor[T], n)
	for i := range n {
		ranges := make([][2]int, len(dataShape))
		for a := range dataShape {
			ranges[a] = [2]int{0, dataShape[a]}
		}
		for a := range k {
			coord := int(idxs[i*k+a])
			if coord < 0 {
				coord += dataShape[a]
			}
			if coord < 0 || coord >= dataShape[a] {
				return nil, ndarray.ErrShape
			}
			ranges[a] = [2]int{coord, coord + 1}
		}
		cell, err := data.Shrink(ranges)
		if err != nil {
			return nil, err
		}
		flat := ndarray.Shape{1}
		flat = append(flat, rest...)
		cell, err = withView(cell, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
			return t.Reshape(flat)
		})
		if err != nil {
			return nil, err
		}
		parts[i] = cell
	}
	out, err := concatTensors(parts, 0)
	if err != nil {
		return nil, err
	}
	outShape := make(ndarray.Shape, 0, outRank)
	outShape = append(outShape, idxShape[:len(idxShape)-1]...)
	outShape = append(outShape, rest...)
	if len(outShape) == 0 {
		outShape = ndarray.Shape{1}
	}
	return withView(out, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Reshape(outShape)
	})
}

func applyReverseSequence[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, integerShapes map[string][]int64) (*ndarray.Tensor[T], error) {
	x, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	if len(node.Inputs) < 2 {
		return nil, ErrOp
	}
	lens, ok := integerShapes[node.Inputs[1]]
	if !ok {
		return nil, fmt.Errorf("%w: sequence_lens", ErrOp)
	}
	x, err = leaf(x)
	if err != nil {
		return nil, err
	}
	shape := x.Shape()
	batchAxis := int(node.attributeInteger("batch_axis", 1))
	timeAxis := int(node.attributeInteger("time_axis", 0))
	if batchAxis < 0 {
		batchAxis += len(shape)
	}
	if timeAxis < 0 {
		timeAxis += len(shape)
	}
	if batchAxis < 0 || timeAxis < 0 || batchAxis >= len(shape) || timeAxis >= len(shape) || batchAxis == timeAxis {
		return nil, ndarray.ErrShape
	}
	batches := shape[batchAxis]
	times := shape[timeAxis]
	batchParts := make([]*ndarray.Tensor[T], batches)
	for b := range batches {
		n := times
		if b < len(lens) {
			n = int(lens[b])
		}
		if n < 0 {
			n = 0
		}
		if n > times {
			n = times
		}
		batch, err := x.Shrink(nn.AxisShrink(shape, batchAxis, b, b+1))
		if err != nil {
			return nil, err
		}
		timeParts := make([]*ndarray.Tensor[T], times)
		for t := range times {
			src := t
			if t < n {
				src = n - 1 - t
			}
			cell, err := batch.Shrink(nn.AxisShrink(batch.Shape(), timeAxis, src, src+1))
			if err != nil {
				return nil, err
			}
			timeParts[t] = cell
		}
		cat, err := concatTensors(timeParts, timeAxis)
		if err != nil {
			return nil, err
		}
		batchParts[b] = cat
	}
	return concatTensors(batchParts, batchAxis)
}

func applyScatterND[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, integerShapes map[string][]int64) (*ndarray.Tensor[T], error) {
	data, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	if len(node.Inputs) < 3 {
		return nil, ErrOp
	}
	idxs, ok := integerShapes[node.Inputs[1]]
	if !ok {
		return nil, fmt.Errorf("%w: indices", ErrOp)
	}
	updates, ok := values[node.Inputs[2]]
	if !ok {
		return nil, fmt.Errorf("%w: updates", ErrGraph)
	}
	idx, ok := values[node.Inputs[1]]
	if !ok {
		return nil, fmt.Errorf("%w: indices", ErrGraph)
	}
	data, err = leaf(data)
	if err != nil {
		return nil, err
	}
	updates, err = leaf(updates)
	if err != nil {
		return nil, err
	}
	idxShape := idx.Shape()
	if len(idxShape) < 1 {
		return nil, ndarray.ErrShape
	}
	k := idxShape[len(idxShape)-1]
	dataShape := data.Shape()
	if k > len(dataShape) {
		return nil, ndarray.ErrShape
	}
	n := 1
	for _, d := range idxShape[:len(idxShape)-1] {
		n *= d
	}
	if n*k != len(idxs) {
		return nil, ndarray.ErrShape
	}
	reduction := node.attributeString("reduction")
	acc := data
	for i := range n {
		coords := make([]int, k)
		for a := range k {
			coord := int(idxs[i*k+a])
			if coord < 0 {
				coord += dataShape[a]
			}
			coords[a] = coord
		}
		cell, err := updates.Shrink(scatterUpdateRange(updates.Shape(), idxShape[:len(idxShape)-1], i))
		if err != nil {
			return nil, err
		}
		cell, err = withView(cell, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
			return t.Reshape(append(ndarray.Shape{}, dataShape[k:]...))
		})
		if err != nil {
			return nil, err
		}
		acc, err = scatterWrite(acc, coords, cell, reduction)
		if err != nil {
			return nil, err
		}
	}
	return acc, nil
}

func scatterUpdateRange(shape ndarray.Shape, prefix ndarray.Shape, i int) [][2]int {
	out := make([][2]int, len(shape))
	for a, dim := range shape {
		out[a] = [2]int{0, dim}
	}
	stride := 1
	for a := len(prefix) - 1; a >= 0; a-- {
		coord := (i / stride) % prefix[a]
		out[a] = [2]int{coord, coord + 1}
		stride *= prefix[a]
	}
	return out
}

func scatterWrite[T ndarray.Number](data *ndarray.Tensor[T], coords []int, update *ndarray.Tensor[T], reduction string) (*ndarray.Tensor[T], error) {
	if len(coords) == 0 {
		switch reduction {
		case "add":
			return data.Add(update), nil
		case "mul":
			return data.Mul(update), nil
		case "max":
			return data.Max(update), nil
		case "min":
			return minimum(data, update), nil
		default:
			return update, nil
		}
	}
	data, err := leaf(data)
	if err != nil {
		return nil, err
	}
	shape := data.Shape()
	a := coords[0]
	if a < 0 || a >= shape[0] {
		return nil, ndarray.ErrShape
	}
	parts := make([]*ndarray.Tensor[T], shape[0])
	for i := range shape[0] {
		cell, err := data.Shrink(nn.AxisShrink(shape, 0, i, i+1))
		if err != nil {
			return nil, err
		}
		if i == a {
			squeezed, err := withView(cell, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
				return t.Reshape(shape[1:])
			})
			if err != nil {
				return nil, err
			}
			written, err := scatterWrite(squeezed, coords[1:], update, reduction)
			if err != nil {
				return nil, err
			}
			cell, err = withView(written, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
				return t.Reshape(append(ndarray.Shape{1}, written.Shape()...))
			})
			if err != nil {
				return nil, err
			}
		}
		parts[i] = cell
	}
	return concatTensors(parts, 0)
}

func applyNonZero[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	x, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	x, err = leaf(x)
	if err != nil {
		return nil, err
	}
	data, err := x.Data()
	if err != nil {
		return nil, err
	}
	shape := x.Shape()
	rank := len(shape)
	var coords [][]int
	for i, v := range data {
		if v == 0 {
			continue
		}
		coord := make([]int, rank)
		n := i
		for a := rank - 1; a >= 0; a-- {
			coord[a] = n % shape[a]
			n /= shape[a]
		}
		coords = append(coords, coord)
	}
	nnz := len(coords)
	out := make([]T, rank*nnz)
	for i, coord := range coords {
		for a, c := range coord {
			out[a*nnz+i] = T(c)
		}
	}
	return ndarray.New(out, ndarray.Shape{rank, nnz})
}

func applyResize[T ndarray.Number](values map[string]*ndarray.Tensor[T], node Node, integerShapes map[string][]int64) (*ndarray.Tensor[T], error) {
	mode := node.attributeString("mode")
	if mode != "" && mode != "nearest" {
		return nil, fmt.Errorf("%w: resize %s", ErrOp, mode)
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
	scales := make([]int, len(shape))
	for i := range scales {
		scales[i] = 1
	}
	if len(node.Inputs) > 3 && node.Inputs[3] != "" {
		if sizes, ok := integerShapes[node.Inputs[3]]; ok {
			off := len(shape) - len(sizes)
			if off < 0 {
				return nil, ndarray.ErrShape
			}
			for i, sz := range sizes {
				dim := shape[off+i]
				if dim == 0 || int(sz)%dim != 0 {
					return nil, fmt.Errorf("%w: resize size", ErrOp)
				}
				scales[off+i] = int(sz) / dim
			}
		}
	} else if len(node.Inputs) > 2 && node.Inputs[2] != "" {
		t, ok := values[node.Inputs[2]]
		if !ok {
			return nil, fmt.Errorf("%w: scales", ErrGraph)
		}
		t, err = leaf(t)
		if err != nil {
			return nil, err
		}
		d, err := t.Data()
		if err != nil {
			return nil, err
		}
		off := len(shape) - len(d)
		if off < 0 {
			return nil, ndarray.ErrShape
		}
		for i, s := range d {
			sc := int(s)
			if float32(sc) != float32(s) || sc < 1 {
				return nil, fmt.Errorf("%w: resize scale", ErrOp)
			}
			scales[off+i] = sc
		}
	} else if attr := node.attributeFloats("scales"); len(attr) == len(shape) {
		for i, s := range attr {
			sc := int(s)
			if float32(sc) != s || sc < 1 {
				return nil, fmt.Errorf("%w: resize scale", ErrOp)
			}
			scales[i] = sc
		}
	} else {
		return nil, ErrOp
	}
	return upsampleInteger(x, scales)
}

func upsampleInteger[T ndarray.Number](x *ndarray.Tensor[T], scales []int) (*ndarray.Tensor[T], error) {
	for i, s := range scales {
		if s == 1 {
			continue
		}
		shape := x.Shape()
		inserted := make(ndarray.Shape, 0, len(shape)+1)
		inserted = append(inserted, shape[:i+1]...)
		inserted = append(inserted, 1)
		inserted = append(inserted, shape[i+1:]...)
		reshaped, err := withView(x, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
			return t.Reshape(inserted)
		})
		if err != nil {
			return nil, err
		}
		expanded := inserted.Clone()
		expanded[i+1] = s
		x, err = withView(reshaped, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
			return t.Expand(expanded)
		})
		if err != nil {
			return nil, err
		}
		merged := shape.Clone()
		merged[i] *= s
		x, err = withView(x, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
			return t.Reshape(merged)
		})
		if err != nil {
			return nil, err
		}
	}
	return x, nil
}
