package gui

import (
	"context"

	"github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
)

// Frame is one lowered picture.
// Fills are in paint order. Under and Ink are tightly packed RGBA8,
// Width by Height by 4, or nil. Both slices are valid only for [Canvas.Draw].
type Frame struct {
	Width, Height int
	Fills         []Draw
	Under         []byte
	Ink           []byte
}

// Canvas owns the pixels of one screen.
// Draw paints Under, then Fills, then Ink.
//
// [Vulkan] attaches a Vulkan swapchain. An OpenGL framebuffer attaches
// by implementing the same Draw: upload the fills, the underlay, and the
// ink, draw those quads, and swap buffers.
type Canvas interface {
	Draw(ctx context.Context, frame Frame) error
}

// Play checks the frame and draws it on the attached canvas.
func Play(ctx context.Context, canvas Canvas, frame Frame) error {
	if canvas == nil {
		return ErrView
	}
	if frame.Width < 1 || frame.Height < 1 {
		return ndarray.ErrShape
	}
	if len(frame.Fills) > 0 {
		frame.Fills = append([]Draw(nil), frame.Fills...)
	}
	return canvas.Draw(ctx, frame)
}

// Vulkan attaches a swapchain as a [Canvas].
// The screen keeps its pipelines. This adapter only supplies the frame.
func Vulkan(screen vulkan.Screen) Canvas { return vulkanCanvas{screen} }

type vulkanCanvas struct{ screen vulkan.Screen }

func (canvas vulkanCanvas) Draw(ctx context.Context, frame Frame) error {
	return drawFills(ctx, canvas.screen, frame.Fills, frame.Under, frame.Ink, frame.Width, frame.Height)
}
