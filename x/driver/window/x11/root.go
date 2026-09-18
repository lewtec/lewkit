package x11

import (
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
)

var _ driver.DriverFactory[window.Driver] = factory{}

func init() {
	driver.Register[window.Driver](&factory{})
}
