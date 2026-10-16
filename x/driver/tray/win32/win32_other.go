//go:build !windows

package win32

import (
	"context"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/tray"
)

type opener struct{}

func (opener) Open(context.Context, tray.Config) (tray.Tray, error) {
	return nil, driver.ForGOOS(build.GOOSWindows)
}
