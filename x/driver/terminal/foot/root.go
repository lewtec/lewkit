package foot

import (
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/terminal"
)

var _ driver.DriverFactory[terminal.Driver] = factory{}
var _ driver.Weighter = factory{}

func init() {
	driver.Register[terminal.Driver](factory{})
}
