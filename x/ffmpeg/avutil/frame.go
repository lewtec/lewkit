package avutil

import "fmt"

// Frame is one decoded picture or one block of PCM.
// Data holds up to eight planes. Linesize is the byte stride of that plane.
type Frame struct {
	Data       [8][]byte
	Linesize   [8]int
	Width      int
	Height     int
	Format     PixelFormat
	SampleFmt  SampleFormat
	SampleRate int
	Layout     ChannelLayout
	NbSamples  int
	PTS        int64
	Duration   int64
	TimeBase   Rational
	SAR        Rational
	Key        bool
	Media      MediaType
	Palette    []byte
}

// NewVideo allocates a picture. Strides are padded to 32 bytes.
func NewVideo(format PixelFormat, width, height int) (*Frame, error) {
	item, ok := Desc(format)
	if !ok || width <= 0 || height <= 0 {
		return nil, fmt.Errorf("%w: %s %dx%d", ErrInvalidData, format, width, height)
	}
	frame := &Frame{
		Width: width, Height: height, Format: format, Media: MediaVideo, Key: true,
		TimeBase: Q(1, 25),
	}
	align := func(n int) int { return (n + 31) &^ 31 }
	switch format {
	case PixYUV420P, PixYUVJ420P, PixYUV422P, PixYUVJ422P, PixYUV444P, PixYUVJ444P, PixYUV410P, PixYUV411P:
		cw, ch := ChromaWidth(format, width), ChromaHeight(format, height)
		ys := align(width)
		cs := align(cw)
		frame.Linesize[0], frame.Linesize[1], frame.Linesize[2] = ys, cs, cs
		frame.Data[0] = make([]byte, ys*height)
		frame.Data[1] = make([]byte, cs*ch)
		frame.Data[2] = make([]byte, cs*ch)
	case PixYUVA420P:
		cw, ch := ChromaWidth(format, width), ChromaHeight(format, height)
		ys := align(width)
		cs := align(cw)
		frame.Linesize[0], frame.Linesize[1], frame.Linesize[2], frame.Linesize[3] = ys, cs, cs, ys
		frame.Data[0] = make([]byte, ys*height)
		frame.Data[1] = make([]byte, cs*ch)
		frame.Data[2] = make([]byte, cs*ch)
		frame.Data[3] = make([]byte, ys*height)
		for i := range frame.Data[3] {
			frame.Data[3][i] = 255
		}
	case PixNV12, PixNV21:
		cw, ch := ChromaWidth(format, width), ChromaHeight(format, height)
		ys := align(width)
		cs := align(cw * 2)
		frame.Linesize[0], frame.Linesize[1] = ys, cs
		frame.Data[0] = make([]byte, ys*height)
		frame.Data[1] = make([]byte, cs*ch)
	case PixGBRP:
		stride := align(width)
		frame.Linesize[0], frame.Linesize[1], frame.Linesize[2] = stride, stride, stride
		frame.Data[0] = make([]byte, stride*height)
		frame.Data[1] = make([]byte, stride*height)
		frame.Data[2] = make([]byte, stride*height)
	case PixGray, PixPAL8:
		stride := align(width)
		frame.Linesize[0] = stride
		frame.Data[0] = make([]byte, stride*height)
		if format == PixPAL8 {
			frame.Palette = grayPalette()
		}
	case PixMonoW, PixMonoB:
		stride := align((width + 7) / 8)
		frame.Linesize[0] = stride
		frame.Data[0] = make([]byte, stride*height)
		if format == PixMonoW {
			for i := range frame.Data[0] {
				frame.Data[0][i] = 0xff
			}
		}
	default:
		packed := item.Packed
		if packed == 0 {
			return nil, fmt.Errorf("%w: %s", ErrUnsupported, format)
		}
		stride := align(width * packed)
		frame.Linesize[0] = stride
		frame.Data[0] = make([]byte, stride*height)
		if item.Alpha {
			fillOpaque(format, frame.Data[0], stride, width, height)
		}
	}
	return frame, nil
}

func grayPalette() []byte {
	out := make([]byte, 256*4)
	for i := 0; i < 256; i++ {
		out[i*4] = byte(i)
		out[i*4+1] = byte(i)
		out[i*4+2] = byte(i)
		out[i*4+3] = 255
	}
	return out
}

func fillOpaque(format PixelFormat, data []byte, stride, width, height int) {
	offset := 3
	switch format {
	case PixARGB, PixABGR:
		offset = 0
	case PixRGBA, PixBGRA:
		offset = 3
	default:
		return
	}
	for y := 0; y < height; y++ {
		row := data[y*stride:]
		for x := 0; x < width; x++ {
			row[x*4+offset] = 255
		}
	}
}

// NewAudio allocates interleaved or planar PCM.
func NewAudio(format SampleFormat, rate int, layout ChannelLayout, samples int) (*Frame, error) {
	width := BytesPerSample(format)
	channels := layout.Channels()
	if width == 0 || rate <= 0 || channels <= 0 || samples < 0 {
		return nil, fmt.Errorf("%w: audio format", ErrInvalidData)
	}
	frame := &Frame{
		SampleFmt: format, SampleRate: rate, Layout: layout, NbSamples: samples,
		Media: MediaAudio, Key: true, TimeBase: Q(1, int32(rate)),
	}
	if IsPlanarSample(format) {
		stride := samples * width
		for c := 0; c < channels && c < 8; c++ {
			frame.Linesize[c] = stride
			frame.Data[c] = make([]byte, stride)
		}
		if channels > 8 {
			return nil, fmt.Errorf("%w: %d channels", ErrRange, channels)
		}
		return frame, nil
	}
	stride := samples * channels * width
	frame.Linesize[0] = stride
	frame.Data[0] = make([]byte, stride)
	return frame, nil
}

// PlaneRow returns row y of plane, sliced to the visible byte width.
func (frame *Frame) PlaneRow(plane, y, width int) []byte {
	stride := frame.Linesize[plane]
	if stride <= 0 || y < 0 {
		return nil
	}
	start := y * stride
	if start > len(frame.Data[plane]) {
		return nil
	}
	row := frame.Data[plane][start:]
	if width > len(row) {
		width = len(row)
	}
	if width < 0 {
		return nil
	}
	return row[:width]
}
