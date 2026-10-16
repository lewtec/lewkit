//go:build !windows

package win32

import (
	"context"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
)

func (wdriver) Open(context.Context, window.Config) (window.Window, error) {
	return nil, driver.ForGOOS(build.GOOSWindows)
}
