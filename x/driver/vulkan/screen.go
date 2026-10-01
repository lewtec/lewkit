package vulkan

import (
	"context"

	ffivulkan "github.com/lewtec/lewkit/x/ffi/native/vulkan"
)

// Screen is a window whose swapchain lives on one GPU.
type (
	Input = ffivulkan.Input
)

const (
	InputResize  = ffivulkan.InputResize
	InputClose   = ffivulkan.InputClose
	InputPointer = ffivulkan.InputPointer
	InputScroll  = ffivulkan.InputScroll
	InputKey     = ffivulkan.InputKey
	InputExpose  = ffivulkan.InputExpose
)

type Screen interface {
	Device() Device
	Present(buf *Buffer, width, height int, spirv []byte) error
	// Fit waits out the previous frame and resizes the swapchain.
	// JoinPresent appends the present copy onto the open command buffer and submits once.
	Fit(width, height int) error
	JoinPresent(buf *Buffer, width, height int, spirv []byte) error
	Adopt(window uintptr, width, height int) error
	// Draw paints an optional RGBA8 underlay, rounded-rect instances, then glyph ink.
	// instances is 16 float32 values per fill. The fill and ink shaders belong to the screen.
	Draw(ctx context.Context, instances, under, ink []byte, width, height int) error
	OnInput(func(Input))
	Close() error
}

type screen struct {
	binding *ffivulkan.Screen
}

// OpenNative builds a swapchain for a window the caller already owns.
// The native window is not closed with the screen.
func OpenNative(ctx context.Context, kind int, a, b uintptr, width, height int) (Screen, error) {
	binding, err := ffivulkan.OpenNative(ctx, kind, a, b, width, height)
	if err != nil {
		return nil, err
	}
	return &screen{binding: binding}, nil
}

// OpenScreen opens an X11 window and a swapchain on the best present-capable GPU.
func OpenScreen(ctx context.Context, width, height int, title string) (Screen, error) {
	binding, err := ffivulkan.OpenScreen(ctx, width, height, title)
	if err != nil {
		return nil, err
	}
	return &screen{binding: binding}, nil
}

func (s *screen) Device() Device {
	if s == nil || s.binding == nil {
		return nil
	}
	return wrap(s.binding.Device())
}

func (s *screen) Present(buf *Buffer, width, height int, spirv []byte) error {
	if s == nil || s.binding == nil {
		return ffivulkan.ErrClosed
	}
	return s.binding.Present(buf, width, height, spirv)
}

func (s *screen) Fit(width, height int) error {
	if s == nil || s.binding == nil {
		return ffivulkan.ErrClosed
	}
	return s.binding.Fit(width, height)
}

func (s *screen) JoinPresent(buf *Buffer, width, height int, spirv []byte) error {
	if s == nil || s.binding == nil {
		return ffivulkan.ErrClosed
	}
	return s.binding.JoinPresent(buf, width, height, spirv)
}

func (s *screen) Adopt(window uintptr, width, height int) error {
	if s == nil || s.binding == nil {
		return ffivulkan.ErrClosed
	}
	return s.binding.Adopt(window, width, height)
}

func (s *screen) Draw(ctx context.Context, instances, under, ink []byte, width, height int) error {
	if s == nil || s.binding == nil {
		return ffivulkan.ErrClosed
	}
	code, err := drawCode.GetContext(ctx)
	if err == nil {
		err = s.binding.Draw(instances, under, ink, width, height, code.vert, code.frag, code.inkVert, code.inkFrag)
	}
	return err
}

func (s *screen) OnInput(fn func(Input)) {
	if s == nil || s.binding == nil {
		return
	}
	s.binding.OnInput(fn)
}

func (s *screen) Close() error {
	if s == nil || s.binding == nil {
		return nil
	}
	err := s.binding.Close()
	s.binding = nil
	return err
}

// Same reports whether a and b wrap one libvulkan device.
func Same(a, b Device) bool {
	left, ok := a.(*device)
	if !ok || left == nil {
		return false
	}
	right, ok := b.(*device)
	return ok && right != nil && left.binding == right.binding
}
