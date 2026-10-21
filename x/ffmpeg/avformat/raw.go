package avformat

import (
	"fmt"
	"io"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

func pcmCodec(name string) (avcodec.ID, int, bool) {
	switch name {
	case "u8":
		return avcodec.IDPCMU8, 1, true
	case "s16le":
		return avcodec.IDPCMS16LE, 2, true
	case "s16be":
		return avcodec.IDPCMS16BE, 2, true
	case "s24le":
		return avcodec.IDPCMS24LE, 3, true
	case "s32le":
		return avcodec.IDPCMS32LE, 4, true
	case "f32le":
		return avcodec.IDPCMF32LE, 4, true
	case "alaw":
		return avcodec.IDPCMALAW, 1, true
	case "mulaw":
		return avcodec.IDPCMMULAW, 1, true
	default:
		return 0, 0, false
	}
}

func sampleFormat(id avcodec.ID) avutil.SampleFormat {
	switch id {
	case avcodec.IDPCMU8:
		return avutil.SampleU8
	case avcodec.IDPCMS16LE, avcodec.IDPCMS16BE, avcodec.IDPCMALAW, avcodec.IDPCMMULAW:
		return avutil.SampleS16
	case avcodec.IDPCMS24LE, avcodec.IDPCMS32LE:
		return avutil.SampleS32
	case avcodec.IDPCMF32LE:
		return avutil.SampleFLT
	default:
		return avutil.SampleS16
	}
}

type memDemux struct {
	streams []Stream
	pkts    []*avutil.Packet
	i       int
}

func (d *memDemux) Streams() []Stream { return d.streams }

func (d *memDemux) ReadPacket() (*avutil.Packet, error) {
	if d.i >= len(d.pkts) {
		return nil, io.EOF
	}
	pkt := d.pkts[d.i]
	d.i++
	return pkt, nil
}

func (d *memDemux) Close() error { return nil }

func openRawAudio(format string, data []byte, opt Options) (Demuxer, error) {
	id, width, ok := pcmCodec(format)
	if !ok {
		return nil, fmt.Errorf("%w: %s", avutil.ErrUnsupported, format)
	}
	rate := opt.SampleRate
	if rate <= 0 {
		rate = 44100
	}
	channels := opt.Layout.Channels()
	if channels <= 0 {
		channels = 1
	}
	layout := layoutOr(opt.Layout, channels)
	stride := width * channels
	if stride <= 0 || (len(data) > 0 && len(data)%stride != 0 && format != "s24le") {
		if len(data)%stride != 0 {
			data = data[:len(data)-len(data)%stride]
		}
	}
	tb := avutil.Q(1, int32(rate))
	var pkts []*avutil.Packet
	const chunk = 1024
	for off := 0; off < len(data); {
		n := chunk * stride
		if off+n > len(data) {
			n = len(data) - off
		}
		n -= n % stride
		if n == 0 {
			break
		}
		pkts = append(pkts, &avutil.Packet{
			Data: append([]byte(nil), data[off:off+n]...),
			PTS:  int64(off / stride), DTS: int64(off / stride),
			Duration: int64(n / stride), TimeBase: tb, Key: true,
		})
		off += n
	}
	if len(pkts) == 0 {
		pkts = []*avutil.Packet{{Data: []byte{}, PTS: 0, DTS: 0, TimeBase: tb, Key: true}}
	}
	return &memDemux{streams: []Stream{{
		Par: avcodec.Parameters{
			Type: avutil.MediaAudio, ID: id, SampleFormat: sampleFormat(id),
			SampleRate: rate, Layout: layout, TimeBase: tb,
		},
		TimeBase: tb,
	}}, pkts: pkts}, nil
}

func openRawVideo(data []byte, opt Options) (Demuxer, error) {
	if opt.Width <= 0 || opt.Height <= 0 {
		return nil, fmt.Errorf("%w: rawvideo size", avutil.ErrOption)
	}
	format := opt.Format
	if format == 0 {
		format = avutil.PixYUV420P
	}
	size, err := rawFrameSize(format, opt.Width, opt.Height)
	if err != nil {
		return nil, err
	}
	if size <= 0 {
		return nil, fmt.Errorf("%w: rawvideo", avutil.ErrInvalidData)
	}
	rate := rateOr(opt.FrameRate)
	tb := rate.Inv()
	var pkts []*avutil.Packet
	for off, pts := 0, int64(0); off+size <= len(data); off, pts = off+size, pts+1 {
		pkts = append(pkts, &avutil.Packet{
			Data: append([]byte(nil), data[off:off+size]...),
			PTS:  pts, DTS: pts, Duration: 1, TimeBase: tb, Key: true,
		})
	}
	return &memDemux{streams: []Stream{{
		Par: avcodec.Parameters{
			Type: avutil.MediaVideo, ID: avcodec.IDRawVideo, Width: opt.Width, Height: opt.Height,
			Format: format, TimeBase: tb, SAR: avutil.Q(1, 1),
		},
		TimeBase: tb, SAR: avutil.Q(1, 1),
	}}, pkts: pkts}, nil
}

func rawFrameSize(format avutil.PixelFormat, w, h int) (int, error) {
	frame, err := avutil.NewVideo(format, w, h)
	if err != nil {
		return 0, err
	}
	enc, err := avcodec.OpenEncoder(avcodec.Parameters{
		Type: avutil.MediaVideo, ID: avcodec.IDRawVideo, Width: w, Height: h, Format: format,
	})
	if err != nil {
		return 0, err
	}
	pkts, err := enc.Encode(frame)
	if err != nil {
		return 0, err
	}
	if len(pkts) == 0 {
		return 0, fmt.Errorf("%w: rawvideo", avutil.ErrInvalidData)
	}
	return len(pkts[0].Data), nil
}
