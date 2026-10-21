//go:build !linux || android

package dbus

import (
	"context"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/notification"
)

func available(context.Context) error {
	return driver.ForGOOS(build.GOOSLinux)
}

func open(context.Context) (notification.Driver, error) {
	return nil, driver.ForGOOS(build.GOOSLinux)
}
