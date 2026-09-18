package win32

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
)

type factory struct{}

func (factory) ID() string   { return "window_win32" }
func (factory) Name() string { return "Win32" }

func (factory) CheckCompatibility(context.Context) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("%w: not windows", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (window.Driver, error) {
	return wdriver{}, nil
}

type wdriver struct{}
