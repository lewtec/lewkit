package webview2

import (
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/webview"
)

var _ driver.DriverFactory[webview.Driver] = factory{}

func init() {
	driver.Register[webview.Driver](factory{})
}
