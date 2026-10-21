package avcodec

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

type pcmCodec struct{ par Parameters }

func (c *pcmCodec) Decode(pkt *avutil.Packet) ([]*avutil.Frame, error) {
	if pkt == nil || len(pkt.Data) == 0 {
		return nil, nil
	}
	if c.par.ID == IDPCMS24LE {
		return c.decodeS24(pkt)
	}
	format := pcmFormat(c.par.ID)
	width := avutil.BytesPerSample(format)
	channels := c.par.Layout.Channels()
	if channels <= 0 {
		channels = 1
	}
	if width == 0 || len(pkt.Data)%(width*channels) != 0 {
		return nil, fmt.Errorf("%w: pcm", avutil.ErrInvalidData)
	}
	samples := len(pkt.Data) / (width * channels)
	rate := c.par.SampleRate
	if rate <= 0 {
		rate = 44100
	}
	frame, err := avutil.NewAudio(format, rate, avutil.ChannelLayout{Count: channels, Mask: c.par.Layout.Mask}, samples)
	if err != nil {
		return nil, err
	}
	if c.par.Layout.Mask != 0 {
		frame.Layout = c.par.Layout
	} else if channels == 1 {
		frame.Layout = avutil.Mono()
	} else if channels == 2 {
		frame.Layout = avutil.Stereo()
	}
	copy(frame.Data[0], pkt.Data)
	if c.par.ID == IDPCMS16BE {
		avutil.Bswap16(frame.Data[0])
	}
	frame.PTS = pkt.PTS
	frame.Duration = int64(samples)
	frame.TimeBase = avutil.Q(1, int32(rate))
	return []*avutil.Frame{frame}, nil
}

func (c *pcmCodec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil || frame.NbSamples == 0 {
		return nil, nil
	}
	if c.par.ID == IDPCMS24LE {
		return c.encodeS24(frame)
	}
	format := pcmFormat(c.par.ID)
	if frame.SampleFmt != format && !(c.par.ID == IDPCMS16BE && frame.SampleFmt == avutil.SampleS16) {
		return nil, fmt.Errorf("%w: pcm %s", avutil.ErrInvalidData, frame.SampleFmt)
	}
	channels := frame.Layout.Channels()
	width := avutil.BytesPerSample(format)
	raw := append([]byte(nil), frame.Data[0][:frame.NbSamples*channels*width]...)
	if c.par.ID == IDPCMS16BE {
		avutil.Bswap16(raw)
	}
	return []*avutil.Packet{{
		Data: raw, PTS: frame.PTS, Duration: int64(frame.NbSamples),
		TimeBase: frame.TimeBase, Key: true,
	}}, nil
}

func pcmFormat(id ID) avutil.SampleFormat {
	switch id {
	case IDPCMU8:
		return avutil.SampleU8
	case IDPCMS16LE, IDPCMS16BE:
		return avutil.SampleS16
	case IDPCMS24LE:
		return avutil.SampleS32
	case IDPCMS32LE:
		return avutil.SampleS32
	case IDPCMF32LE:
		return avutil.SampleFLT
	default:
		return 0
	}
}

func (c *pcmCodec) decodeS24(pkt *avutil.Packet) ([]*avutil.Frame, error) {
	channels := c.par.Layout.Channels()
	if channels <= 0 {
		channels = 1
	}
	if len(pkt.Data)%(3*channels) != 0 {
		return nil, fmt.Errorf("%w: pcm_s24le", avutil.ErrInvalidData)
	}
	samples := len(pkt.Data) / (3 * channels)
	rate := c.par.SampleRate
	if rate <= 0 {
		rate = 44100
	}
	layout := layoutOf(c.par.Layout, channels)
	frame, err := avutil.NewAudio(avutil.SampleS32, rate, layout, samples)
	if err != nil {
		return nil, err
	}
	for i := 0; i < samples*channels; i++ {
		v := int32(pkt.Data[i*3]) | int32(pkt.Data[i*3+1])<<8 | int32(pkt.Data[i*3+2])<<16
		if v&0x800000 != 0 {
			v |= ^0xffffff
		}
		binary.LittleEndian.PutUint32(frame.Data[0][i*4:], uint32(v))
	}
	frame.PTS = pkt.PTS
	frame.Duration = int64(samples)
	frame.TimeBase = avutil.Q(1, int32(rate))
	return []*avutil.Frame{frame}, nil
}

func (c *pcmCodec) encodeS24(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame.SampleFmt != avutil.SampleS32 {
		return nil, fmt.Errorf("%w: pcm_s24le %s", avutil.ErrInvalidData, frame.SampleFmt)
	}
	channels := frame.Layout.Channels()
	raw := make([]byte, frame.NbSamples*channels*3)
	for i := 0; i < frame.NbSamples*channels; i++ {
		v := binary.LittleEndian.Uint32(frame.Data[0][i*4:])
		raw[i*3] = byte(v)
		raw[i*3+1] = byte(v >> 8)
		raw[i*3+2] = byte(v >> 16)
	}
	return []*avutil.Packet{{
		Data: raw, PTS: frame.PTS, Duration: int64(frame.NbSamples),
		TimeBase: frame.TimeBase, Key: true,
	}}, nil
}
