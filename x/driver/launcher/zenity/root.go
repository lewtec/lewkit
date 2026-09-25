package zenity

import (
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/launcher"
)

var _ driver.DriverFactory[launcher.Prompter] = prompterFactory{}
var _ driver.DriverFactory[launcher.Confirmer] = confirmerFactory{}
var _ driver.Weighter = prompterFactory{}
var _ driver.Weighter = confirmerFactory{}

func init() {
	driver.Register[launcher.Prompter](prompterFactory{})
	driver.Register[launcher.Confirmer](confirmerFactory{})
}
