package avcodec

import (
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

type rawCodec struct{ par Parameters }

func (c *rawCodec) Decode(pkt *avutil.Packet) ([]*avutil.Frame, error) {
	if pkt == nil || len(pkt.Data) == 0 {
		return nil, nil
	}
	format := c.par.Format
	if format == 0 {
		format = avutil.PixYUV420P
	}
	frame, err := avutil.NewVideo(format, c.par.Width, c.par.Height)
	if err != nil {
		return nil, err
	}
	if err := unpackRaw(frame, pkt.Data); err != nil {
		return nil, err
	}
	frame.PTS = pkt.PTS
	frame.Duration = pkt.Duration
	if frame.Duration == 0 {
		frame.Duration = 1
	}
	return []*avutil.Frame{frame}, nil
}

func (c *rawCodec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil {
		return nil, nil
	}
	return []*avutil.Packet{{
		Data: packRaw(frame), PTS: frame.PTS, Duration: frame.Duration,
		TimeBase: frame.TimeBase, Key: true,
	}}, nil
}

func packRaw(frame *avutil.Frame) []byte {
	var out []byte
	switch frame.Format {
	case avutil.PixYUV420P, avutil.PixYUVJ420P, avutil.PixYUV422P, avutil.PixYUVJ422P, avutil.PixYUV444P, avutil.PixYUVJ444P, avutil.PixYUV410P, avutil.PixYUV411P, avutil.PixYUVA420P:
		planes := 3
		if frame.Format == avutil.PixYUVA420P {
			planes = 4
		}
		for plane := 0; plane < planes; plane++ {
			w, h := frame.Width, frame.Height
			if plane == 1 || plane == 2 {
				w = avutil.ChromaWidth(frame.Format, frame.Width)
				h = avutil.ChromaHeight(frame.Format, frame.Height)
			}
			for y := 0; y < h; y++ {
				out = append(out, frame.PlaneRow(plane, y, w)...)
			}
		}
	case avutil.PixNV12, avutil.PixNV21:
		for y := 0; y < frame.Height; y++ {
			out = append(out, frame.PlaneRow(0, y, frame.Width)...)
		}
		ch := avutil.ChromaHeight(frame.Format, frame.Height)
		cw := avutil.ChromaWidth(frame.Format, frame.Width) * 2
		for y := 0; y < ch; y++ {
			out = append(out, frame.PlaneRow(1, y, cw)...)
		}
	case avutil.PixGBRP:
		for plane := 0; plane < 3; plane++ {
			for y := 0; y < frame.Height; y++ {
				out = append(out, frame.PlaneRow(plane, y, frame.Width)...)
			}
		}
	case avutil.PixGray, avutil.PixPAL8:
		for y := 0; y < frame.Height; y++ {
			out = append(out, frame.PlaneRow(0, y, frame.Width)...)
		}
	default:
		item, ok := avutil.Desc(frame.Format)
		packed := 3
		if ok && item.Packed > 0 {
			packed = item.Packed
		}
		for y := 0; y < frame.Height; y++ {
			out = append(out, frame.PlaneRow(0, y, frame.Width*packed)...)
		}
	}
	return out
}

func unpackRaw(frame *avutil.Frame, raw []byte) error {
	need := len(packRaw(frame))
	if len(raw) < need {
		return fmt.Errorf("%w: rawvideo %d", avutil.ErrInvalidData, len(raw))
	}
	off := 0
	take := func(plane, y, width int) {
		row := frame.PlaneRow(plane, y, width)
		copy(row, raw[off:off+width])
		off += width
	}
	switch frame.Format {
	case avutil.PixYUV420P, avutil.PixYUVJ420P, avutil.PixYUV422P, avutil.PixYUVJ422P, avutil.PixYUV444P, avutil.PixYUVJ444P, avutil.PixYUV410P, avutil.PixYUV411P, avutil.PixYUVA420P:
		planes := 3
		if frame.Format == avutil.PixYUVA420P {
			planes = 4
		}
		for plane := 0; plane < planes; plane++ {
			w, h := frame.Width, frame.Height
			if plane == 1 || plane == 2 {
				w = avutil.ChromaWidth(frame.Format, frame.Width)
				h = avutil.ChromaHeight(frame.Format, frame.Height)
			}
			for y := 0; y < h; y++ {
				take(plane, y, w)
			}
		}
	case avutil.PixNV12, avutil.PixNV21:
		for y := 0; y < frame.Height; y++ {
			take(0, y, frame.Width)
		}
		ch := avutil.ChromaHeight(frame.Format, frame.Height)
		cw := avutil.ChromaWidth(frame.Format, frame.Width) * 2
		for y := 0; y < ch; y++ {
			take(1, y, cw)
		}
	case avutil.PixGBRP:
		for plane := 0; plane < 3; plane++ {
			for y := 0; y < frame.Height; y++ {
				take(plane, y, frame.Width)
			}
		}
	case avutil.PixGray, avutil.PixPAL8:
		for y := 0; y < frame.Height; y++ {
			take(0, y, frame.Width)
		}
	default:
		item, ok := avutil.Desc(frame.Format)
		packed := 3
		if ok && item.Packed > 0 {
			packed = item.Packed
		}
		for y := 0; y < frame.Height; y++ {
			take(0, y, frame.Width*packed)
		}
	}
	return nil
}
