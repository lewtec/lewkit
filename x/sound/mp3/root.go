// Package mp3 registers an MP3 decoder.
package mp3

import (
	"io"

	gomp3 "github.com/hajimehoshi/go-mp3"
	"github.com/lewtec/lewkit/x/sound"
)

func init() { sound.MustRegister(decoder{}) }

type decoder struct{}

func (decoder) Name() string { return "mp3" }

func (decoder) Extensions() []string { return []string{".mp3"} }

func (decoder) Magic() [][]byte {
	return [][]byte{
		{'I', 'D', '3'},
		{0xff, 0xfb},
		{0xff, 0xf3},
		{0xff, 0xf2},
	}
}

func (decoder) Decode(r io.Reader) (*sound.Pipeline, error) {
	decoded, err := gomp3.NewDecoder(r)
	if err != nil {
		return nil, err
	}
	format := sound.Format{Rate: decoded.SampleRate(), Channels: 2, Sample: sound.SampleS16LE}
	// Length stays unknown unless the source seeks. go-mp3 then leaves its
	// frame size at zero, and seeking that decoder divides by zero.
	if n := decoded.Length(); n >= 0 {
		return sound.FromSeeker(format, n/4, decoded)
	}
	return sound.FromReader(format, decoded)
}
