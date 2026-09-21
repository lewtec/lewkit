package nn

import (
	"fmt"

	"github.com/lewtec/lewkit/x/ndarray"
)

// Convolution2D is y[n,oc,oh,ow] = sum over input channel and kernel of
// x[n,ic,oh*sh+kh,ow*sw+kw] * w[oc,ic,kh,kw]. x is NCHW, w is (out, in, kh, kw).
// pads is [heightBefore, widthBefore, heightAfter, widthAfter].
func Convolution2D[T ndarray.Number](input, weight *ndarray.Tensor[T], pads []int, strideHeight, strideWidth int) (*ndarray.Tensor[T], error) {
	if input == nil || weight == nil {
		return nil, ndarray.ErrOp
	}
	if strideHeight <= 0 {
		strideHeight = 1
	}
	if strideWidth <= 0 {
		strideWidth = 1
	}
	inputShape, weightShape := input.Shape(), weight.Shape()
	if len(inputShape) != 4 || len(weightShape) != 4 {
		return nil, fmt.Errorf("%w: input %v weight %v", ndarray.ErrShape, inputShape, weightShape)
	}
	if len(pads) != 4 {
		return nil, fmt.Errorf("%w: pads %v", ndarray.ErrPad, pads)
	}
	batch, channelsIn, height, width := inputShape[0], inputShape[1], inputShape[2], inputShape[3]
	channelsOut, weightIn, kernelHeight, kernelWidth := weightShape[0], weightShape[1], weightShape[2], weightShape[3]
	if channelsIn != weightIn {
		return nil, fmt.Errorf("%w: input %v weight %v", ndarray.ErrShape, inputShape, weightShape)
	}
	heightOut := (height+pads[0]+pads[2]-kernelHeight)/strideHeight + 1
	widthOut := (width+pads[1]+pads[3]-kernelWidth)/strideWidth + 1
	if heightOut <= 0 || widthOut <= 0 {
		return nil, fmt.Errorf("%w: input %v pads %v strides %d %d", ndarray.ErrShape, inputShape, pads, strideHeight, strideWidth)
	}
	padded, err := padNCHW(input, coverStrided(height, width, kernelHeight, kernelWidth, strideHeight, strideWidth, heightOut, widthOut, pads))
	if err != nil {
		return nil, err
	}
	paddedShape := padded.Shape()
	outShape := ndarray.Shape{batch, channelsOut, heightOut, widthOut}
	var accumulated *ndarray.Tensor[T]
	for inputChannel := range channelsIn {
		channel, err := padded.Shrink([][2]int{{0, batch}, {inputChannel, inputChannel + 1}, {0, paddedShape[2]}, {0, paddedShape[3]}})
		if err != nil {
			return nil, err
		}
		for kernelY := range kernelHeight {
			for kernelX := range kernelWidth {
				patch, err := stridedNCHW(channel, kernelY, kernelX, heightOut, widthOut, strideHeight, strideWidth)
				if err != nil {
					return nil, err
				}
				patch, err = patch.Reshape(ndarray.Shape{batch, 1, heightOut, widthOut})
				if err != nil {
					return nil, err
				}
				patch, err = patch.Expand(outShape)
				if err != nil {
					return nil, err
				}
				coeff, err := weight.Shrink([][2]int{{0, channelsOut}, {inputChannel, inputChannel + 1}, {kernelY, kernelY + 1}, {kernelX, kernelX + 1}})
				if err != nil {
					return nil, err
				}
				coeff, err = coeff.Reshape(ndarray.Shape{1, channelsOut, 1, 1})
				if err != nil {
					return nil, err
				}
				coeff, err = coeff.Expand(outShape)
				if err != nil {
					return nil, err
				}
				term := patch.Mul(coeff)
				if accumulated == nil {
					accumulated = term
				} else {
					accumulated = accumulated.Add(term)
				}
			}
		}
	}
	if accumulated == nil {
		return ndarray.Zeros[T](outShape)
	}
	return accumulated, nil
}
