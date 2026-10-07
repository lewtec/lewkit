package android

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/webview"
)

type factory struct{}

type pageDriver struct{}

func (factory) ID() string   { return "webview_android" }
func (factory) Name() string { return "Android WebView" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	if runtime.GOOS != "android" {
		return fmt.Errorf("%w: not android", driver.ErrIncompatible)
	}
	return vmReady()
}

func (factory) New(context.Context) (webview.Driver, error) { return pageDriver{}, nil }

var (
	_ driver.DriverFactory[webview.Driver] = factory{}
	_ webview.Driver                       = pageDriver{}
	_ webview.View                         = (*pageView)(nil)
)
