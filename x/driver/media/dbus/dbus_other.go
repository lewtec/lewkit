//go:build !linux || android

package dbus

import (
	"context"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/media"
)

func available(context.Context) error {
	return driver.ForGOOS(build.GOOSLinux)
}

func open(context.Context) (media.Driver, error) {
	return nil, driver.ForGOOS(build.GOOSLinux)
}
