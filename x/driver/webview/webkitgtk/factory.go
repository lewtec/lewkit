package webkitgtk

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/webview"
)

type factory struct{}

func (factory) ID() string   { return "webview_webkitgtk" }
func (factory) Name() string { return "WebKitGTK" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("%w: not linux", driver.ErrIncompatible)
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
