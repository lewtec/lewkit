// Package android opens the activity WebView and serves the page in this process.
// Each request goes through [webview.Dispatch]. Nothing listens, and the
// packaged service is not started.
package android

import (
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/webview"
)

func init() { driver.Register[webview.Driver](factory{}) }
