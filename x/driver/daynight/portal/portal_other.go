//go:build !linux || android

package portal

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/daynight"
)

func available(context.Context) error {
	return driver.ForGOOS("linux")
}

func open(context.Context) (daynight.Driver, error) {
	return nil, driver.ForGOOS("linux")
}
