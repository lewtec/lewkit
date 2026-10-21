package avcodec

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// apmCodec encodes and decodes mono IMA APM ADPCM.
// The nibble coder is the QuickTime IMA compressor. A full frame is
// 2048 samples. An odd trailing sample is dropped.
type apmCodec struct {
	pcm  []int16
	prev int
	step int
	rate int
}

func newIMAAPM(par Parameters) *apmCodec {
	rate := par.SampleRate
	if rate <= 0 {
		rate = 44100
	}
	return &apmCodec{rate: rate}
}

func (c *apmCodec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil || frame.NbSamples == 0 {
		return nil, nil
	}
	if frame.SampleFmt != avutil.SampleS16 || frame.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: ima apm", avutil.ErrInvalidData)
	}
	if len(frame.Data) == 0 || len(frame.Data[0]) < frame.NbSamples*2 {
		return nil, fmt.Errorf("%w: ima apm samples", avutil.ErrInvalidData)
	}
	for i := 0; i < frame.NbSamples; i++ {
		c.pcm = append(c.pcm, int16(binary.LittleEndian.Uint16(frame.Data[0][i*2:])))
	}
	return c.emitAPM(false)
}

func (c *apmCodec) Flush() ([]*avutil.Packet, error) { return c.emitAPM(true) }

func (c *apmCodec) emitAPM(flush bool) ([]*avutil.Packet, error) {
	const frame = 2048
	var pkts []*avutil.Packet
	for len(c.pcm) >= frame || (flush && len(c.pcm) >= 2) {
		n := frame
		if n > len(c.pcm) {
			n = len(c.pcm) - len(c.pcm)%2
		}
		raw, prev, step := ssiPacket(c.pcm[:n], c.prev, c.step)
		c.prev, c.step = prev, step
		c.pcm = append([]int16(nil), c.pcm[n:]...)
		pkts = append(pkts, &avutil.Packet{Data: raw, Duration: int64(n)})
	}
	if flush {
		c.pcm = nil
	}
	return pkts, nil
}

func (c *apmCodec) Decode(pkt *avutil.Packet) ([]*avutil.Frame, error) {
	if pkt == nil || len(pkt.Data) == 0 {
		return nil, nil
	}
	samples := ssiExpand(pkt.Data, &c.prev, &c.step)
	frame, err := pcmS16Mono(samples, c.rate, pkt.PTS)
	if err != nil || frame == nil {
		return nil, err
	}
	return []*avutil.Frame{frame}, nil
}

// IMAAPMEncode packs mono PCM into IMA APM ADPCM.
func IMAAPMEncode(pcm []int16) []byte {
	const frame = 2048
	var out []byte
	prev, step := 0, 0
	for len(pcm) >= 2 {
		n := frame
		if n > len(pcm) {
			n = len(pcm) - len(pcm)%2
		}
		raw, p, s := ssiPacket(pcm[:n], prev, step)
		prev, step = p, s
		out = append(out, raw...)
		pcm = pcm[n:]
	}
	return out
}

// IMAAPMDecode expands mono IMA APM ADPCM. State starts at zero.
func IMAAPMDecode(data []byte) []int16 {
	return IMASSIDecode(data)
}
