//go:build darwin && !ios

package wkwebview

import (
	"context"

	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/webview"
)

// The shared darwin file names these so both hosts typecheck.
// runtime.GOOS keeps them off macOS.

func pumpIOS() {}

func (*webKitView) createIOS(context.Context, webview.Config) error { return nil }

func (*webKitView) useSchemeIOS(daynight.Mode, bool) {}

func (*webKitView) closeIOS() error { return nil }
