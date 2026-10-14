package webkitgtk

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/webview"
)

type factory struct{}

func (factory) ID() string   { return "webview_webkitgtk" }
func (factory) Name() string { return "WebKitGTK" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if err := driver.ForGOOS("linux"); err != nil {
		return err
	}
	if err := driver.RequireAnyEnv(ctx, "DISPLAY", "WAYLAND_DISPLAY"); err != nil {
		return err
	}
	return libraries(ctx)
}

func (factory) New(context.Context) (webview.Driver, error) {
	return gtkDriver{}, nil
}

type gtkDriver struct{}
