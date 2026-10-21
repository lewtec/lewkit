package avcodec

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// swfCodec encodes mono Shockwave Flash ADPCM. Each frame is 4096 samples.
type swfCodec struct {
	pcm  []int16
	step int
}

func newSWF(par Parameters) (*swfCodec, error) {
	switch par.SampleRate {
	case 0, 11025, 22050, 44100:
		return &swfCodec{}, nil
	default:
		return nil, fmt.Errorf("%w: adpcm swf rate %d", avutil.ErrInvalidData, par.SampleRate)
	}
}

func (c *swfCodec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil || frame.NbSamples == 0 {
		return nil, nil
	}
	if frame.SampleFmt != avutil.SampleS16 || frame.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: adpcm swf", avutil.ErrInvalidData)
	}
	if len(frame.Data) == 0 || len(frame.Data[0]) < frame.NbSamples*2 {
		return nil, fmt.Errorf("%w: adpcm swf samples", avutil.ErrInvalidData)
	}
	for i := 0; i < frame.NbSamples; i++ {
		c.pcm = append(c.pcm, int16(binary.LittleEndian.Uint16(frame.Data[0][i*2:])))
	}
	return c.emitSWF(false)
}

func (c *swfCodec) Flush() ([]*avutil.Packet, error) { return c.emitSWF(true) }

func (c *swfCodec) emitSWF(flush bool) ([]*avutil.Packet, error) {
	const frame = 4096
	var pkts []*avutil.Packet
	for len(c.pcm) >= frame {
		chunk := append([]int16(nil), c.pcm[:frame]...)
		c.pcm = c.pcm[frame:]
		raw, step := swfPacket(chunk, c.step)
		c.step = step
		pkts = append(pkts, &avutil.Packet{Data: raw, Duration: frame})
	}
	if flush {
		c.pcm = nil
	}
	return pkts, nil
}

// SWFEncode packs mono PCM into Shockwave ADPCM frames of 4096 samples.
// A trailing partial frame is omitted. rate must be 11025, 22050, or 44100;
// the rate is checked by the caller because the bitstream does not store it.
func SWFEncode(pcm []int16) []byte {
	const frame = 4096
	var out []byte
	step := 0
	for len(pcm) >= frame {
		raw, next := swfPacket(pcm[:frame], step)
		step = next
		out = append(out, raw...)
		pcm = pcm[frame:]
	}
	return out
}

func swfPacket(pcm []int16, step int) ([]byte, int) {
	if step > 63 {
		step = 63
	}
	var w bitw
	w.u(2, 2)
	w.u(uint32(uint16(pcm[0])), 16)
	w.u(uint32(step), 6)
	prev := int(pcm[0])
	for i := 1; i < len(pcm); i++ {
		w.u(uint32(imaCompress(&prev, &step, pcm[i])), 4)
	}
	return w.bytes(), step
}
