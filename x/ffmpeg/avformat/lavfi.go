package avformat

import (
	"context"
	"io"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avfilter"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

type lavfiDemux struct {
	g       *avfilter.Graph
	streams []Stream
	enc     avcodec.Encoder
	q       []*avutil.Packet
	done    bool
	primed  bool
}

func openLavfi(ctx context.Context, spec string, opt Options) (Demuxer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	graph, err := avfilter.Parse(spec, 0)
	if err != nil {
		return nil, err
	}
	d := &lavfiDemux{g: graph}
	frame, err := graph.Next()
	if err == io.EOF {
		d.done = true
		return d, nil
	}
	if err != nil {
		return nil, err
	}
	if err := d.take(frame); err != nil {
		return nil, err
	}
	d.primed = true
	_ = opt
	return d, nil
}

func (d *lavfiDemux) Streams() []Stream { return d.streams }

func (d *lavfiDemux) ReadPacket() (*avutil.Packet, error) {
	for len(d.q) == 0 && !d.done {
		frame, err := d.g.Next()
		if err == io.EOF {
			d.done = true
			if d.enc != nil {
				rest, ferr := d.enc.Flush()
				if ferr != nil {
					return nil, ferr
				}
				d.q = append(d.q, rest...)
			}
			break
		}
		if err != nil {
			return nil, err
		}
		if err := d.take(frame); err != nil {
			return nil, err
		}
	}
	if len(d.q) == 0 {
		return nil, io.EOF
	}
	pkt := d.q[0]
	d.q = d.q[1:]
	return pkt, nil
}

func (d *lavfiDemux) Close() error {
	if d.enc != nil {
		return d.enc.Close()
	}
	return nil
}

func (d *lavfiDemux) take(frame *avutil.Frame) error {
	if d.enc == nil {
		par, err := lavfiPar(frame)
		if err != nil {
			return err
		}
		enc, err := avcodec.OpenEncoder(par)
		if err != nil {
			return err
		}
		d.enc = enc
		d.streams = []Stream{{Par: par, TimeBase: par.TimeBase, SAR: frame.SAR}}
	}
	pkts, err := d.enc.Encode(frame)
	if err != nil {
		return err
	}
	for _, pkt := range pkts {
		if pkt.Duration == 0 {
			pkt.Duration = frame.Duration
		}
		if pkt.TimeBase.Den == 0 {
			pkt.TimeBase = frame.TimeBase
		}
		pkt.PTS = frame.PTS
		pkt.DTS = frame.PTS
		d.q = append(d.q, pkt)
	}
	return nil
}

func lavfiPar(frame *avutil.Frame) (avcodec.Parameters, error) {
	if frame.Width > 0 {
		tb := frame.TimeBase
		if tb.Den == 0 {
			tb = avutil.Q(1, 25)
		}
		return avcodec.Parameters{
			Type: avutil.MediaVideo, ID: avcodec.IDRawVideo, Width: frame.Width, Height: frame.Height,
			Format: frame.Format, TimeBase: tb, SAR: frame.SAR,
		}, nil
	}
	id := avcodec.IDPCMS16LE
	switch frame.SampleFmt {
	case avutil.SampleU8:
		id = avcodec.IDPCMU8
	case avutil.SampleS32:
		id = avcodec.IDPCMS32LE
	case avutil.SampleFLT:
		id = avcodec.IDPCMF32LE
	}
	rate := frame.SampleRate
	if rate <= 0 {
		rate = 44100
	}
	tb := frame.TimeBase
	if tb.Den == 0 {
		tb = avutil.Q(1, int32(rate))
	}
	layout := frame.Layout
	if layout.Channels() <= 0 {
		layout = avutil.Mono()
	}
	return avcodec.Parameters{
		Type: avutil.MediaAudio, ID: id, SampleFormat: frame.SampleFmt, SampleRate: rate,
		Layout: layout, TimeBase: tb,
	}, nil
}
