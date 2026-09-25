package rofi

import (
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/launcher"
)

var _ driver.DriverFactory[launcher.Chooser] = chooserFactory{}
var _ driver.DriverFactory[launcher.Driver] = driverFactory{}
var _ driver.Weighter = chooserFactory{}
var _ driver.Weighter = driverFactory{}

func init() {
	driver.Register[launcher.Chooser](chooserFactory{})
	driver.Register[launcher.Driver](driverFactory{})
}
