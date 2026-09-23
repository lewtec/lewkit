package vulkanwindow

import (
	"context"
	"errors"
	"fmt"
	"image"
	"runtime"
	"sync"
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
	out := &screen{
		swap:      swap,
		evaluator: ndeval.Bind(swap.Device()),
		size:      image.Pt(w, h),
		period:    window.DefaultFramePeriod,
		bus:       event.New[window.Event](),
	}
	swap.OnInput(out.publish)
	return out, nil
}

type screen struct {
	mu        sync.Mutex
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
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.size
}

func (s *screen) publish(in vulkan.Input) {
	if s == nil || s.bus == nil {
		return
	}
	switch in.Kind {
	case vulkan.InputResize:
		size := image.Pt(in.X, in.Y)
		s.mu.Lock()
		s.size = size
		s.mu.Unlock()
		s.bus.Publish(window.Resize{Size: size})
	case vulkan.InputClose:
		s.bus.Publish(window.Close{})
	case vulkan.InputExpose:
		s.bus.Publish(window.Expose{})
	case vulkan.InputPointer:
		s.bus.Publish(window.Pointer{Pos: image.Pt(in.X, in.Y), Button: in.Button, Pressed: in.Pressed, Buttons: in.Buttons})
	case vulkan.InputScroll:
		s.bus.Publish(window.Scroll{Pos: image.Pt(in.X, in.Y), Delta: image.Pt(in.DX, in.DY)})
	case vulkan.InputKey:
		s.bus.Publish(window.Key{Rune: in.Rune, Code: in.Code, Pressed: in.Pressed, Repeat: in.Repeat, Mod: window.Modifier(in.Mod)})
	}
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
