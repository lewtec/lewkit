package nn

import (
	"fmt"

	"github.com/lewtec/lewkit/x/ndarray"
)

// MaximumPool2D is a window max over NCHW. pads is
// [heightBefore, widthBefore, heightAfter, widthAfter]. Leftover spatial
// edges are dropped (floor).
func MaximumPool2D[T ndarray.Number](input *ndarray.Tensor[T], kernelHeight, kernelWidth, strideHeight, strideWidth int, pads []int) (*ndarray.Tensor[T], error) {
	if input == nil {
		return nil, ndarray.ErrOp
	}
	if kernelHeight <= 0 || kernelWidth <= 0 {
		return nil, ndarray.ErrOp
	}
	if strideHeight <= 0 {
		strideHeight = 1
	}
	if strideWidth <= 0 {
		strideWidth = 1
	}
	if len(pads) == 0 {
		pads = []int{0, 0, 0, 0}
	}
	if len(pads) != 4 {
		return nil, fmt.Errorf("%w: pads %v", ndarray.ErrPad, pads)
	}
	inputShape := input.Shape()
	if len(inputShape) != 4 {
		return nil, fmt.Errorf("%w: %v", ndarray.ErrShape, inputShape)
	}
	height, width := inputShape[2], inputShape[3]
	heightOut := (height+pads[0]+pads[2]-kernelHeight)/strideHeight + 1
	widthOut := (width+pads[1]+pads[3]-kernelWidth)/strideWidth + 1
	if heightOut <= 0 || widthOut <= 0 {
		return nil, fmt.Errorf("%w: %v kernel %d %d", ndarray.ErrShape, inputShape, kernelHeight, kernelWidth)
	}
	padded, err := padNCHWMax(input, coverStrided(height, width, kernelHeight, kernelWidth, strideHeight, strideWidth, heightOut, widthOut, pads))
	if err != nil {
		return nil, err
	}
	var accumulated *ndarray.Tensor[T]
	for kernelY := range kernelHeight {
		for kernelX := range kernelWidth {
			cell, err := stridedNCHW(padded, kernelY, kernelX, heightOut, widthOut, strideHeight, strideWidth)
			if err != nil {
				return nil, err
			}
			if accumulated == nil {
				accumulated = cell
			} else {
				accumulated = accumulated.Max(cell)
			}
		}
	}
	return accumulated, nil
}

// AveragePool2D is a window mean over NCHW. pads is
// [heightBefore, widthBefore, heightAfter, widthAfter].
func AveragePool2D[T ndarray.Number](input *ndarray.Tensor[T], kernelHeight, kernelWidth, strideHeight, strideWidth int, pads []int, countIncludePad bool) (*ndarray.Tensor[T], error) {
	if input == nil {
		return nil, ndarray.ErrOp
	}
	if kernelHeight <= 0 || kernelWidth <= 0 {
		return nil, ndarray.ErrOp
	}
	if strideHeight <= 0 {
		strideHeight = 1
	}
	if strideWidth <= 0 {
		strideWidth = 1
	}
	if len(pads) == 0 {
		pads = []int{0, 0, 0, 0}
	}
	if len(pads) != 4 {
		return nil, fmt.Errorf("%w: pads %v", ndarray.ErrPad, pads)
	}
	inputShape := input.Shape()
	if len(inputShape) != 4 {
		return nil, fmt.Errorf("%w: %v", ndarray.ErrShape, inputShape)
	}
	height, width := inputShape[2], inputShape[3]
	heightOut := (height+pads[0]+pads[2]-kernelHeight)/strideHeight + 1
	widthOut := (width+pads[1]+pads[3]-kernelWidth)/strideWidth + 1
	if heightOut <= 0 || widthOut <= 0 {
		return nil, fmt.Errorf("%w: %v kernel %d %d", ndarray.ErrShape, inputShape, kernelHeight, kernelWidth)
	}
	cover := coverStrided(height, width, kernelHeight, kernelWidth, strideHeight, strideWidth, heightOut, widthOut, pads)
	padded, err := padNCHW(input, cover)
	if err != nil {
		return nil, err
	}
	var sum *ndarray.Tensor[T]
	for kernelY := range kernelHeight {
		for kernelX := range kernelWidth {
			cell, err := stridedNCHW(padded, kernelY, kernelX, heightOut, widthOut, strideHeight, strideWidth)
			if err != nil {
				return nil, err
			}
			if sum == nil {
				sum = cell
			} else {
				sum = sum.Add(cell)
			}
		}
	}
	if countIncludePad || (cover[0] == 0 && cover[1] == 0 && cover[2] == 0 && cover[3] == 0) {
		return elementDiv(sum, ndarray.Const(T(kernelHeight*kernelWidth))), nil
	}
	ones, err := ndarray.Ones[T](input.Shape())
	if err != nil {
		return nil, err
	}
	onesPad, err := padNCHW(ones, cover)
	if err != nil {
		return nil, err
	}
	var count *ndarray.Tensor[T]
	for kernelY := range kernelHeight {
		for kernelX := range kernelWidth {
			cell, err := stridedNCHW(onesPad, kernelY, kernelX, heightOut, widthOut, strideHeight, strideWidth)
			if err != nil {
				return nil, err
			}
			if count == nil {
				count = cell
			} else {
				count = count.Add(cell)
			}
		}
	}
	return elementDiv(sum, count), nil
}

func elementDiv[T ndarray.Number](a, b *ndarray.Tensor[T]) *ndarray.Tensor[T] {
	switch any(T(0)).(type) {
	case float32:
		return a.Div(b)
	default:
		return a.IDiv(b)
	}
}
