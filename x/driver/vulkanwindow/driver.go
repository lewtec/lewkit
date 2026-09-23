package vulkanwindow

import (
	"context"
	"errors"
	"fmt"
	"image"
	"runtime"
	"time"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/ndeval"
	"github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/event"
	"github.com/lewtec/lewkit/x/ndarray"
)

// ID is the registry id for this driver. It is not a window.Driver.
const ID = "vulkan_window"

// Driver opens a swapchain screen. Image windows stay on window.Driver.
type Driver interface {
	Open(ctx context.Context, cfg window.Config) (Screen, error)
}

// Screen is the swapchain. It has no image back buffer.
type Screen interface {
	Close() error
	Size() image.Point
	FramePeriod() time.Duration
	Subscribe(ctx context.Context) <-chan window.Event
	Paint(ctx context.Context, tensor *ndarray.Tensor[uint8]) error
}

type factory struct{}

func (factory) ID() string   { return ID }
func (factory) Name() string { return "Vulkan window" }
func (factory) Weight() int  { return 50 }

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

func (factory) New(context.Context) (Driver, error) { return opener{}, nil }

type opener struct{}

// Open opens a swapchain screen.
func Open(ctx context.Context, cfg window.Config) (Screen, error) {
	return opener{}.Open(ctx, cfg)
}

func (opener) Open(ctx context.Context, cfg window.Config) (Screen, error) {
	w, h, err := cfg.Size()
	if err != nil {
		return nil, err
	}
	title := cfg.Title
	if title == "" {
		title = "lewkit"
	}
	swap, err := vulkan.OpenScreen(ctx, w, h, title)
	if err != nil {
		return nil, err
	}
	return &screen{
		swap:      swap,
		evaluator: ndeval.Bind(swap.Device()),
		size:      image.Pt(w, h),
		period:    window.DefaultFramePeriod,
		bus:       event.New[window.Event](),
	}, nil
}

type screen struct {
	swap      vulkan.Screen
	evaluator ndarray.Evaluator
	size      image.Point
	period    time.Duration
	bus       *event.Bus[window.Event]
}

func (s *screen) Size() image.Point {
	if s == nil {
		return image.Point{}
	}
	return s.size
}

func (s *screen) FramePeriod() time.Duration {
	if s == nil || s.period <= 0 {
		return window.DefaultFramePeriod
	}
	return s.period
}

func (s *screen) Subscribe(ctx context.Context) <-chan window.Event {
	if s == nil || s.bus == nil {
		ch := make(chan window.Event)
		close(ch)
		return ch
	}
	return s.bus.Subscribe(ctx)
}

func (s *screen) Paint(ctx context.Context, tensor *ndarray.Tensor[uint8]) error {
	if s == nil || s.swap == nil || s.evaluator == nil {
		return window.ErrClosed
	}
	return ndeval.Paint(ctx, s.evaluator, tensor, s.swap)
}

func (s *screen) Close() error {
	if s == nil {
		return nil
	}
	var err error
	if s.evaluator != nil {
		err = s.evaluator.Close()
		s.evaluator = nil
	}
	if s.swap != nil {
		err = errors.Join(err, s.swap.Close())
		s.swap = nil
	}
	return err
}
