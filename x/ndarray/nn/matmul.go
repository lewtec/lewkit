package nn

import (
	"fmt"

	"github.com/lewtec/lewkit/x/ndarray"
)

// MatrixMultiply is numpy-style a @ b. 1D operands are promoted and squeezed.
// Leading dims broadcast.
func MatrixMultiply[T ndarray.Number](a, b *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
	if a == nil || b == nil {
		return nil, ndarray.ErrOp
	}
	aShape, bShape := a.Shape(), b.Shape()
	if len(aShape) == 0 || len(bShape) == 0 {
		return nil, fmt.Errorf("%w: a %v b %v", ndarray.ErrShape, aShape, bShape)
	}
	squeezeRows, squeezeColumns := false, false
	var err error
	if len(aShape) == 1 {
		a, err = a.Reshape(ndarray.Shape{1, aShape[0]})
		if err != nil {
			return nil, err
		}
		squeezeRows = true
		aShape = a.Shape()
	}
	if len(bShape) == 1 {
		b, err = b.Reshape(ndarray.Shape{bShape[0], 1})
		if err != nil {
			return nil, err
		}
		squeezeColumns = true
		bShape = b.Shape()
	}
	aBatch, bBatch := aShape[:len(aShape)-2], bShape[:len(bShape)-2]
	rows, inner := aShape[len(aShape)-2], aShape[len(aShape)-1]
	innerB, columns := bShape[len(bShape)-2], bShape[len(bShape)-1]
	if inner != innerB {
		return nil, fmt.Errorf("%w: a %v b %v", ndarray.ErrShape, aShape, bShape)
	}
	batch, err := broadcastPrefix(aBatch, bBatch)
	if err != nil {
		return nil, fmt.Errorf("%w: a %v b %v", err, aShape, bShape)
	}
	outRank := len(batch) + 2
	a, err = prependOnes(a, outRank)
	if err != nil {
		return nil, err
	}
	b, err = prependOnes(b, outRank)
	if err != nil {
		return nil, err
	}
	aMat := append(batch.Clone(), rows, inner)
	bMat := append(batch.Clone(), inner, columns)
	a, err = a.Expand(aMat)
	if err != nil {
		return nil, err
	}
	b, err = b.Expand(bMat)
	if err != nil {
		return nil, err
	}
	outShape := append(batch.Clone(), rows, columns)
	if inner == 0 {
		return ndarray.Zeros[T](outShape)
	}
	var accumulated *ndarray.Tensor[T]
	for i := range inner {
		row, err := a.Shrink(axisShrink(aMat, len(aMat)-1, i, i+1))
		if err != nil {
			return nil, err
		}
		row, err = row.Expand(outShape)
		if err != nil {
			return nil, err
		}
		column, err := b.Shrink(axisShrink(bMat, len(bMat)-2, i, i+1))
		if err != nil {
			return nil, err
		}
		column, err = column.Expand(outShape)
		if err != nil {
			return nil, err
		}
		term := row.Mul(column)
		if accumulated == nil {
			accumulated = term
		} else {
			accumulated = accumulated.Add(term)
		}
	}
	want := outShape
	switch {
	case squeezeRows && squeezeColumns:
		want = ndarray.Shape{}
	case squeezeRows:
		want = append(batch.Clone(), columns)
	case squeezeColumns:
		want = append(batch.Clone(), rows)
	}
	if accumulated.Shape().Equal(want) {
		return accumulated, nil
	}
	out, err := accumulated.Reshape(want)
	if err != nil {
		return accumulated, nil
	}
	return out, nil
}
