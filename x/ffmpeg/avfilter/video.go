package avfilter

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
	"github.com/lewtec/lewkit/x/ffmpeg/swscale"
)

func newScale(args argSet, flags swscale.Flags) func(*avutil.Frame) ([]*avutil.Frame, error) {
	if text, ok := args.get("flags"); ok {
		flags = parseSWS(text)
	}
	return func(in *avutil.Frame) ([]*avutil.Frame, error) {
		if in.Media == avutil.MediaAudio || in.SampleRate != 0 && in.Width == 0 {
			return nil, fmt.Errorf("%w: scale on audio", avutil.ErrInvalidData)
		}
		vars := map[string]float64{
			"iw": inW(in), "ih": inH(in), "in_w": inW(in), "in_h": inH(in),
			"a": inW(in) / inH(in), "sar": sarOf(in), "dar": inW(in) / inH(in) * sarOf(in),
		}
		w, err := dim(args, 0, []string{"w", "width"}, int(in.Width), vars)
		if err != nil {
			return nil, err
		}
		h, err := dim(args, 1, []string{"h", "height"}, int(in.Height), vars)
		if err != nil {
			return nil, err
		}
		w, h = fitScale(w, h, in.Width, in.Height)
		format := in.Format
		if text, ok := args.get("format", "pix_fmt"); ok {
			format, err = avutil.ParsePixelFormat(text)
			if err != nil {
				return nil, err
			}
		}
		dst, err := avutil.NewVideo(format, w, h)
		if err != nil {
			return nil, err
		}
		sw, err := swscale.New(in.Width, in.Height, in.Format, w, h, format, flags)
		if err != nil {
			return nil, err
		}
		if err := sw.Scale(dst, in); err != nil {
			return nil, err
		}
		copyMeta(dst, in)
		return one(dst), nil
	}
}

func newFormat(args argSet, flags swscale.Flags) func(*avutil.Frame) ([]*avutil.Frame, error) {
	return func(in *avutil.Frame) ([]*avutil.Frame, error) {
		text, ok := args.at(0, "pix_fmts", "pix_fmt")
		if !ok {
			return nil, fmt.Errorf("%w: format", avutil.ErrOption)
		}
		name := strings.Split(text, "|")[0]
		format, err := avutil.ParsePixelFormat(name)
		if err != nil {
			return nil, err
		}
		if in.Format == format {
			out, err := cloneVideo(in)
			if err != nil {
				return nil, err
			}
			return one(out), nil
		}
		dst, err := avutil.NewVideo(format, in.Width, in.Height)
		if err != nil {
			return nil, err
		}
		sw, err := swscale.New(in.Width, in.Height, in.Format, in.Width, in.Height, format, flags)
		if err != nil {
			return nil, err
		}
		if err := sw.Scale(dst, in); err != nil {
			return nil, err
		}
		copyMeta(dst, in)
		return one(dst), nil
	}
}

func newCrop(args argSet, flags swscale.Flags) func(*avutil.Frame) ([]*avutil.Frame, error) {
	return func(in *avutil.Frame) ([]*avutil.Frame, error) {
		rgb, err := toRGB(in, flags)
		if err != nil {
			return nil, err
		}
		vars := map[string]float64{"iw": float64(rgb.Width), "ih": float64(rgb.Height), "in_w": float64(rgb.Width), "in_h": float64(rgb.Height)}
		w, err := dim(args, 0, []string{"w", "width"}, rgb.Width, vars)
		if err != nil {
			return nil, err
		}
		h, err := dim(args, 1, []string{"h", "height"}, rgb.Height, vars)
		if err != nil {
			return nil, err
		}
		if w <= 0 {
			w = rgb.Width
		}
		if h <= 0 {
			h = rgb.Height
		}
		x := (rgb.Width - w) / 2
		y := (rgb.Height - h) / 2
		if text, ok := args.at(2, "x"); ok {
			x, err = evalInt(text, vars)
			if err != nil {
				return nil, err
			}
		}
		if text, ok := args.at(3, "y"); ok {
			y, err = evalInt(text, vars)
			if err != nil {
				return nil, err
			}
		}
		dst, err := avutil.NewVideo(avutil.PixRGB24, w, h)
		if err != nil {
			return nil, err
		}
		for yy := 0; yy < h; yy++ {
			for xx := 0; xx < w; xx++ {
				r, g, b := getRGB24(rgb, x+xx, y+yy)
				putRGB24(dst, xx, yy, r, g, b)
			}
		}
		copyMeta(dst, in)
		out, err := fromRGB(dst, in.Format, flags)
		if err != nil {
			return nil, err
		}
		return one(out), nil
	}
}

