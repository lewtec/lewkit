package gui

import (
	"context"
	"encoding/binary"
	"math"

	"github.com/lewtec/lewkit/x/driver/ndeval"
	"github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
)

// Show records root and draws it on the swapchain. It does not run the fused kernel.
func (picture *Picture) Show(ctx context.Context, screen vulkan.Screen, root Node, size Size) error {
	if picture == nil || root == nil {
		return ErrView
	}
	picture.recordOnly = true
	pixels, err := picture.Render(root, size)
	picture.recordOnly = false
	if err != nil {
		return err
	}
	if pixels != nil && picture.raster != nil {
		return picture.paintMounted(ctx, screen, pixels)
	}
	var ink []byte
	if picture.hadInk && picture.inkRGBA != nil {
		ink = picture.inkRGBA.Pix
	}
	under, err := picture.rasterBytes(ctx, nil, int(size.Width), int(size.Height))
	if err != nil {
		return err
	}
	return Play(ctx, Vulkan(screen), Frame{
		Width: int(size.Width), Height: int(size.Height),
		Fills: picture.fills, Under: under, Ink: ink,
	})
}

// drawFills paints an optional tensor image, then fills, then glyph ink.
func drawFills(ctx context.Context, screen vulkan.Screen, frame Frame) error {
	if screen == nil || frame.Width < 1 || frame.Height < 1 {
		return ndarray.ErrShape
	}
	raw := make([]byte, len(frame.Fills)*64)
	for i, fill := range frame.Fills {
		putFill(raw[i*64:(i+1)*64], fill)
	}
	return screen.Draw(ctx, raw, frame.Under, frame.Ink, frame.Width, frame.Height)
}

func (picture *Picture) paintMounted(ctx context.Context, screen vulkan.Screen, pixels *ndarray.Tensor[uint8]) error {
	if picture.paintEval == nil || !vulkan.Same(picture.paintDevice, screen.Device()) {
		if picture.paintEval != nil {
			_ = picture.paintEval.Close()
		}
		picture.paintEval = ndeval.Bind(screen.Device())
		picture.paintDevice = screen.Device()
	}
	return ndeval.Paint(ctx, picture.paintEval, pixels, screen)
}

func putFill(dst []byte, fill Draw) {
	vals := [16]float32{
		fill.X, fill.Y, fill.Width, fill.Height,
		fill.Red, fill.Green, fill.Blue, fill.Alpha,
		fill.Radius, fill.ClipX, fill.ClipY, fill.ClipWidth,
		fill.ClipHeight, 0, 0, 0,
	}
	for i, v := range vals {
		binary.LittleEndian.PutUint32(dst[i*4:], math.Float32bits(v))
	}
}
