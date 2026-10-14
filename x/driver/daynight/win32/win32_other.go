//go:build !windows

package win32

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/daynight"
)

func available(context.Context) error {
	return driver.ForGOOS("windows")
}

func open(context.Context) (daynight.Driver, error) {
	return nil, driver.ForGOOS("windows")
}
