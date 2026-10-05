package gui

import (
	"context"

	"github.com/lewtec/lewkit/x/ndarray"
)

// Raster paints a float tensor as the frame behind later fills.
// Pixels are (height, width, 4) in 0..255. A shape of {1, 1, 4} follows
// the frame size. A leaf of the frame shape is copied to the underlay.
// Any other expression is evaluated for that underlay.
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
	picture.Backdrop(raster.Pixels)
	return accumulatorOf(picture)
}

func (picture *Picture) fuseRaster() {
	if picture == nil || picture.raster == nil || picture.recordOnly {
		return
	}
	shape := picture.raster.Shape()
	if shape != nil && !shape.Equal(ndarray.Shape{1, 1, 4}) {
		return
	}
	picture.base = picture.raster
	picture.slots = nil
	picture.composites = nil
	picture.inkedFrom = nil
	picture.replayFills()
}

func (picture *Picture) rasterBytes(ctx context.Context, evaluator ndarray.Evaluator, width, height int) ([]byte, error) {
	if picture == nil || picture.raster == nil || width < 1 || height < 1 {
		return nil, nil
	}
	if packed := leafUnderlay(picture.raster, width, height); packed != nil {
		return packed, nil
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

// leafUnderlay quantizes a stored frame to RGBA8.
// Present uploads those bytes. The fill-list shaders draw them on Metal and Vulkan.
func leafUnderlay(pixels *ndarray.Tensor[float32], width, height int) []byte {
	if pixels == nil || width < 1 || height < 1 {
		return nil
	}
	shape := pixels.Shape()
	if shape == nil || !shape.Equal(ndarray.Shape{height, width, 4}) {
		return nil
	}
	buf, err := pixels.Data()
	if err != nil {
		return nil
	}
	need := width * height * 4
	if len(buf) < need {
		return nil
	}
	out := make([]byte, need)
	for i, v := range buf[:need] {
		n := int32(v)
		if n < 0 {
			continue
		}
		if n > 255 {
			n = 255
		}
		out[i] = byte(n)
	}
	return out
}
