// Package pulse binds libpulse and libpulse-simple without cgo.
package pulse

import (
	"errors"
	"unsafe"
)

var (
	errUnavailable = errors.New("pulse unavailable")
	errClosed      = errors.New("pulse stream closed")
	errMainloop    = errors.New("pulse mainloop")
	errContext     = errors.New("pulse context")
	errSinkList    = errors.New("pulse sink list")
	errTimeout     = errors.New("pulse timeout")
)

// Sample matches pa_sample_format_t values used for playback.
type Sample int32

const (
	// SampleS16LE is PA_SAMPLE_S16LE.
	SampleS16LE Sample = 3
	// SampleF32LE is PA_SAMPLE_FLOAT32LE.
	SampleF32LE Sample = 5
)

// Sink is one PulseAudio or PipeWire playback device.
type Sink struct {
	Name        string
	Description string
}

func ptr(b []byte) uintptr {
	if len(b) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(&b[0]))
}
