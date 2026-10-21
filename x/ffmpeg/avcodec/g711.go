package avcodec

import (
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

type xlawCodec struct{ par Parameters }

func alawToLinear(a byte) int16 {
	a ^= 0x55
	t := int(a & 0x0f)
	seg := int(a&0x70) >> 4
	if seg != 0 {
		t = (t + t + 1 + 32) << (seg + 2)
	} else {
		t = (t + t + 1) << 3
	}
	if a&0x80 != 0 {
		return int16(t)
	}
	return int16(-t)
}

func ulawToLinear(u byte) int16 {
	u = ^u
	t := ((int(u&0x0f) << 3) + 0x84) << ((uint(u) & 0x70) >> 4)
	if u&0x80 != 0 {
		return int16(0x84 - t)
	}
	return int16(t - 0x84)
}

var (
	linearToAlaw [16384]byte
	linearToUlaw [16384]byte
	xlawReady    bool
)

func buildXlaw(dst []byte, decode func(byte) int16, mask int) {
	j := 1
	dst[8192] = byte(mask)
	for i := 0; i < 127; i++ {
		v1 := int(decode(byte(i ^ mask)))
		v2 := int(decode(byte((i + 1) ^ mask)))
		v := (v1 + v2 + 4) >> 3
		for ; j < v; j++ {
			dst[8192-j] = byte(i ^ (mask ^ 0x80))
			dst[8192+j] = byte(i ^ mask)
		}
	}
	for ; j < 8192; j++ {
		dst[8192-j] = byte(127 ^ (mask ^ 0x80))
		dst[8192+j] = byte(127 ^ mask)
	}
	dst[0] = dst[1]
}

func ensureXlaw() {
	if xlawReady {
		return
	}
	buildXlaw(linearToAlaw[:], alawToLinear, 0xd5)
	buildXlaw(linearToUlaw[:], ulawToLinear, 0xff)
	xlawReady = true
}

func linearIndex(sample int16) int {
	v := int(sample) >> 2
	if v < -8192 {
		v = -8192
	} else if v > 8191 {
		v = 8191
	}
	return v + 8192
}

func (c *xlawCodec) Decode(pkt *avutil.Packet) ([]*avutil.Frame, error) {
	if pkt == nil || len(pkt.Data) == 0 {
		return nil, nil
	}
	rate := c.par.SampleRate
	if rate <= 0 {
		rate = 8000
	}
	channels := c.par.Layout.Channels()
	if channels <= 0 {
		channels = 1
	}
	if len(pkt.Data)%channels != 0 {
		return nil, fmt.Errorf("%w: xlaw", avutil.ErrInvalidData)
	}
	samples := len(pkt.Data) / channels
	frame, err := avutil.NewAudio(avutil.SampleS16, rate, layoutOf(c.par.Layout, channels), samples)
	if err != nil {
		return nil, err
	}
	fn := alawToLinear
	if c.par.ID == IDPCMMULAW {
		fn = ulawToLinear
	}
	for i, b := range pkt.Data {
		v := fn(b)
		frame.Data[0][i*2] = byte(v)
		frame.Data[0][i*2+1] = byte(uint16(v) >> 8)
	}
	frame.PTS = pkt.PTS
	frame.Duration = int64(samples)
	return []*avutil.Frame{frame}, nil
}

func (c *xlawCodec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil || frame.NbSamples == 0 {
		return nil, nil
	}
	if frame.SampleFmt != avutil.SampleS16 {
		return nil, fmt.Errorf("%w: xlaw %s", avutil.ErrInvalidData, frame.SampleFmt)
	}
	ensureXlaw()
	table := linearToAlaw[:]
	if c.par.ID == IDPCMMULAW {
		table = linearToUlaw[:]
	}
	channels := frame.Layout.Channels()
	out := make([]byte, frame.NbSamples*channels)
	for i := 0; i < len(out); i++ {
		sample := int16(frame.Data[0][i*2]) | int16(frame.Data[0][i*2+1])<<8
		out[i] = table[linearIndex(sample)]
	}
	return []*avutil.Packet{{
		Data: out, PTS: frame.PTS, Duration: int64(frame.NbSamples),
		TimeBase: frame.TimeBase, Key: true,
	}}, nil
}

func layoutOf(layout avutil.ChannelLayout, channels int) avutil.ChannelLayout {
	if layout.Channels() == channels && (layout.Mask != 0 || layout.Count > 0) {
		if layout.Count == 0 {
			layout.Count = channels
		}
		return layout
	}
	if channels == 1 {
		return avutil.Mono()
	}
	if channels == 2 {
		return avutil.Stereo()
	}
	return avutil.ChannelLayout{Count: channels}
}
