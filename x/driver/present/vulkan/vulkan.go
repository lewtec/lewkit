package vulkan

import (
	"context"
	"errors"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	ndvulkan "github.com/lewtec/lewkit/x/driver/ndeval/vulkan"
	"github.com/lewtec/lewkit/x/driver/present"
	drvvulkan "github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
)

type factory struct{}

func (factory) ID() string   { return "present_vulkan" }
func (factory) Name() string { return "Vulkan" }
func (factory) Weight() int  { return 40 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if _, err := drvvulkan.List(ctx); err != nil {
		return fmt.Errorf("%w: %v", driver.ErrIncompatible, err)
	}
	return nil
}

func (factory) New(context.Context) (present.Driver, error) { return opener{}, nil }

type opener struct{}

func (opener) Open(ctx context.Context, kind int, a, b uintptr, width, height int) (present.Screen, error) {
	screen, err := drvvulkan.OpenNative(ctx, kind, a, b, width, height)
	if err != nil {
		return nil, err
	}
	return &gpuScreen{screen: screen, eval: ndvulkan.Bind(screen.Device())}, nil
}

type gpuScreen struct {
	screen drvvulkan.Screen
	eval   ndarray.Evaluator
}

func (s *gpuScreen) Draw(ctx context.Context, instances, under, ink []byte, width, height int) error {
	if s == nil || s.screen == nil {
		return present.ErrClosed
	}
	return mapErr(s.screen.Draw(ctx, instances, under, ink, width, height))
}

func (s *gpuScreen) Adopt(window uintptr, width, height int) error {
	if s == nil || s.screen == nil {
		return present.ErrClosed
	}
	return mapErr(s.screen.Adopt(window, width, height))
}

func (s *gpuScreen) Close() error {
	if s == nil || s.screen == nil {
		return nil
	}
	err := s.screen.Close()
	s.screen = nil
	s.eval = nil
	return err
}

func (s *gpuScreen) Paint(ctx context.Context, pixels *ndarray.Tensor[uint8]) error {
	if s == nil || s.screen == nil || s.eval == nil {
		return present.ErrClosed
	}
	return mapErr(ndvulkan.Paint(ctx, s.eval, pixels, s.screen))
}

func (s *gpuScreen) Evaluator() ndarray.Evaluator {
	if s == nil {
		return nil
	}
	return s.eval
}

func mapErr(err error) error {
	if errors.Is(err, drvvulkan.ErrLost) {
		return present.ErrLost
	}
	return err
}
