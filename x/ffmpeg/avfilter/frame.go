package avfilter

import (
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
	"github.com/lewtec/lewkit/x/ffmpeg/swscale"
)

func copyMeta(dst, src *avutil.Frame) {
	dst.PTS = src.PTS
	dst.Duration = src.Duration
	dst.TimeBase = src.TimeBase
	dst.SAR = src.SAR
	dst.Key = src.Key
}

func cloneVideo(in *avutil.Frame) (*avutil.Frame, error) {
	dst, err := avutil.NewVideo(in.Format, in.Width, in.Height)
	if err != nil {
		return nil, err
	}
	for i := 0; i < 8; i++ {
		if len(in.Data[i]) == 0 || len(dst.Data[i]) == 0 {
			continue
		}
		n := len(in.Data[i])
		if n > len(dst.Data[i]) {
			n = len(dst.Data[i])
		}
		copy(dst.Data[i][:n], in.Data[i][:n])
	}
	if len(in.Palette) > 0 {
		dst.Palette = append([]byte(nil), in.Palette...)
	}
	copyMeta(dst, in)
	return dst, nil
}

func cloneAudio(in *avutil.Frame) (*avutil.Frame, error) {
	dst, err := avutil.NewAudio(in.SampleFmt, in.SampleRate, in.Layout, in.NbSamples)
	if err != nil {
		return nil, err
	}
	for i := 0; i < 8; i++ {
		if len(in.Data[i]) == 0 || len(dst.Data[i]) == 0 {
			continue
		}
		n := len(in.Data[i])
		if n > len(dst.Data[i]) {
			n = len(dst.Data[i])
		}
		copy(dst.Data[i][:n], in.Data[i][:n])
	}
	copyMeta(dst, in)
	return dst, nil
}

func toRGB(in *avutil.Frame, flags swscale.Flags) (*avutil.Frame, error) {
	if in.Format == avutil.PixRGB24 {
		return cloneVideo(in)
	}
	dst, err := avutil.NewVideo(avutil.PixRGB24, in.Width, in.Height)
	if err != nil {
		return nil, err
	}
	sw, err := swscale.New(in.Width, in.Height, in.Format, in.Width, in.Height, avutil.PixRGB24, flags)
	if err != nil {
		return nil, err
	}
	if err := sw.Scale(dst, in); err != nil {
		return nil, err
	}
	copyMeta(dst, in)
	return dst, nil
}

func fromRGB(in *avutil.Frame, format avutil.PixelFormat, flags swscale.Flags) (*avutil.Frame, error) {
	if format == avutil.PixRGB24 || format == 0 {
		return in, nil
	}
	dst, err := avutil.NewVideo(format, in.Width, in.Height)
	if err != nil {
		return nil, err
	}
	sw, err := swscale.New(in.Width, in.Height, avutil.PixRGB24, in.Width, in.Height, format, flags)
	if err != nil {
		return nil, err
	}
	if err := sw.Scale(dst, in); err != nil {
		return nil, err
	}
	copyMeta(dst, in)
	return dst, nil
}

func putRGB24(frame *avutil.Frame, x, y int, r, g, b byte) {
	row := frame.PlaneRow(0, y, frame.Width*3)
	if x < 0 || x*3+2 >= len(row) {
		return
	}
	row[x*3] = r
	row[x*3+1] = g
	row[x*3+2] = b
}

func getRGB24(frame *avutil.Frame, x, y int) (byte, byte, byte) {
	row := frame.PlaneRow(0, y, frame.Width*3)
	if x < 0 || x*3+2 >= len(row) {
		return 0, 0, 0
	}
	return row[x*3], row[x*3+1], row[x*3+2]
}

func fillRGB(frame *avutil.Frame, c [4]byte) {
	for y := 0; y < frame.Height; y++ {
		row := frame.PlaneRow(0, y, frame.Width*3)
		for x := 0; x < frame.Width; x++ {
			row[x*3] = c[0]
			row[x*3+1] = c[1]
			row[x*3+2] = c[2]
		}
	}
}

func videoTime(frame *avutil.Frame) float64 {
	if frame.TimeBase.Den == 0 {
		return 0
	}
	return float64(frame.PTS) * frame.TimeBase.Float64()
}

func one(frame *avutil.Frame) []*avutil.Frame { return []*avutil.Frame{frame} }

func badFilter(name string) error {
	return fmt.Errorf("%w: filter %s", avutil.ErrUnsupported, name)
}
