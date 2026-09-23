package wkwebview

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/webview"
)

type factory struct{}

func (factory) ID() string   { return "webview_wkwebview" }
func (factory) Name() string { return "WKWebView" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("%w: not darwin", driver.ErrIncompatible)
	}
	return frameworks()
}

func (factory) New(context.Context) (webview.Driver, error) {
	return webKitDriver{}, nil
}

type webKitDriver struct{}
