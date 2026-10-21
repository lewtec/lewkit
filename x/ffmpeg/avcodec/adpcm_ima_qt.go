package avcodec

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// qtCodec encodes and decodes mono IMA QuickTime ADPCM in 64-sample packets.
type qtCodec struct {
	pcm  []int16
	prev int
	step int
	rate int
}

func newIMAQT(par Parameters) *qtCodec {
	rate := par.SampleRate
	if rate <= 0 {
		rate = 44100
	}
	return &qtCodec{rate: rate}
}

func (c *qtCodec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil || frame.NbSamples == 0 {
		return nil, nil
	}
	if frame.SampleFmt != avutil.SampleS16 || frame.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: ima qt", avutil.ErrInvalidData)
	}
	if len(frame.Data) == 0 || len(frame.Data[0]) < frame.NbSamples*2 {
		return nil, fmt.Errorf("%w: ima qt samples", avutil.ErrInvalidData)
	}
	for i := 0; i < frame.NbSamples; i++ {
		c.pcm = append(c.pcm, int16(binary.LittleEndian.Uint16(frame.Data[0][i*2:])))
	}
	return c.emitQT(false)
}

func (c *qtCodec) Flush() ([]*avutil.Packet, error) { return c.emitQT(true) }

func (c *qtCodec) emitQT(flush bool) ([]*avutil.Packet, error) {
	const frame = 64
	var pkts []*avutil.Packet
	for len(c.pcm) >= frame {
		chunk := append([]int16(nil), c.pcm[:frame]...)
		c.pcm = c.pcm[frame:]
		raw, prev, step := imaQTPacket(chunk, c.prev, c.step)
		c.prev, c.step = prev, step
		pkts = append(pkts, &avutil.Packet{Data: raw, Duration: frame})
	}
	if flush {
		c.pcm = nil
	}
	return pkts, nil
}

func (c *qtCodec) Decode(pkt *avutil.Packet) ([]*avutil.Frame, error) {
	if pkt == nil || len(pkt.Data) == 0 {
		return nil, nil
	}
	samples, err := c.qtDecode(pkt.Data)
	if err != nil || len(samples) == 0 {
		return nil, err
	}
	frame, err := pcmS16Mono(samples, c.rate, pkt.PTS)
	if err != nil || frame == nil {
		return nil, err
	}
	return []*avutil.Frame{frame}, nil
}

// IMAQTDecode expands mono IMA QuickTime packets of 34 bytes from a reset state.
func IMAQTDecode(data []byte) ([]int16, error) {
	var c qtCodec
	return c.qtDecode(data)
}

func (c *qtCodec) qtDecode(data []byte) ([]int16, error) {
	if len(data)%34 != 0 {
		return nil, fmt.Errorf("%w: ima qt size", avutil.ErrInvalidData)
	}
	out := make([]int16, 0, len(data)/34*64)
	for off := 0; off < len(data); off += 34 {
		samples, err := qtChunk(&c.prev, &c.step, data[off:off+34])
		if err != nil {
			return nil, err
		}
		out = append(out, samples...)
	}
	return out, nil
}

func qtChunk(prev, step *int, chunk []byte) ([]int16, error) {
	hdr := int(int16(binary.BigEndian.Uint16(chunk[:2])))
	stepIndex := hdr & 0x7F
	predictor := hdr &^ 0x7F
	if *step == stepIndex {
		diff := predictor - *prev
		if diff < 0 {
			diff = -diff
		}
		if diff > 0x7f {
			*step = stepIndex
			*prev = predictor
		}
	} else {
		*step = stepIndex
		*prev = predictor
	}
	if *step < 0 || *step > 88 {
		return nil, fmt.Errorf("%w: ima qt step %d", avutil.ErrInvalidData, *step)
	}
	out := make([]int16, 64)
	o := 0
	for i := 2; i < 34; i++ {
		v := int(chunk[i])
		out[o] = imaQTExpand(prev, step, v&0x0F)
		out[o+1] = imaQTExpand(prev, step, v>>4)
		o += 2
	}
	return out, nil
}

// imaQTExpand is ffmpeg's adpcm_ima_qt_expand_nibble. The bit adds are not
// the same as the IMA WAV multiply when the step does not divide evenly.
func imaQTExpand(pred, step *int, nibble int) int16 {
	st := int(imaStep[*step])
	idx := *step + int(imaIndex[nibble])
	if idx < 0 {
		idx = 0
	} else if idx > 88 {
		idx = 88
	}
	diff := st >> 3
	if nibble&4 != 0 {
		diff += st
	}
	if nibble&2 != 0 {
		diff += st >> 1
	}
	if nibble&1 != 0 {
		diff += st >> 2
	}
	p := *pred
	if nibble&8 != 0 {
		p -= diff
	} else {
		p += diff
	}
	*pred = clipInt16(p)
	*step = idx
	return int16(*pred)
}

// IMAQTEncode packs mono PCM into IMA QuickTime packets of 64 samples.
// A trailing partial packet is omitted, matching ffmpeg without a small last frame.
func IMAQTEncode(pcm []int16) []byte {
	const frame = 64
	var out []byte
	prev, step := 0, 0
	for len(pcm) >= frame {
		raw, p, s := imaQTPacket(pcm[:frame], prev, step)
		prev, step = p, s
		out = append(out, raw...)
		pcm = pcm[frame:]
	}
	return out
}

func imaQTPacket(pcm []int16, prev, step int) ([]byte, int, int) {
	out := make([]byte, 34)
	hdr := ((prev & 0xFFFF) >> 7 << 7) | (step & 0x7F)
	out[0] = byte(hdr >> 8)
	out[1] = byte(hdr)
	for i := 0; i < 64; i += 2 {
		t1 := imaQTCompress(&prev, &step, pcm[i])
		t2 := imaQTCompress(&prev, &step, pcm[i+1])
		out[2+i/2] = t2<<4 | t1
	}
	return out, prev, step
}

func imaQTCompress(prev, step *int, sample int16) byte {
	delta := int(sample) - *prev
	stepSize := int(imaStep[*step])
	nibble := 0
	if delta < 0 {
		nibble = 8
		delta = -delta
	}
	diff := delta + (stepSize >> 3)
	if delta >= stepSize {
		nibble |= 4
		delta -= stepSize
	}
	stepSize >>= 1
	if delta >= stepSize {
		nibble |= 2
		delta -= stepSize
	}
	stepSize >>= 1
	if delta >= stepSize {
		nibble |= 1
		delta -= stepSize
	}
	diff -= delta
	if nibble&8 != 0 {
		*prev -= diff
	} else {
		*prev += diff
	}
	if *prev > 32767 {
		*prev = 32767
	} else if *prev < -32768 {
		*prev = -32768
	}
	*step += int(imaIndex[nibble])
	if *step < 0 {
		*step = 0
	} else if *step > 88 {
		*step = 88
	}
	return byte(nibble)
}
