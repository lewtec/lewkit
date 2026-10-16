//go:build !linux

package kms

import (
	"context"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
)

func (factory) CheckCompatibility(context.Context) error {
	return driver.ForGOOS(build.GOOSLinux)
}

func (opener) Open(context.Context, window.Config) (window.Window, error) {
	return nil, driver.ForGOOS(build.GOOSLinux)
}
