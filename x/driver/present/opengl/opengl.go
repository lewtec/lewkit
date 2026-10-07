// Package opengl is the present screen backed by OpenGL or OpenGL ES.
// Weight 15 puts it after Vulkan. Apple stays on Metal, so this factory is
// incompatible on darwin and ios. The screen paints the fill list. It does
// not implement present.Painter.
package opengl

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/present"
	ffiopengl "github.com/lewtec/lewkit/x/ffi/native/opengl"
)

type factory struct{}

func (factory) ID() string   { return "present_opengl" }
func (factory) Name() string { return "OpenGL" }
func (factory) Weight() int  { return 15 }

func (factory) CheckCompatibility(context.Context) error {
	if runtime.GOOS == "darwin" || runtime.GOOS == "ios" {
		return fmt.Errorf("%w: apple uses metal", driver.ErrIncompatible)
	}
	if err := ffiopengl.Available(); err != nil {
		return fmt.Errorf("%w: %v", driver.ErrIncompatible, err)
	}
	return nil
}

func (factory) New(context.Context) (present.Driver, error) { return opener{}, nil }

type opener struct{}

func (opener) Open(_ context.Context, kind int, a, b uintptr, width, height int) (present.Screen, error) {
	binding, err := ffiopengl.OpenNative(kind, a, b, width, height)
	if err != nil {
		if errors.Is(err, ffiopengl.ErrUnavailable) {
			return nil, fmt.Errorf("%w: %v", driver.ErrIncompatible, err)
		}
		return nil, mapErr(err)
	}
	return &screen{binding: binding}, nil
}

type screen struct {
	binding *ffiopengl.Screen
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
	case errors.Is(err, ffiopengl.ErrLost):
		return present.ErrLost
	case errors.Is(err, ffiopengl.ErrSize):
		return present.ErrSize
	case errors.Is(err, ffiopengl.ErrClosed):
		return present.ErrClosed
	default:
		return err
	}
}
