// Package android reads one line of text from the activity dialog.
package android

import (
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/launcher"
)

var _ driver.DriverFactory[launcher.Prompter] = factory{}
var _ driver.Weighter = factory{}
