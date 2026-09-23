package webview2

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/webview"
)

type factory struct{}

func (factory) ID() string   { return "webview_webview2" }
func (factory) Name() string { return "WebView2" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("%w: not windows", driver.ErrIncompatible)
	}
	return loader()
}

func (factory) New(context.Context) (webview.Driver, error) {
	return edgeDriver{}, nil
}

type edgeDriver struct{}
