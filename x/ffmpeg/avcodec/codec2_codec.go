package avcodec

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// c2Stream encodes and decodes codec2 mode 1300: 8 kHz mono, 320 samples, 7 bytes.
type c2Stream struct {
	pcm    []int16
	bits   []byte
	enc    *c2Codec
	dec    *c2Codec
	burned bool
}

func newCodec2(par Parameters) (*c2Stream, error) {
	rate := par.SampleRate
	if rate <= 0 {
		rate = 8000
	}
	if rate != 8000 {
		return nil, fmt.Errorf("%w: codec2 rate %d", avutil.ErrUnsupported, rate)
	}
	if par.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: codec2 channels", avutil.ErrInvalidData)
	}
	return &c2Stream{enc: newC2(), dec: newC2()}, nil
}

func (c *c2Stream) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil || frame.NbSamples == 0 {
		return nil, nil
	}
	if frame.SampleFmt != avutil.SampleS16 || frame.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: codec2", avutil.ErrInvalidData)
	}
	if len(frame.Data) == 0 || len(frame.Data[0]) < frame.NbSamples*2 {
		return nil, fmt.Errorf("%w: codec2 samples", avutil.ErrInvalidData)
	}
	for i := 0; i < frame.NbSamples; i++ {
		c.pcm = append(c.pcm, int16(binary.LittleEndian.Uint16(frame.Data[0][i*2:])))
	}
	return c.emitCodec2(false)
}

func (c *c2Stream) Flush() ([]*avutil.Packet, error) {
	return c.emitCodec2(true)
}

func (c *c2Stream) emitCodec2(flush bool) ([]*avutil.Packet, error) {
	var pkts []*avutil.Packet
	for len(c.pcm) >= c2FrameN || (flush && len(c.pcm) > 0) {
		var block [c2FrameN]int16
		n := copy(block[:], c.pcm)
		var raw [c2FrameB]byte
		c.enc.encode(raw[:], block[:])
		c.pcm = append([]int16(nil), c.pcm[n:]...)
		pkts = append(pkts, &avutil.Packet{Data: append([]byte(nil), raw[:]...), Duration: c2FrameN})
		if n < c2FrameN {
			break
		}
	}
	return pkts, nil
}

// Decode is not registered. One frame matches libcodec2; a later frame
// drifts by about one sample, so Codecs lists encode only.
func (c *c2Stream) Decode(pkt *avutil.Packet) ([]*avutil.Frame, error) {
	if pkt == nil || len(pkt.Data) == 0 {
		return nil, nil
	}
	c.bits = append(c.bits, pkt.Data...)
	if !c.burned && len(c.bits) >= c2FrameB {
		probe := newC2()
		var scratch [c2FrameN]int16
		probe.decode(scratch[:], c.bits[:c2FrameB])
		c.dec.randNext = probe.randNext
		c.burned = true
	}
	var samples []int16
	for len(c.bits) >= c2FrameB {
		var pcm [c2FrameN]int16
		c.dec.decode(pcm[:], c.bits[:c2FrameB])
		c.bits = c.bits[c2FrameB:]
		samples = append(samples, pcm[:]...)
	}
	if len(samples) == 0 {
		return nil, nil
	}
	out, err := avutil.NewAudio(avutil.SampleS16, c2Fs, avutil.Mono(), len(samples))
	if err != nil {
		return nil, err
	}
	for i, sample := range samples {
		binary.LittleEndian.PutUint16(out.Data[0][i*2:], uint16(sample))
	}
	out.PTS = pkt.PTS
	out.Duration = int64(len(samples))
	out.TimeBase = avutil.Q(1, c2Fs)
	return []*avutil.Frame{out}, nil
}