func newPad(args argSet, flags swscale.Flags) func(*avutil.Frame) ([]*avutil.Frame, error) {
	return func(in *avutil.Frame) ([]*avutil.Frame, error) {
		rgb, err := toRGB(in, flags)
		if err != nil {
			return nil, err
		}
		vars := map[string]float64{"iw": float64(rgb.Width), "ih": float64(rgb.Height), "in_w": float64(rgb.Width), "in_h": float64(rgb.Height)}
		w, err := dim(args, 0, []string{"w", "width"}, rgb.Width, vars)
		if err != nil {
			return nil, err
		}
		h, err := dim(args, 1, []string{"h", "height"}, rgb.Height, vars)
		if err != nil {
			return nil, err
		}
		if w < rgb.Width {
			w = rgb.Width
		}
		if h < rgb.Height {
			h = rgb.Height
		}
		vars["ow"] = float64(w)
		vars["oh"] = float64(h)
		vars["out_w"] = float64(w)
		vars["out_h"] = float64(h)
		x := (w - rgb.Width) / 2
		y := (h - rgb.Height) / 2
		if text, ok := args.at(2, "x"); ok {
			x, err = evalInt(text, vars)
			if err != nil {
				return nil, err
			}
		}
		if text, ok := args.at(3, "y"); ok {
			y, err = evalInt(text, vars)
			if err != nil {
				return nil, err
			}
		}
		color := [4]byte{0, 0, 0, 255}
		if text, ok := args.at(4, "color"); ok {
			color, err = avutil.ParseColor(text)
			if err != nil {
				return nil, err
			}
		}
		dst, err := avutil.NewVideo(avutil.PixRGB24, w, h)
		if err != nil {
			return nil, err
		}
		fillRGB(dst, color)
		for yy := 0; yy < rgb.Height; yy++ {
			for xx := 0; xx < rgb.Width; xx++ {
				r, g, b := getRGB24(rgb, xx, yy)
				putRGB24(dst, x+xx, y+yy, r, g, b)
			}
		}
		copyMeta(dst, in)
		out, err := fromRGB(dst, in.Format, flags)
		if err != nil {
			return nil, err
		}
		return one(out), nil
	}
}

func newNegate(args argSet) func(*avutil.Frame) ([]*avutil.Frame, error) {
	alpha := false
	if text, ok := args.get("negate_alpha"); ok && (text == "1" || text == "true") {
		alpha = true
	}
	return func(in *avutil.Frame) ([]*avutil.Frame, error) {
		out, err := cloneVideo(in)
		if err != nil {
			return nil, err
		}
		negatePlanes(out, alpha)
		return one(out), nil
	}
}

func negatePlanes(frame *avutil.Frame, alpha bool) {
	item, ok := avutil.Desc(frame.Format)
	if !ok {
		return
	}
	if item.Planar {
		planes := item.Comp
		if item.Alpha && !alpha && planes > 1 {
			planes--
		}
		for plane := 0; plane < planes; plane++ {
			w, h := frame.Width, frame.Height
			if plane == 1 || plane == 2 {
				w = avutil.ChromaWidth(frame.Format, frame.Width)
				h = avutil.ChromaHeight(frame.Format, frame.Height)
			}
			for y := 0; y < h; y++ {
				row := frame.PlaneRow(plane, y, w)
				for i := range row {
					row[i] = 255 - row[i]
				}
			}
		}
		return
	}
	packed := item.Packed
	if packed <= 0 {
		packed = 1
	}
	for y := 0; y < frame.Height; y++ {
		row := frame.PlaneRow(0, y, frame.Width*packed)
		for x := 0; x < frame.Width; x++ {
			base := x * packed
			n := packed
			if item.Alpha && !alpha && n > 0 {
				n--
			}
			for i := 0; i < n && base+i < len(row); i++ {
				row[base+i] = 255 - row[base+i]
			}
		}
	}
}

func newFlip(vertical bool, flags swscale.Flags) func(*avutil.Frame) ([]*avutil.Frame, error) {
	return func(in *avutil.Frame) ([]*avutil.Frame, error) {
		rgb, err := toRGB(in, flags)
		if err != nil {
			return nil, err
		}
		dst, err := avutil.NewVideo(avutil.PixRGB24, rgb.Width, rgb.Height)
		if err != nil {
			return nil, err
		}
		for y := 0; y < rgb.Height; y++ {
			for x := 0; x < rgb.Width; x++ {
				sx, sy := x, y
				if vertical {
					sy = rgb.Height - 1 - y
				} else {
					sx = rgb.Width - 1 - x
				}
				r, g, b := getRGB24(rgb, sx, sy)
				putRGB24(dst, x, y, r, g, b)
			}
		}
		copyMeta(dst, in)
		out, err := fromRGB(dst, in.Format, flags)
		if err != nil {
			return nil, err
		}
		return one(out), nil
	}
}

