package vulkanwindow

import (
	"context"
	"errors"
	"fmt"
	"image"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/ndeval"
	"github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ndarray"
)

// ID is the driver registry id. gui.Open asks for this id. window.Open does not.
const ID = "vulkan_window"

type factory struct{}

func (factory) ID() string   { return ID }
func (factory) Name() string { return "Vulkan window" }
func (factory) Weight() int  { return 10 }

func (factory) CheckCompatibility(ctx context.Context) error {
	switch runtime.GOOS {
	case "linux":
		return driver.RequireEnv(ctx, "DISPLAY")
	case "windows", "darwin":
		return nil
	default:
		return fmt.Errorf("%w: %s", driver.ErrIncompatible, runtime.GOOS)
	}
}

func (factory) New(context.Context) (window.Driver, error) { return opener{}, nil }

type opener struct{}

// Open opens the swapchain window. gui.Open uses it when this driver is available.
func Open(ctx context.Context, cfg window.Config) (window.Window, error) {
	return opener{}.Open(ctx, cfg)
}

func (opener) Open(ctx context.Context, cfg window.Config) (window.Window, error) {
	w, h, err := cfg.Size()
	if err != nil {
		return nil, err
	}
	title := cfg.Title
	if title == "" {
		title = "lewkit"
	}
	screen, err := vulkan.OpenScreen(ctx, w, h, title)
	if err != nil {
		return nil, err
	}
	return &win{
		Buffer:    window.NewBuffer(w, h),
		screen:    screen,
		evaluator: ndeval.Bind(screen.Device()),
	}, nil
}

type win struct {
	*window.Buffer
	screen    vulkan.Screen
	evaluator ndarray.Evaluator
}

func (w *win) Evaluator(context.Context) (ndarray.Evaluator, error) {
	if w == nil || w.evaluator == nil {
		return nil, window.ErrClosed
	}
	return w.evaluator, nil
}

func (w *win) PaintDevice(ctx context.Context, tensor *ndarray.Tensor[uint8], evaluator ndarray.Evaluator) error {
	if w == nil || w.screen == nil {
		return window.ErrClosed
	}
	if evaluator == nil {
		evaluator = w.evaluator
	}
	return ndeval.Paint(ctx, evaluator, tensor, w.screen)
}

func (w *win) Draw() error {
	if w == nil || w.Buffer == nil {
		return window.ErrClosed
	}
	return nil
}

func (w *win) Close() error {
	if w == nil {
		return nil
	}
	var err error
	if w.evaluator != nil {
		err = w.evaluator.Close()
		w.evaluator = nil
	}
	if w.screen != nil {
		err = errors.Join(err, w.screen.Close())
		w.screen = nil
	}
	if w.Buffer != nil {
		err = errors.Join(err, w.Buffer.Close())
	}
	return err
}

func (w *win) Resize(size image.Point) error {
	return window.ErrSize
}
