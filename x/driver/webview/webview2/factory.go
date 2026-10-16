package webview2

import (
	"context"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/webview"
)

type factory struct{}

func (factory) ID() string   { return "webview_webview2" }
func (factory) Name() string { return "WebView2" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error {
	if err := driver.ForGOOS(build.GOOSWindows); err != nil {
		return err
	}
	return loader()
}

func (factory) New(context.Context) (webview.Driver, error) {
	return edgeDriver{}, nil
}

type edgeDriver struct{}