func newTranspose(args argSet, flags swscale.Flags) func(*avutil.Frame) ([]*avutil.Frame, error) {
	dir := 0
	if text, ok := args.at(0, "dir"); ok {
		switch text {
		case "cclock_flip", "0":
			dir = 0
		case "clock", "1":
			dir = 1
		case "cclock", "2":
			dir = 2
		case "clock_flip", "3":
			dir = 3
		default:
			n, err := strconv.Atoi(text)
			if err != nil {
				return nil
			}
			dir = n & 3
		}
	}
	return func(in *avutil.Frame) ([]*avutil.Frame, error) {
		rgb, err := toRGB(in, flags)
		if err != nil {
			return nil, err
		}
		dst, err := avutil.NewVideo(avutil.PixRGB24, rgb.Height, rgb.Width)
		if err != nil {
			return nil, err
		}
		for y := 0; y < dst.Height; y++ {
			for x := 0; x < dst.Width; x++ {
				sx, sy := y, x
				if dir&1 != 0 {
					sy = rgb.Height - 1 - x
					sx = y
				}
				dx, dy := x, y
				if dir&2 != 0 {
					dy = dst.Height - 1 - y
				}
				r, g, b := getRGB24(rgb, sx, sy)
				putRGB24(dst, dx, dy, r, g, b)
			}
		}
		copyMeta(dst, in)
		if in.SAR.Num != 0 && in.SAR.Den != 0 {
			dst.SAR = in.SAR.Inv()
		}
		out, err := fromRGB(dst, in.Format, flags)
		if err != nil {
			return nil, err
		}
		return one(out), nil
	}
}

func newFPS(args argSet) func(*avutil.Frame) ([]*avutil.Frame, error) {
	var rate avutil.Rational
	var have bool
	n := 0
	return func(in *avutil.Frame) ([]*avutil.Frame, error) {
		if !have {
			text, ok := args.at(0, "fps")
			if !ok {
				text = "25"
			}
			var err error
			rate, err = avutil.ParseRate(text)
			if err != nil {
				return nil, err
			}
			have = true
		}
		inRate := avutil.Q(1, 1)
		if in.Duration > 0 && in.TimeBase.Den != 0 {
			inRate = avutil.Q(int32(in.TimeBase.Den), int32(in.TimeBase.Num)*int32(in.Duration))
		} else if in.TimeBase.Num != 0 {
			inRate = in.TimeBase.Inv()
		}
		emit := 1
		if inRate.Num > 0 && inRate.Den > 0 && rate.Num > 0 {
			next := int(avutil.Rescale(int64(n+1)*int64(rate.Num)*int64(inRate.Den), 1, int64(rate.Den)*int64(inRate.Num)))
			prev := int(avutil.Rescale(int64(n)*int64(rate.Num)*int64(inRate.Den), 1, int64(rate.Den)*int64(inRate.Num)))
			emit = next - prev
		}
		n++
		if emit < 1 {
			return nil, nil
		}
		var out []*avutil.Frame
		tb := rate.Inv()
		for i := 0; i < emit; i++ {
			frame, err := cloneVideo(in)
			if err != nil {
				return nil, err
			}
			frame.TimeBase = tb
			frame.Duration = 1
			frame.PTS = int64(len(out)) + int64(n-1)
			out = append(out, frame)
		}
		return out, nil
	}
}

func newTrim(audio bool, args argSet) func(*avutil.Frame) ([]*avutil.Frame, error) {
	start := 0.0
	end := math.Inf(1)
	if text, ok := args.get("start", "st"); ok {
		us, err := avutil.ParseTimeUS(text)
		if err == nil {
			start = float64(us) / 1e6
		}
	}
	if text, ok := args.get("end"); ok {
		us, err := avutil.ParseTimeUS(text)
		if err == nil {
			end = float64(us) / 1e6
		}
	}
	if text, ok := args.get("duration", "d"); ok {
		us, err := avutil.ParseTimeUS(text)
		if err == nil {
			end = start + float64(us)/1e6
		}
	}
	_ = audio
	return func(in *avutil.Frame) ([]*avutil.Frame, error) {
		t := videoTime(in)
		if t < start || t >= end {
			return nil, nil
		}
		if in.Width > 0 {
			frame, err := cloneVideo(in)
			if err != nil {
				return nil, err
			}
			return one(frame), nil
		}
		frame, err := cloneAudio(in)
		if err != nil {
			return nil, err
		}
		return one(frame), nil
	}
}

