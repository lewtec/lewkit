package avcodec

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// swfCodec encodes and decodes mono Shockwave Flash ADPCM.
// Each encoded frame is 4096 samples. The bitstream stores the predictor and step.
type swfCodec struct {
	pcm  []int16
	step int
	rate int
}

func newSWF(par Parameters) (*swfCodec, error) {
	rate := par.SampleRate
	if rate <= 0 {
		rate = 44100
	}
	switch rate {
	case 11025, 22050, 44100:
		return &swfCodec{rate: rate}, nil
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

func (c *swfCodec) Decode(pkt *avutil.Packet) ([]*avutil.Frame, error) {
	if pkt == nil || len(pkt.Data) == 0 {
		return nil, nil
	}
	samples, err := SWFDecode(pkt.Data)
	if err != nil || len(samples) == 0 {
		return nil, err
	}
	frame, err := pcmS16Mono(samples, c.rate, pkt.PTS)
	if err != nil || frame == nil {
		return nil, err
	}
	return []*avutil.Frame{frame}, nil
}

// SWFDecode expands mono Shockwave ADPCM. The code size is the first two bits.
// This build's encoder writes 4-bit frames of 4096 samples.
func SWFDecode(data []byte) ([]int16, error) {
	const block = 2051
	if len(data) == 0 {
		return nil, nil
	}
	// Each WAV packet is one byte-aligned frame. A 4096-sample frame is
	// 16404 bits, so the last byte holds four padding bits.
	if len(data)%block == 0 {
		var out []int16
		for off := 0; off < len(data); off += block {
			samples, err := swfDecodePacket(data[off : off+block])
			if err != nil {
				return nil, err
			}
			out = append(out, samples...)
		}
		return out, nil
	}
	return swfDecodePacket(data)
}

func swfDecodePacket(data []byte) ([]int16, error) {
	r := bitr{b: data}
	nbBitsU, err := r.u(2)
	if err != nil {
		return nil, err
	}
	nbBits := int(nbBitsU) + 2
	if nbBits != 4 {
		return nil, fmt.Errorf("%w: adpcm swf bits %d", avutil.ErrUnsupported, nbBits)
	}
	size := len(data) * 8
	var out []int16
	for r.i <= size-22 {
		predBits, err := r.s(16)
		if err != nil {
			return nil, err
		}
		stepBits, err := r.u(6)
		if err != nil {
			return nil, err
		}
		pred := int(predBits)
		step := int(stepBits)
		if step > 88 {
			return nil, fmt.Errorf("%w: adpcm swf step %d", avutil.ErrInvalidData, step)
		}
		out = append(out, int16(pred))
		for count := 0; r.i <= size-nbBits && count < 4095; count++ {
			delta, err := r.u(nbBits)
			if err != nil {
				return nil, err
			}
			out = append(out, swfExpand(&pred, &step, int(delta), nbBits))
		}
	}
	return out, nil
}

func swfExpand(pred, step *int, delta, nbBits int) int16 {
	st := int(imaStep[*step])
	k := 1 << (nbBits - 2)
	sign := 1 << (nbBits - 1)
	vpdiff := 0
	for {
		if delta&k != 0 {
			vpdiff += st
		}
		st >>= 1
		k >>= 1
		if k == 0 {
			break
		}
	}
	vpdiff += st
	if delta&sign != 0 {
		*pred -= vpdiff
	} else {
		*pred += vpdiff
	}
	*step += int(imaIndex[delta&^sign])
	if *step < 0 {
		*step = 0
	} else if *step > 88 {
		*step = 88
	}
	*pred = clipInt16(*pred)
	return int16(*pred)
}

// SWFEncode packs mono PCM into Shockwave ADPCM frames of 4096 samples.
// A trailing partial frame is omitted. The rate is not stored in the bitstream.
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
