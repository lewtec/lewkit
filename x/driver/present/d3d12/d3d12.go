package d3d12

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/internal/gpuhost"
	"github.com/lewtec/lewkit/x/driver/present"
	"github.com/lewtec/lewkit/x/driver/present/internal/nativeerr"
	"github.com/lewtec/lewkit/x/driver/window"
	ffid3d12 "github.com/lewtec/lewkit/x/ffi/native/d3d12"
)

type factory struct{}

func (factory) ID() string   { return "present_d3d12" }
func (factory) Name() string { return "Direct3D 12" }
func (factory) Weight() int  { return 70 }

func (factory) CheckCompatibility(context.Context) error { return gpuhost.Direct3D() }

func (factory) New(context.Context) (present.Driver, error) { return opener{}, nil }

type opener struct{}

func (opener) Open(_ context.Context, kind int, a, _ uintptr, width, height int) (present.Screen, error) {
	if kind != window.SurfaceWin32 || a == 0 {
		return nil, fmt.Errorf("%w: surface %d", driver.ErrIncompatible, kind)
	}
	binding, err := ffid3d12.OpenNative(kind, a, width, height)
	if err != nil {
		return nil, nativeerr.Map(err, ffid3d12.ErrLost, ffid3d12.ErrSize, ffid3d12.ErrClosed)
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
	return nativeerr.Map(s.binding.Draw(instances, under, ink, width, height), ffid3d12.ErrLost, ffid3d12.ErrSize, ffid3d12.ErrClosed)
}

func (s *screen) Adopt(window uintptr, width, height int) error {
	if s == nil || s.binding == nil {
		return present.ErrClosed
	}
	return nativeerr.Map(s.binding.Adopt(window, width, height), ffid3d12.ErrLost, ffid3d12.ErrSize, ffid3d12.ErrClosed)
}

func (s *screen) Close() error {
	if s == nil || s.binding == nil {
		return nil
	}
	err := s.binding.Close()
	s.binding = nil
	return nativeerr.Map(err, ffid3d12.ErrLost, ffid3d12.ErrSize, ffid3d12.ErrClosed)
}
