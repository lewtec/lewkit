package avcodec

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// qtCodec encodes mono IMA QuickTime ADPCM in 64-sample packets.
type qtCodec struct {
	pcm  []int16
	prev int
	step int
}

func newIMAQT(Parameters) *qtCodec { return &qtCodec{} }

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