func newSetSAR(args argSet) func(*avutil.Frame) ([]*avutil.Frame, error) {
	return func(in *avutil.Frame) ([]*avutil.Frame, error) {
		frame, err := cloneVideo(in)
		if err != nil {
			return nil, err
		}
		text, ok := args.at(0, "sar", "r")
		if !ok {
			text = "1"
		}
		sar, err := avutil.ParseRate(text)
		if err != nil {
			return nil, err
		}
		frame.SAR = sar
		return one(frame), nil
	}
}

func newSetPTS(args argSet) func(*avutil.Frame) ([]*avutil.Frame, error) {
	text, ok := args.at(0, "expr")
	if !ok {
		text = "PTS"
	}
	var expr *avutil.Expr
	var start int64
	var seen bool
	n := 0
	return func(in *avutil.Frame) ([]*avutil.Frame, error) {
		if expr == nil {
			var err error
			expr, err = avutil.ParseExpr(text, "PTS", "STARTPTS", "N", "TB", "T")
			if err != nil {
				return nil, err
			}
		}
		if !seen {
			start = in.PTS
			seen = true
		}
		var frame *avutil.Frame
		var err error
		if in.Width > 0 {
			frame, err = cloneVideo(in)
		} else {
			frame, err = cloneAudio(in)
		}
		if err != nil {
			return nil, err
		}
		tb := in.TimeBase.Float64()
		val := expr.Eval(map[string]float64{
			"PTS": float64(in.PTS), "STARTPTS": float64(start), "N": float64(n),
			"TB": tb, "T": float64(in.PTS) * tb,
		})
		frame.PTS = int64(math.RoundToEven(val))
		n++
		return one(frame), nil
	}
}

func newNull() func(*avutil.Frame) ([]*avutil.Frame, error) {
	return func(in *avutil.Frame) ([]*avutil.Frame, error) {
		if in.Width > 0 {
			frame, err := cloneVideo(in)
			if err != nil {
				return nil, err
			}
			return one(frame), nil
		}
		frame, err := cloneAudio(in)
		if err != nil {
			return nil, err
		}
		return one(frame), nil
	}
}

func parseSWS(text string) swscale.Flags {
	var flags swscale.Flags
	for _, part := range strings.Split(text, "+") {
		switch strings.TrimSpace(part) {
		case "fast_bilinear":
			flags |= swscale.FastBilinear
		case "bilinear":
			flags |= swscale.Bilinear
		case "bicubic":
			flags |= swscale.Bicubic
		case "neighbor", "point":
			flags |= swscale.Point
		case "area":
			flags |= swscale.Area
		case "lanczos":
			flags |= swscale.Lanczos
		case "spline":
			flags |= swscale.Spline
		case "gauss", "gaussian":
			flags |= swscale.Gauss
		case "accurate_rnd":
			flags |= swscale.AccurateRnd
		}
	}
	return flags
}

func inW(f *avutil.Frame) float64 { return float64(f.Width) }
func inH(f *avutil.Frame) float64 {
	if f.Height == 0 {
		return 1
	}
	return float64(f.Height)
}
func sarOf(f *avutil.Frame) float64 {
	if f.SAR.Num == 0 || f.SAR.Den == 0 {
		return 1
	}
	return f.SAR.Float64()
}

func dim(args argSet, index int, keys []string, fallback int, vars map[string]float64) (int, error) {
	text, ok := args.at(index, keys...)
	if !ok {
		return fallback, nil
	}
	return evalInt(text, vars)
}

func evalInt(text string, vars map[string]float64) (int, error) {
	if n, err := strconv.Atoi(text); err == nil {
		return n, nil
	}
	expr, err := avutil.ParseExpr(text, "iw", "ih", "ow", "oh", "a", "sar", "dar", "in_w", "in_h", "out_w", "out_h", "x", "y")
	if err != nil {
		return 0, err
	}
	return int(math.RoundToEven(expr.Eval(vars))), nil
}

func fitScale(w, h, iw, ih int) (int, int) {
	if iw < 1 {
		iw = 1
	}
	if ih < 1 {
		ih = 1
	}
	ow, oh := w, h
	if ow == 0 {
		ow = iw
	}
	if oh == 0 {
		oh = ih
	}
	if ow < 0 && oh > 0 {
		ow = roundTo(iw*oh/ih, -w)
	} else if oh < 0 && ow > 0 {
		oh = roundTo(ih*ow/iw, -h)
	} else if ow < 0 && oh < 0 {
		ow, oh = iw, ih
	}
	if ow < 1 {
		ow = 1
	}
	if oh < 1 {
		oh = 1
	}
	return ow, oh
}

func roundTo(v, m int) int {
	if m <= 1 {
		if v < 1 {
			return 1
		}
		return v
	}
	v = (v + m/2) / m * m
	if v < m {
		return m
	}
	return v
}
