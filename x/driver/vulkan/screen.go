package vulkan

import (
	"context"

	ffivulkan "github.com/lewtec/lewkit/x/ffi/native/vulkan"
)

// Screen is a window whose swapchain lives on one compute device.
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
