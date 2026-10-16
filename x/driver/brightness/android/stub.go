//go:build !android

package android

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/brightness"
)

func init() { driver.Register[brightness.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "brightness_android" }
func (factory) Name() string { return "Android brightness" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	return driver.JavaVM()
}

func (factory) New(context.Context) (brightness.Driver, error) {
	return nil, driver.JavaVM()
}
