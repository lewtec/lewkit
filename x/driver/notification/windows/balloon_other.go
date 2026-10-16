//go:build !windows

package windows

import (
	"context"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/notification"
)

func available(context.Context) error {
	return driver.ForGOOS(build.GOOSWindows)
}

func (backend) Notify(context.Context, notification.Notification) error {
	return driver.ForGOOS(build.GOOSWindows)
}
