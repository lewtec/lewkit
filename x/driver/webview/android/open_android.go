//go:build android

package android

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/webview"
	"github.com/lewtec/lewkit/x/ffi/jni"
)

func (pageDriver) Open(ctx context.Context, cfg webview.Config) (webview.View, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	view := &pageView{
		id:       int(pageSeq.Add(1)),
		handler:  cfg.Handler,
		html:     cfg.HTML,
		files:    cfg.FS,
		messages: make(chan []byte, 32),
		done:     make(chan struct{}),
	}
	proxy, err := jni.Proxy("lewkit.Page$Bridge", view.invoke)
	if err != nil {
		return nil, err
	}
	view.proxy = proxy
	ok, err := jni.Bool(jni.CallStatic("lewkit.Page", "open", view.id, proxy))
	if err != nil || !ok {
		proxy.Release()
		view.proxy = nil
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%w: android page", driver.ErrUnavailable)
	}
	webview.CloseWhenDone(ctx, view)
	webview.Follow(ctx, func(mode daynight.Mode) {
		_, _ = jni.CallStatic("lewkit.Page", "preferDark", view.id, mode == daynight.Dark)
	})
	return view, nil
}
