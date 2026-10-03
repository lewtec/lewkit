package metal

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/present"
	"github.com/lewtec/lewkit/x/driver/thread"
	_ "github.com/lewtec/lewkit/x/driver/thread/std"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ffi/native/dispatch"
	ffimetal "github.com/lewtec/lewkit/x/ffi/native/metal"
)

func init() {
	ffimetal.SetUI(func(fn func()) {
		if runtime.GOOS == "ios" {
			dispatch.OnMain(fn)
			return
		}
		thread.Do(fn)
	})
}

type factory struct{}

func (factory) ID() string   { return "present_metal" }
func (factory) Name() string { return "Metal" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	if runtime.GOOS != "darwin" && runtime.GOOS != "ios" {
		return fmt.Errorf("%w: not apple", driver.ErrIncompatible)
	}
	if err := ffimetal.Available(); err != nil {
		return fmt.Errorf("%w: %v", driver.ErrIncompatible, err)
	}
	return nil
}

func (factory) New(context.Context) (present.Driver, error) { return opener{}, nil }

type opener struct{}

func (opener) Open(_ context.Context, kind int, a, _ uintptr, width, height int) (present.Screen, error) {
	if kind != window.SurfaceView && kind != window.SurfaceUIView {
		return nil, fmt.Errorf("%w: surface %d", driver.ErrIncompatible, kind)
	}
	binding, err := ffimetal.OpenNative(kind, a, width, height)
	if err != nil {
		return nil, err
	}
	return &screen{binding: binding}, nil
}

type screen struct {
	binding *ffimetal.Screen
}

func (s *screen) Draw(_ context.Context, instances, under, ink []byte, width, height int) error {
	if s == nil || s.binding == nil {
		return present.ErrClosed
	}
	return mapErr(s.binding.Draw(instances, under, ink, width, height))
}

func (s *screen) Adopt(window uintptr, width, height int) error {
	if s == nil || s.binding == nil {
		return present.ErrClosed
	}
	return mapErr(s.binding.Adopt(window, width, height))
}

func (s *screen) Close() error {
	if s == nil || s.binding == nil {
		return nil
	}
	err := s.binding.Close()
	s.binding = nil
	return mapErr(err)
}

func mapErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ffimetal.ErrLost):
		return present.ErrLost
	case errors.Is(err, ffimetal.ErrSize):
		return present.ErrSize
	case errors.Is(err, ffimetal.ErrClosed):
		return present.ErrClosed
	default:
		return err
	}
}
