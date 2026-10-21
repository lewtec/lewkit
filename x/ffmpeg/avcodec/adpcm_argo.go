package avcodec

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// argoCodec encodes and decodes mono ADPCM Argonaut.
// Each block is 32 samples and 17 bytes. A short tail is dropped.
// The previous two samples carry across blocks and start at zero.
type argoCodec struct {
	pcm     []int16
	sample1 int
	sample2 int
	rate    int
}

func newARGO(par Parameters) *argoCodec {
	rate := par.SampleRate
	if rate <= 0 {
		rate = 22050
	}
	return &argoCodec{rate: rate}
}

func (c *argoCodec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil || frame.NbSamples == 0 {
		return nil, nil
	}
	if frame.SampleFmt != avutil.SampleS16 || frame.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: adpcm argo", avutil.ErrInvalidData)
	}
	if len(frame.Data) == 0 || len(frame.Data[0]) < frame.NbSamples*2 {
		return nil, fmt.Errorf("%w: adpcm argo samples", avutil.ErrInvalidData)
	}
	for i := 0; i < frame.NbSamples; i++ {
		c.pcm = append(c.pcm, int16(binary.LittleEndian.Uint16(frame.Data[0][i*2:])))
	}
	return c.emitARGO(false)
}

func (c *argoCodec) Flush() ([]*avutil.Packet, error) { return c.emitARGO(true) }

func (c *argoCodec) emitARGO(flush bool) ([]*avutil.Packet, error) {
	const frame = 32
	var pkts []*avutil.Packet
	for len(c.pcm) >= frame || (flush && len(c.pcm) > 0) {
		var block [32]int16
		n := copy(block[:], c.pcm)
		raw, s1, s2 := argoBlock(block[:], c.sample1, c.sample2)
		c.sample1, c.sample2 = s1, s2
		c.pcm = append([]int16(nil), c.pcm[n:]...)
		pkts = append(pkts, &avutil.Packet{Data: raw, Duration: frame})
	}
	return pkts, nil
}

func (c *argoCodec) Decode(pkt *avutil.Packet) ([]*avutil.Frame, error) {
	if pkt == nil || len(pkt.Data) == 0 {
		return nil, nil
	}
	samples := argoExpand(pkt.Data, &c.sample1, &c.sample2)
	frame, err := pcmS16Mono(samples, c.rate, pkt.PTS)
	if err != nil || frame == nil {
		return nil, err
	}
	return []*avutil.Frame{frame}, nil
}

// ARGOEncode packs mono PCM into ADPCM Argonaut blocks of 32 samples.
// A trailing partial block is zero-padded, matching an encoder without
// a small last frame.
func ARGOEncode(pcm []int16) []byte {
	var out []byte
	s1, s2 := 0, 0
	for len(pcm) > 0 {
		var block [32]int16
		n := copy(block[:], pcm)
		raw, a, b := argoBlock(block[:], s1, s2)
		s1, s2 = a, b
		out = append(out, raw...)
		pcm = pcm[n:]
	}
	return out
}

// ARGODecode expands mono ADPCM Argonaut. State starts at zero.
func ARGODecode(data []byte) []int16 {
	s1, s2 := 0, 0
	return argoExpand(data, &s1, &s2)
}

func argoBlock(pcm []int16, sample1, sample2 int) ([]byte, int, int) {
	bestShift, bestFlag := 2, 0
	bestErr := int64(^uint64(0) >> 1)
	for shift := 2; shift < 18; shift++ {
		for flag := 0; flag < 2; flag++ {
			s1, s2 := sample1, sample2
			err := argoError(pcm, &s1, &s2, shift, flag)
			if err < bestErr {
				bestShift, bestFlag, bestErr = shift, flag, err
			}
			if err == 0 {
				break
			}
		}
		if bestErr == 0 {
			break
		}
	}
	out := make([]byte, 17)
	out[0] = byte((bestShift-2)<<4 | bestFlag<<2)
	s1, s2 := sample1, sample2
	for i := 0; i < 16; i++ {
		n1 := argoCompress(&s1, &s2, pcm[i*2], bestShift, bestFlag)
		n2 := argoCompress(&s1, &s2, pcm[i*2+1], bestShift, bestFlag)
		out[1+i] = n1<<4 | n2
	}
	return out, s1, s2
}

func argoError(pcm []int16, s1, s2 *int, shift, flag int) int64 {
	var err int64
	for _, s := range pcm {
		n := argoNibble(*s1, *s2, s, shift, flag)
		got := argoExpandOne(s1, s2, n, shift, flag)
		d := int(s) - int(got)
		if d < 0 {
			d = -d
		}
		err += int64(d)
	}
	return err
}

func argoNibble(s1, s2 int, sample int16, shift, flag int) int {
	var nibble int
	if flag != 0 {
		nibble = 4*int(sample) - 8*s1 + 4*s2
	} else {
		nibble = 4*int(sample) - 4*s1
	}
	return (nibble >> shift) & 0x0F
}

func argoCompress(s1, s2 *int, sample int16, shift, flag int) byte {
	n := argoNibble(*s1, *s2, sample, shift, flag)
	argoExpandOne(s1, s2, n, shift, flag)
	return byte(n)
}

func argoExpand(data []byte, s1, s2 *int) []int16 {
	var out []int16
	for len(data) >= 17 {
		control := int(data[0])
		shift := (control >> 4) + 2
		flag := 0
		if control&0x04 != 0 {
			flag = 1
		}
		for i := 0; i < 16; i++ {
			b := int(data[1+i])
			out = append(out, argoExpandOne(s1, s2, b>>4, shift, flag))
			out = append(out, argoExpandOne(s1, s2, b&0x0F, shift, flag))
		}
		data = data[17:]
	}
	return out
}

func argoExpandOne(s1, s2 *int, nibble, shift, flag int) int16 {
	n := nibble & 0x0F
	if n >= 8 {
		n -= 16
	}
	sample := n * (1 << shift)
	if flag != 0 {
		sample += 8**s1 - 4**s2
	} else {
		sample += 4 * *s1
	}
	sample = clipInt16(sample >> 2)
	*s2 = *s1
	*s1 = sample
	return int16(sample)
}
