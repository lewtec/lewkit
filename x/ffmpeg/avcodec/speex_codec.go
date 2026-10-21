package avcodec

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// spxCodec encodes 8 kHz mono Speex narrowband at quality 8.
// Frames are 160 samples. A trailing partial frame is dropped.
type spxCodec struct {
	pcm []int16
}

func newSpeex(par Parameters) (*spxCodec, error) {
	rate := par.SampleRate
	if rate <= 0 {
		rate = 8000
	}
	if rate != 8000 {
		return nil, fmt.Errorf("%w: speex rate %d", avutil.ErrUnsupported, rate)
	}
	if par.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: speex channels", avutil.ErrInvalidData)
	}
	return &spxCodec{}, nil
}

func (c *spxCodec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil || frame.NbSamples == 0 {
		return nil, nil
	}
	if frame.SampleFmt != avutil.SampleS16 || frame.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: speex", avutil.ErrInvalidData)
	}
	if len(frame.Data) == 0 || len(frame.Data[0]) < frame.NbSamples*2 {
		return nil, fmt.Errorf("%w: speex samples", avutil.ErrInvalidData)
	}
	for i := 0; i < frame.NbSamples; i++ {
		c.pcm = append(c.pcm, int16(binary.LittleEndian.Uint16(frame.Data[0][i*2:])))
	}
	return nil, nil
}

func (c *spxCodec) Flush() ([]*avutil.Packet, error) {
	n := len(c.pcm)
	raw, err := SpeexEncodeNB(c.pcm)
	c.pcm = nil
	if err != nil || len(raw) == 0 {
		return nil, err
	}
	return []*avutil.Packet{{Data: raw, Duration: int64(n)}}, nil
}

// SpeexExtradata is the 80-byte Speex header libspeex writes for
// narrowband quality 8: 8 kHz, mode 0, one channel, 15000 bit/s, 160-sample frames.
func SpeexExtradata() []byte {
	b := make([]byte, 80)
	copy(b[0:8], "Speex   ")
	copy(b[8:28], "1.2.1")
	binary.LittleEndian.PutUint32(b[28:], 1)
	binary.LittleEndian.PutUint32(b[32:], 80)
	binary.LittleEndian.PutUint32(b[36:], 8000)
	binary.LittleEndian.PutUint32(b[44:], 4)
	binary.LittleEndian.PutUint32(b[48:], 1)
	binary.LittleEndian.PutUint32(b[52:], 15000)
	binary.LittleEndian.PutUint32(b[56:], 160)
	binary.LittleEndian.PutUint32(b[64:], 1)
	return b
}
