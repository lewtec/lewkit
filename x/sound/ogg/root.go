// Package ogg registers an Ogg Vorbis decoder.
package ogg

import (
	"encoding/binary"
	"io"
	"math"

	"github.com/jfreymuth/oggvorbis"
	"github.com/lewtec/lewkit/x/sound"
)

func init() { sound.MustRegister(decoder{}) }

type decoder struct{}

func (decoder) Name() string { return "ogg" }

func (decoder) Extensions() []string { return []string{".ogg"} }

func (decoder) Magic() [][]byte { return [][]byte{[]byte("OggS")} }

func (decoder) Decode(r io.Reader) (*sound.Pipeline, error) {
	reader, err := oggvorbis.NewReader(r)
	if err != nil {
		return nil, err
	}
	format := sound.Format{
		Rate:     reader.SampleRate(),
		Channels: reader.Channels(),
		Sample:   sound.SampleF32LE,
	}
	frames := reader.Length()
	if frames == 0 {
		frames = -1
	}
	return sound.FromSeeker(format, frames, &floatPCM{reader: reader, channels: format.Channels})
}

// floatPCM is interleaved float32 PCM as little-endian bytes.
type floatPCM struct {
	reader   *oggvorbis.Reader
	channels int
	pending  []byte
}

func (f *floatPCM) Read(p []byte) (int, error) {
	if len(f.pending) == 0 {
		samples := make([]float32, 1024*f.channels)
		n, err := f.reader.Read(samples)
		if n > 0 {
			f.pending = encode(samples[:n])
		}
		if err != nil && n == 0 {
			return 0, err
		}
	}
	n := copy(p, f.pending)
	f.pending = f.pending[n:]
	return n, nil
}

func (f *floatPCM) Seek(offset int64, whence int) (int64, error) {
	if whence != io.SeekStart {
		return 0, sound.ErrSeek
	}
	frame := offset / int64(f.channels*4)
	if err := f.reader.SetPosition(frame); err != nil {
		return 0, err
	}
	f.pending = nil
	return frame * int64(f.channels*4), nil
}

func encode(samples []float32) []byte {
	out := make([]byte, len(samples)*4)
	for i, sample := range samples {
		binary.LittleEndian.PutUint32(out[i*4:], math.Float32bits(sample))
	}
	return out
}
