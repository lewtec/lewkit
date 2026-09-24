package gui

import (
	"context"

	"github.com/lewtec/lewkit/x/ndarray"
)

// Raster paints a float tensor as the frame behind later fills.
// Pixels are (height, width, 4) in 0..255. A shape of {1, 1, 4} follows
// the frame size. A shaped tensor is resized to the frame on the swapchain.
type Raster struct {
	Pixels *ndarray.Tensor[float32]
	size   Size
}

func (raster *Raster) Layout(constraints BoxConstraints) Size {
	if raster == nil {
		return Size{}
	}
	raster.size = constraints.Constrain(Size{Width: constraints.MaxWidth, Height: constraints.MaxHeight})
	return raster.size
}

func (raster *Raster) Paint(origin Offset, clip Rect, picture *Picture) *ndarray.Tensor[float32] {
	if raster == nil || picture == nil || raster.Pixels == nil {
		return accumulatorOf(picture)
	}
	picture.useRaster(raster.Pixels)
	return accumulatorOf(picture)
}

func (picture *Picture) useRaster(src *ndarray.Tensor[float32]) {
	if picture == nil || src == nil {
		return
	}
	picture.raster = src
}

func (picture *Picture) fuseRaster() {
	if picture == nil || picture.raster == nil || picture.recordOnly {
		return
	}
	shape := picture.raster.Shape()
	if shape != nil && !shape.Equal(ndarray.Shape{1, 1, 4}) {
		return
	}
	fills := append([]Draw(nil), picture.fills...)
	picture.base = picture.raster
	picture.slots = nil
	picture.composites = nil
	picture.inkedFrom = nil
	picture.fills = picture.fills[:0]
	picture.fillCount = 0
	picture.accumulator = picture.base
	for _, fill := range fills {
		picture.over(fill)
	}
}

func (picture *Picture) rasterBytes(ctx context.Context, evaluator ndarray.Evaluator, width, height int) ([]byte, error) {
	if picture == nil || picture.raster == nil || width < 1 || height < 1 {
		return nil, nil
	}
	if evaluator == nil {
		evaluator = ndarray.CPU
	}
	if picture.rasterCast == nil || picture.rasterFrom != picture.raster {
		picture.rasterCast = picture.raster.Cast[uint8]()
		picture.rasterFrom = picture.raster
	}
	view := picture.rasterCast
	if err := view.Resize(ndarray.Shape{height, width, 4}); err != nil {
		return nil, err
	}
	buffer := make([]uint8, height*width*4)
	if err := view.Eval(ctx, evaluator, buffer); err != nil {
		return nil, err
	}
	return buffer, nil
}
