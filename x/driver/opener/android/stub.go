//go:build !android

package android

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/opener"
)

func (factory) CheckCompatibility(context.Context) error {
	return driver.JavaVM()
}

func (factory) New(context.Context) (opener.Driver, error) {
	return nil, driver.JavaVM()
}
