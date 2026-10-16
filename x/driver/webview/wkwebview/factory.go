package wkwebview

import (
	"context"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/webview"
)

type factory struct{}

func (factory) ID() string   { return "webview_wkwebview" }
func (factory) Name() string { return "WKWebView" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error {
	if err := driver.ForGOOS(build.GOOSDarwin); err != nil {
		return err
	}
	return frameworks()
}

func (factory) New(context.Context) (webview.Driver, error) {
	return webKitDriver{}, nil
}

type webKitDriver struct{}
