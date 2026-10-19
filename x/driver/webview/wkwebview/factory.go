package wkwebview

import (
	"context"
	"runtime"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/webview"
)

type factory struct{}

func (factory) ID() string   { return "webview_wkwebview" }
func (factory) Name() string { return "WKWebView" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error {
	if runtime.GOOS != "darwin" && runtime.GOOS != "ios" {
		return driver.ForGOOS(build.GOOSDarwin)
	}
	// Opening UIKit here runs on the caller. The UI thread loads it.
	if runtime.GOOS == "ios" {
		return nil
	}
	return frameworks()
}

func (factory) New(context.Context) (webview.Driver, error) {
	return webKitDriver{}, nil
}

type webKitDriver struct{}
