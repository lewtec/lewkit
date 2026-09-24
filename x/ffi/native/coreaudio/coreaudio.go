// Package coreaudio binds AudioQueue playback and CoreAudio device names without cgo.
package coreaudio

import "errors"

// Sample selects signed 16-bit or float PCM.
type Sample uint8

const (
	// SampleS16LE is packed little-endian signed 16-bit PCM.
	SampleS16LE Sample = iota + 1
	// SampleF32LE is packed little-endian float PCM.
	SampleF32LE
)

// Layout is the PCM written to a device.
type Layout struct {
	Sample   Sample
	Rate     int
	Channels int
}

// Device is one output device. ID is the CoreAudio UID.
type Device struct {
	ID   string
	Name string
}

var errUnavailable = errors.New("coreaudio unavailable")

func fourcc(s string) uint32 {
	return uint32(s[0])<<24 | uint32(s[1])<<16 | uint32(s[2])<<8 | uint32(s[3])
}

// streamDesc matches AudioStreamBasicDescription.
type streamDesc struct {
	rate            float64
	formatID        uint32
	flags           uint32
	bytesPerPacket  uint32
	framesPerPacket uint32
	bytesPerFrame   uint32
	channels        uint32
	bits            uint32
	reserved        uint32
}

// propAddr matches AudioObjectPropertyAddress.
type propAddr struct {
	selector uint32
	scope    uint32
	element  uint32
}
