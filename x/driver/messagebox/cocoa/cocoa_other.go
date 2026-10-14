//go:build !darwin

package cocoa

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/messagebox"
)

func available(context.Context) error {
	return driver.ForGOOS("darwin")
}

func open(context.Context) (messagebox.Driver, error) {
	return nil, driver.ForGOOS("darwin")
}
