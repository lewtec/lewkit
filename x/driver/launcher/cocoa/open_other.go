//go:build !darwin

package cocoa

import (
	"context"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/launcher"
)

func (backend) Choose(context.Context, launcher.ChooseOptions) (*launcher.Item, error) {
	return nil, driver.ForGOOS(build.GOOSDarwin)
}

func (backend) Prompt(context.Context, string) (string, error) {
	return "", driver.ForGOOS(build.GOOSDarwin)
}

func (backend) Confirm(context.Context, string) (bool, error) {
	return false, driver.ForGOOS(build.GOOSDarwin)
}
