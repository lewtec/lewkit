package coreaudio

import (
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/audio_play"
)

var _ driver.DriverFactory[audio_play.Driver] = factory{}
var _ driver.Weighter = factory{}

func init() {
	driver.Register[audio_play.Driver](&factory{})
}
