// Package android asks the foreground activity for a yes or no answer.
package android

import (
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/launcher"
)

var _ driver.DriverFactory[launcher.Confirmer] = factory{}
var _ driver.Weighter = factory{}
