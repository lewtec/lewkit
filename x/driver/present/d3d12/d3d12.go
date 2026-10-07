package d3d12

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/present"
	"github.com/lewtec/lewkit/x/driver/window"
	ffid3d12 "github.com/lewtec/lewkit/x/ffi/native/d3d12"
)

type factory struct{}

func (factory) ID() string   { return "present_d3d12" }
func (factory) Name() string { return "Direct3D 12" }
func (factory) Weight() int  { return 70 }

func (factory) CheckCompatibility(context.Context) error {
	if runtime.GOOS != "windows" || (runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64") {
		return fmt.Errorf("%w: not windows", driver.ErrIncompatible)
	}
	if err := ffid3d12.Available(); err != nil {
		return fmt.Errorf("%w: %v", driver.ErrIncompatible, err)
	}
	return nil
}

func (factory) New(context.Context) (present.Driver, error) { return opener{}, nil }

type opener struct{}

func (opener) Open(_ context.Context, kind int, a, _ uintptr, width, height int) (present.Screen, error) {
	if kind != window.SurfaceWin32 || a == 0 {
		return nil, fmt.Errorf("%w: surface %d", driver.ErrIncompatible, kind)
	}
	binding, err := ffid3d12.OpenNative(kind, a, width, height)
	if err != nil {
		return nil, mapErr(err)
	}
	return &screen{binding: binding}, nil
}

type screen struct {
	binding *ffid3d12.Screen
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
	case errors.Is(err, ffid3d12.ErrLost):
		return present.ErrLost
	case errors.Is(err, ffid3d12.ErrSize):
		return present.ErrSize
	case errors.Is(err, ffid3d12.ErrClosed):
		return present.ErrClosed
	default:
		return err
	}
}
