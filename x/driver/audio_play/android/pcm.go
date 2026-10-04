package android

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/audio_play"
	pcs "github.com/lewtec/lewkit/x/sound"
)

// Android audio constants. Names match the Java fields.
const (
	streamMusic      = 3  // AudioManager.STREAM_MUSIC
	channelMono      = 4  // AudioFormat.CHANNEL_OUT_MONO
	channelStereo    = 12 // AudioFormat.CHANNEL_OUT_STEREO
	encodingPCM16    = 2  // AudioFormat.ENCODING_PCM_16BIT
	modeStream       = 1  // AudioTrack.MODE_STREAM
	stateInit        = 1  // AudioTrack.STATE_INITIALIZED
	writeNonBlocking = 1  // AudioTrack.WRITE_NON_BLOCKING
	sinkID           = "default"
)

func sinks() []audio_play.Sink {
	return []audio_play.Sink{{ID: sinkID, Name: "Default"}}
}

func channelMask(channels int) (int, error) {
	switch channels {
	case 1:
		return channelMono, nil
	case 2:
		return channelStereo, nil
	default:
		return 0, fmt.Errorf("%w: %d channels", pcs.ErrFormat, channels)
	}
}

// alignBuffer rounds min up to a whole frame. min is getMinBufferSize.
func alignBuffer(min, frame int) (int, error) {
	if min <= 0 || frame <= 0 {
		return 0, fmt.Errorf("%w: buffer %d", driver.ErrUnavailable, min)
	}
	if rem := min % frame; rem != 0 {
		min += frame - rem
	}
	return min, nil
}

// pcm16 returns interleaved s16le. p is returned as-is for s16le.
func pcm16(sample pcs.Sample, p []byte) ([]byte, error) {
	switch sample {
	case pcs.SampleS16LE:
		return p, nil
	case pcs.SampleF32LE:
		return floatToS16(p)
	default:
		return nil, pcs.ErrFormat
	}
}

func floatToS16(p []byte) ([]byte, error) {
	if len(p)%4 != 0 {
		return nil, pcs.ErrFrame
	}
	out := make([]byte, len(p)/2)
	for i := 0; i < len(p); i += 4 {
		f := math.Float32frombits(binary.LittleEndian.Uint32(p[i : i+4]))
		binary.LittleEndian.PutUint16(out[i/2:], uint16(int16(clampS16(f))))
	}
	return out, nil
}

func clampS16(f float32) int32 {
	if f > 1 {
		f = 1
	} else if f < -1 {
		f = -1
	}
	v := int32(math.Round(float64(f) * 32767))
	if v > 32767 {
		return 32767
	}
	if v < -32768 {
		return -32768
	}
	return v
}
