package winmm

import (
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/sound"
)

var _ driver.DriverFactory[sound.Driver] = factory{}
var _ driver.Weighter = factory{}

func init() {
	driver.Register[sound.Driver](&factory{})
}
