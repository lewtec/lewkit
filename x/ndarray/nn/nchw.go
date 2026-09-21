package nn

import (
	"github.com/lewtec/lewkit/x/ndarray"
)

func padNCHW[T ndarray.Number](t *ndarray.Tensor[T], pads []int) (*ndarray.Tensor[T], error) {
	if pads[0] == 0 && pads[1] == 0 && pads[2] == 0 && pads[3] == 0 {
		return t, nil
	}
	return t.Pad([][2]int{{0, 0}, {0, 0}, {pads[0], pads[2]}, {pads[1], pads[3]}})
}

func padNCHWMax[T ndarray.Number](t *ndarray.Tensor[T], pads []int) (*ndarray.Tensor[T], error) {
	if pads[0] == 0 && pads[1] == 0 && pads[2] == 0 && pads[3] == 0 {
		return t, nil
	}
	padded, err := padNCHW(t, pads)
	if err != nil {
		return nil, err
	}
	fill := ndarray.Lowest[T]()
	if fill == 0 {
		return padded, nil
	}
	ones, err := ndarray.Ones[T](t.Shape())
	if err != nil {
		return nil, err
	}
	mask, err := padNCHW(ones, pads)
	if err != nil {
		return nil, err
	}
	return mask.CmpNe(ndarray.Const(T(0))).Where(padded, ndarray.Const(fill)), nil
}

func stridedNCHW[T ndarray.Number](t *ndarray.Tensor[T], startHeight, startWidth, heightOut, widthOut, strideHeight, strideWidth int) (*ndarray.Tensor[T], error) {
	if t == nil {
		return nil, ndarray.ErrOp
	}
	shape := t.Shape()
	if len(shape) != 4 {
		return nil, ndarray.ErrShape
	}
	batch, channels := shape[0], shape[1]
	windowHeight := heightOut * strideHeight
	windowWidth := widthOut * strideWidth
	cropped, err := t.Shrink([][2]int{
		{0, batch},
		{0, channels},
		{startHeight, startHeight + windowHeight},
		{startWidth, startWidth + windowWidth},
	})
	if err != nil {
		return nil, err
	}
	shaped, err := cropped.Reshape(ndarray.Shape{batch, channels, heightOut, strideHeight, widthOut, strideWidth})
	if err != nil {
		return nil, err
	}
	cell, err := shaped.Shrink([][2]int{
		{0, batch},
		{0, channels},
		{0, heightOut},
		{0, 1},
		{0, widthOut},
		{0, 1},
	})
	if err != nil {
		return nil, err
	}
	return cell.Reshape(ndarray.Shape{batch, channels, heightOut, widthOut})
}

// AxisShrink returns Shrink ranges: full span on every axis, with axis set to [start, end).
func AxisShrink(shape ndarray.Shape, axis, start, end int) [][2]int {
	out := make([][2]int, len(shape))
	for i, dim := range shape {
		out[i] = [2]int{0, dim}
	}
	out[axis] = [2]int{start, end}
	return out
}

func prependOnes[T ndarray.Number](t *ndarray.Tensor[T], rank int) (*ndarray.Tensor[T], error) {
	shape := t.Shape()
	if len(shape) == rank {
		return t, nil
	}
	if len(shape) > rank {
		return nil, ndarray.ErrShape
	}
	padded := make(ndarray.Shape, rank)
	copy(padded[rank-len(shape):], shape)
	for i := 0; i < rank-len(shape); i++ {
		padded[i] = 1
	}
	return t.Reshape(padded)
}

func broadcastPrefix(left, right ndarray.Shape) (ndarray.Shape, error) {
	rank := max(len(left), len(right))
	out := make(ndarray.Shape, rank)
	for i := range rank {
		dimLeft, dimRight := 1, 1
		if j := i - (rank - len(left)); j >= 0 {
			dimLeft = left[j]
		}
		if j := i - (rank - len(right)); j >= 0 {
			dimRight = right[j]
		}
		switch {
		case dimLeft == dimRight:
			out[i] = dimLeft
		case dimLeft == 1:
			out[i] = dimRight
		case dimRight == 1:
			out[i] = dimLeft
		default:
			return nil, ndarray.ErrShape
		}
	}
	return out, nil
}
