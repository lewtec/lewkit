//go:build !android

package jni

import (
	"context"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/driver"
	uithread "github.com/lewtec/lewkit/x/driver/thread"
)

func init() { driver.Register[uithread.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "thread_jni" }
func (factory) Name() string { return "Android looper" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	return driver.ForGOOS(build.GOOSAndroid)
}

func (factory) New(context.Context) (uithread.Driver, error) {
	return nil, driver.ForGOOS(build.GOOSAndroid)
}
