// Package winmm binds winmm waveOut without cgo.
package winmm

import (
	"errors"
	"unicode/utf16"
	"unsafe"
)

// Sample matches WAVEFORMATEX format tags used for playback.
type Sample uint16

const (
	// SampleS16LE is WAVE_FORMAT_PCM.
	SampleS16LE Sample = 1
	// SampleF32LE is WAVE_FORMAT_IEEE_FLOAT.
	SampleF32LE Sample = 3
)

// Layout is the PCM written to a device.
type Layout struct {
	Sample   Sample
	Rate     int
	Channels int
}

// Device is one waveOut endpoint. ID is the decimal device index.
type Device struct {
	ID   string
	Name string
}

var errUnavailable = errors.New("winmm unavailable")

func utf16z(chars []uint16) string {
	n := 0
	for n < len(chars) && chars[n] != 0 {
		n++
	}
	return string(utf16.Decode(chars[:n]))
}

// waveHdr matches WAVEHDR on 64-bit Windows.
type waveHdr struct {
	data     uintptr
	length   uint32
	recorded uint32
	user     uintptr
	flags    uint32
	loops    uint32
	next     uintptr
	reserved uintptr
}

// waveFormat matches WAVEFORMATEX.
type waveFormat struct {
	tag      uint16
	channels uint16
	rate     uint32
	avg      uint32
	block    uint16
	bits     uint16
	extra    uint16
}

// waveCaps matches WAVEOUTCAPSW.
type waveCaps struct {
	mid      uint16
	pid      uint16
	version  uint32
	name     [32]uint16
	formats  uint32
	channels uint16
	reserved uint16
	support  uint32
}

func layoutOK() bool {
	return unsafe.Sizeof(uintptr(0)) == 8
}
