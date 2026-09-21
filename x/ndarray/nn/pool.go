package nn

import (
	"fmt"

	"github.com/lewtec/lewkit/x/ndarray"
)

// MaximumPool2D is a window max over NCHW. pads is
// [heightBefore, widthBefore, heightAfter, widthAfter]. Leftover spatial
// edges are dropped (floor).
func MaximumPool2D[T ndarray.Number](input *ndarray.Tensor[T], kernelHeight, kernelWidth, strideHeight, strideWidth int, pads []int) (*ndarray.Tensor[T], error) {
	w, err := poolWindowOf(input, poolKernel{
		kernelHeight: kernelHeight,
		kernelWidth:  kernelWidth,
		strideHeight: strideHeight,
		strideWidth:  strideWidth,
	}, pads)
	if err != nil {
		return nil, err
	}
	padded, err := padNCHWMax(input, []int{w.heightBefore, w.widthBefore, w.heightAfter, w.widthAfter})
	if err != nil {
		return nil, err
	}
	var accumulated *ndarray.Tensor[T]
	for kernelY := range kernelHeight {
		for kernelX := range kernelWidth {
			cell, err := stridedNCHW(padded, kernelY, kernelX, w.heightOut, w.widthOut, w.strideHeight, w.strideWidth)
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
	w, err := poolWindowOf(input, poolKernel{
		kernelHeight: kernelHeight,
		kernelWidth:  kernelWidth,
		strideHeight: strideHeight,
		strideWidth:  strideWidth,
	}, pads)
	if err != nil {
		return nil, err
	}
	padded, err := padNCHW(input, []int{w.heightBefore, w.widthBefore, w.heightAfter, w.widthAfter})
	if err != nil {
		return nil, err
	}
	var sum *ndarray.Tensor[T]
	for kernelY := range kernelHeight {
		for kernelX := range kernelWidth {
			cell, err := stridedNCHW(padded, kernelY, kernelX, w.heightOut, w.widthOut, w.strideHeight, w.strideWidth)
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
	cover := []int{w.heightBefore, w.widthBefore, w.heightAfter, w.widthAfter}
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
			cell, err := stridedNCHW(onesPad, kernelY, kernelX, w.heightOut, w.widthOut, w.strideHeight, w.strideWidth)
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

type poolKernel struct {
	kernelHeight int
	kernelWidth  int
	strideHeight int
	strideWidth  int
}

type poolWindow struct {
	strideHeight int
	strideWidth  int
	heightOut    int
	widthOut     int
	heightBefore int
	widthBefore  int
	heightAfter  int
	widthAfter   int
}

func poolWindowOf[T ndarray.Number](input *ndarray.Tensor[T], kernel poolKernel, pads []int) (poolWindow, error) {
	kernelHeight := kernel.kernelHeight
	kernelWidth := kernel.kernelWidth
	strideHeight := kernel.strideHeight
	strideWidth := kernel.strideWidth
	if input == nil {
		return poolWindow{}, ndarray.ErrOp
	}
	if kernelHeight <= 0 || kernelWidth <= 0 {
		return poolWindow{}, ndarray.ErrOp
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
		return poolWindow{}, fmt.Errorf("%w: pads %v", ndarray.ErrPad, pads)
	}
	inputShape := input.Shape()
	if len(inputShape) != 4 {
		return poolWindow{}, fmt.Errorf("%w: %v", ndarray.ErrShape, inputShape)
	}
	height, width := inputShape[2], inputShape[3]
	heightOut := (height+pads[0]+pads[2]-kernelHeight)/strideHeight + 1
	widthOut := (width+pads[1]+pads[3]-kernelWidth)/strideWidth + 1
	if heightOut <= 0 || widthOut <= 0 {
		return poolWindow{}, fmt.Errorf("%w: %v kernel %d %d", ndarray.ErrShape, inputShape, kernelHeight, kernelWidth)
	}
	heightBefore, heightAfter := ndarray.CoverPad(height, kernelHeight, strideHeight, heightOut, pads[0], pads[2])
	widthBefore, widthAfter := ndarray.CoverPad(width, kernelWidth, strideWidth, widthOut, pads[1], pads[3])
	return poolWindow{
		strideHeight: strideHeight,
		strideWidth:  strideWidth,
		heightOut:    heightOut,
		widthOut:     widthOut,
		heightBefore: heightBefore,
		widthBefore:  widthBefore,
		heightAfter:  heightAfter,
		widthAfter:   widthAfter,
	}, nil
}

func elementDiv[T ndarray.Number](a, b *ndarray.Tensor[T]) *ndarray.Tensor[T] {
	switch any(T(0)).(type) {
	case float32:
		return a.Div(b)
	default:
		return a.IDiv(b)
	}
}
