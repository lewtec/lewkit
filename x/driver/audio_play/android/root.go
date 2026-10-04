// Package android plays interleaved PCM through AudioTrack.
//
// Config.Sink is "default", "Default", or empty. Playback is 16-bit.
// Float frames are converted. One or two channels are accepted.
package android

import (
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/audio_play"
)

var _ driver.DriverFactory[audio_play.Driver] = factory{}
var _ driver.Weighter = factory{}

func init() { driver.Register[audio_play.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "audio_play_android" }
func (factory) Name() string { return "Android AudioTrack" }
func (factory) Weight() int  { return 80 }
