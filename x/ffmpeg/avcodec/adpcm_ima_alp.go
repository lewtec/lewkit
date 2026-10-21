package avcodec

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// alpCodec encodes mono IMA ALP ADPCM. A full frame is 2048 samples.
// Predictor and step carry. An odd trailing sample is dropped.
type alpCodec struct {
	pcm  []int16
	prev int
	step int
}

func newIMAALP(Parameters) *alpCodec { return &alpCodec{} }

func (c *alpCodec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil || frame.NbSamples == 0 {
		return nil, nil
	}
	if frame.SampleFmt != avutil.SampleS16 || frame.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: ima alp", avutil.ErrInvalidData)
	}
	if len(frame.Data) == 0 || len(frame.Data[0]) < frame.NbSamples*2 {
		return nil, fmt.Errorf("%w: ima alp samples", avutil.ErrInvalidData)
	}
	for i := 0; i < frame.NbSamples; i++ {
		c.pcm = append(c.pcm, int16(binary.LittleEndian.Uint16(frame.Data[0][i*2:])))
	}
	return c.emitALP(false)
}

func (c *alpCodec) Flush() ([]*avutil.Packet, error) { return c.emitALP(true) }

func (c *alpCodec) emitALP(flush bool) ([]*avutil.Packet, error) {
	const frame = 2048
	var pkts []*avutil.Packet
	for len(c.pcm) >= frame || (flush && len(c.pcm) >= 2) {
		n := frame
		if n > len(c.pcm) {
			n = len(c.pcm) - len(c.pcm)%2
		}
		raw, prev, step := alpPacket(c.pcm[:n], c.prev, c.step)
		c.prev, c.step = prev, step
		c.pcm = append([]int16(nil), c.pcm[n:]...)
		pkts = append(pkts, &avutil.Packet{Data: raw, Duration: int64(n)})
	}
	if flush {
		c.pcm = nil
	}
	return pkts, nil
}

// IMAALPEncode packs mono PCM into IMA ALP ADPCM.
// Nibbles are high-first. The step update uses (nibble&7)*step>>2.
func IMAALPEncode(pcm []int16) []byte {
	const frame = 2048
	var out []byte
	prev, step := 0, 0
	for len(pcm) >= 2 {
		n := frame
		if n > len(pcm) {
			n = len(pcm) - len(pcm)%2
		}
		raw, p, s := alpPacket(pcm[:n], prev, step)
		prev, step = p, s
		out = append(out, raw...)
		pcm = pcm[n:]
	}
	return out
}

func alpPacket(pcm []int16, prev, step int) ([]byte, int, int) {
	out := make([]byte, len(pcm)/2)
	for i := 0; i < len(out); i++ {
		hi := alpCompress(&prev, &step, pcm[i*2])
		lo := alpCompress(&prev, &step, pcm[i*2+1])
		out[i] = hi<<4 | lo
	}
	return out, prev, step
}

func alpCompress(prev, step *int, sample int16) byte {
	delta := int(sample) - *prev
	st := int(imaStep[*step])
	sign := 0
	mag := delta
	if delta < 0 {
		sign = 8
		mag = -delta
	}
	nibble := mag * 4 / st
	if nibble > 7 {
		nibble = 7
	}
	diff := (st * nibble) >> 2
	if sign != 0 {
		diff = -diff
	}
	nibble |= sign
	*prev = clipInt16(*prev + diff)
	*step += int(imaIndex[nibble])
	if *step < 0 {
		*step = 0
	} else if *step > 88 {
		*step = 88
	}
	return byte(nibble)
}
