// Package sound is interleaved PCM: format, mixing, WAV bytes, and a seekable pipeline.
//
// [Mixer.Add] is an [io.WriteCloser]. [Mixer.Read] sums queued frames and
// uses silence for a track that has no frame yet. [Pipeline] pulls frames
// and seeks by frame. [Decode] selects a registered decoder by file name or
// magic. Copy either reader into a playback writer from
// [github.com/lewtec/lewkit/x/driver/sound]. A trailing partial frame is
// dropped when its writer closes.
package sound

import "errors"

var (
	// ErrFormat means the rate, channel count, or sample encoding is unusable.
	ErrFormat = errors.New("invalid sound format")
	// ErrFrame means a buffer is not a whole number of PCM frames.
	ErrFrame = errors.New("pcm length is not a whole number of frames")
	// ErrClosed means a write happened after close.
	ErrClosed = errors.New("sound closed")
)

// Sample is the encoding of one interleaved channel.
type Sample uint8

const (
	// SampleS16LE is signed 16-bit little-endian PCM.
	SampleS16LE Sample = iota + 1
	// SampleF32LE is 32-bit little-endian float PCM in the range -1 to 1.
	SampleF32LE
)

// Format is interleaved PCM. Rate is samples per second. Channels is the
// interleave width. A frame is one sample for every channel.
type Format struct {
	Rate     int
	Channels int
	Sample   Sample
}

// Width is the number of bytes in one sample.
func (f Format) Width() int {
	switch f.Sample {
	case SampleS16LE:
		return 2
	case SampleF32LE:
		return 4
	default:
		return 0
	}
}

// Validate reports a usable playback format.
func (f Format) Validate() error {
	if f.Rate <= 0 || f.Rate > 384000 {
		return ErrFormat
	}
	if f.Channels <= 0 || f.Channels > 32 {
		return ErrFormat
	}
	if f.Width() == 0 {
		return ErrFormat
	}
	return nil
}

// Frame is the number of bytes in one interleaved frame.
func (f Format) Frame() (int, error) {
	if err := f.Validate(); err != nil {
		return 0, err
	}
	return f.Channels * f.Width(), nil
}
