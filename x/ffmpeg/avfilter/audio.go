package avfilter

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
	"github.com/lewtec/lewkit/x/ffmpeg/swresample"
)

func newVolume(args argSet) func(*avutil.Frame) ([]*avutil.Frame, error) {
	text, ok := args.at(0, "volume")
	if !ok {
		text = "1"
	}
	gain := 1.0
	if strings.HasSuffix(text, "dB") {
		db, err := strconv.ParseFloat(strings.TrimSuffix(text, "dB"), 64)
		if err == nil {
			gain = math.Pow(10, db/20)
		}
	} else if v, err := strconv.ParseFloat(text, 64); err == nil {
		gain = v
	}
	return func(in *avutil.Frame) ([]*avutil.Frame, error) {
		out, err := cloneAudio(in)
		if err != nil {
			return nil, err
		}
		if gain == 1 {
			return one(out), nil
		}
		walkSamples(out, func(b []byte) {
			switch out.SampleFmt {
			case avutil.SampleU8, avutil.SampleU8P:
				v := (float64(int(b[0])-128) * gain) + 128
				b[0] = avutil.ClipUint8(int(math.RoundToEven(v)))
			case avutil.SampleS16, avutil.SampleS16P:
				v := float64(int16(binary.LittleEndian.Uint16(b))) * gain
				binary.LittleEndian.PutUint16(b, uint16(avutil.ClipInt16(int(math.RoundToEven(v)))))
			case avutil.SampleS32, avutil.SampleS32P:
				v := float64(int32(binary.LittleEndian.Uint32(b))) * gain
				binary.LittleEndian.PutUint32(b, uint32(clipS32(v)))
			case avutil.SampleFLT, avutil.SampleFLTP:
				v := math.Float32frombits(binary.LittleEndian.Uint32(b))
				binary.LittleEndian.PutUint32(b, math.Float32bits(float32(float64(v)*gain)))
			case avutil.SampleDBL, avutil.SampleDBLP:
				v := math.Float64frombits(binary.LittleEndian.Uint64(b))
				binary.LittleEndian.PutUint64(b, math.Float64bits(v*gain))
			}
		})
		return one(out), nil
	}
}

func newAresample(args argSet) func(*avutil.Frame) ([]*avutil.Frame, error) {
	return func(in *avutil.Frame) ([]*avutil.Frame, error) {
		rate := in.SampleRate
		format := in.SampleFmt
		layout := in.Layout
		if text, ok := args.at(0, "osr", "sample_rate", "r", "sample_rates"); ok {
			text = strings.Split(text, "|")[0]
			n, err := strconv.Atoi(text)
			if err != nil || n <= 0 {
				return nil, fmt.Errorf("%w: aresample %s", avutil.ErrOption, text)
			}
			rate = n
		}
		if text, ok := args.get("osf", "sample_fmt", "sample_fmts"); ok {
			var err error
			format, err = avutil.ParseSampleFormat(strings.Split(text, "|")[0])
			if err != nil {
				return nil, err
			}
		}
		if text, ok := args.get("ocl", "channel_layout", "channel_layouts"); ok {
			text = strings.Split(text, "|")[0]
			var err error
			layout, err = avutil.ParseLayout(text)
			if err != nil {
				return nil, err
			}
		}
		if rate == in.SampleRate && format == in.SampleFmt && layout == in.Layout {
			out, err := cloneAudio(in)
			if err != nil {
				return nil, err
			}
			return one(out), nil
		}
		sw, err := swresample.New(in.SampleRate, in.SampleFmt, in.Layout, rate, format, layout)
		if err != nil {
			return nil, err
		}
		outN := in.NbSamples
		if in.SampleRate != rate {
			outN = int(int64(in.NbSamples) * int64(rate) / int64(in.SampleRate))
			if outN < 1 && in.NbSamples > 0 {
				outN = 1
			}
		}
		dst, err := avutil.NewAudio(format, rate, layout, outN)
		if err != nil {
			return nil, err
		}
		if err := sw.Convert(dst, in); err != nil {
			return nil, err
		}
		dst.PTS = in.PTS
		if in.SampleRate > 0 {
			dst.PTS = avutil.Rescale(in.PTS, int64(rate), int64(in.SampleRate))
		}
		dst.Duration = int64(dst.NbSamples)
		dst.TimeBase = avutil.Q(1, int32(rate))
		return one(dst), nil
	}
}

func newAformat(args argSet) func(*avutil.Frame) ([]*avutil.Frame, error) {
	return newAresample(args)
}

func newPan(args argSet) func(*avutil.Frame) ([]*avutil.Frame, error) {
	return func(in *avutil.Frame) ([]*avutil.Frame, error) {
		text := args.raw
		if text == "" {
			out, err := cloneAudio(in)
			if err != nil {
				return nil, err
			}
			return one(out), nil
		}
		layoutText, rest, _ := strings.Cut(text, "|")
		layout, err := avutil.ParseLayout(layoutText)
		if err != nil {
			return nil, err
		}
		outCh := layout.Channels()
		inCh := in.Layout.Channels()
		matrix := make([][]float64, outCh)
		for i := range matrix {
			matrix[i] = make([]float64, inCh)
			if i < inCh {
				matrix[i][i] = 1
			}
		}
		if rest != "" {
			for i := range matrix {
				for j := range matrix[i] {
					matrix[i][j] = 0
				}
			}
			for _, part := range strings.Split(rest, "|") {
				left, expr, ok := strings.Cut(part, "=")
				if !ok || !strings.HasPrefix(left, "c") {
					return nil, fmt.Errorf("%w: pan %s", avutil.ErrOption, part)
				}
				dst, err := strconv.Atoi(left[1:])
				if err != nil || dst < 0 || dst >= outCh {
					return nil, fmt.Errorf("%w: pan %s", avutil.ErrOption, part)
				}
				if err := parseMix(expr, matrix[dst]); err != nil {
					return nil, err
				}
			}
		}
		dst, err := avutil.NewAudio(in.SampleFmt, in.SampleRate, layout, in.NbSamples)
		if err != nil {
			return nil, err
		}
		for i := 0; i < in.NbSamples; i++ {
			inV := make([]float64, inCh)
			for c := 0; c < inCh; c++ {
				inV[c] = loadAudio(in, i, c)
			}
			for c := 0; c < outCh; c++ {
				var sum float64
				for ic := 0; ic < inCh; ic++ {
					sum += matrix[c][ic] * inV[ic]
				}
				storeAudio(dst, i, c, sum)
			}
		}
		copyMeta(dst, in)
		dst.Layout = layout
		return one(dst), nil
	}
}

func parseMix(expr string, row []float64) error {
	expr = strings.ReplaceAll(expr, "-", "+-")
	if strings.HasPrefix(expr, "+") {
		expr = expr[1:]
	}
	for _, term := range strings.Split(expr, "+") {
		term = strings.TrimSpace(term)
		if term == "" {
			continue
		}
		coef := 1.0
		ch := term
		if left, right, ok := strings.Cut(term, "*"); ok {
			v, err := strconv.ParseFloat(left, 64)
			if err != nil {
				return fmt.Errorf("%w: pan %s", avutil.ErrOption, term)
			}
			coef = v
			ch = right
		}
		if !strings.HasPrefix(ch, "c") {
			return fmt.Errorf("%w: pan %s", avutil.ErrOption, term)
		}
		idx, err := strconv.Atoi(ch[1:])
		if err != nil || idx < 0 || idx >= len(row) {
			return fmt.Errorf("%w: pan %s", avutil.ErrOption, term)
		}
		row[idx] += coef
	}
	return nil
}

func walkSamples(frame *avutil.Frame, fn func([]byte)) {
	width := avutil.BytesPerSample(frame.SampleFmt)
	channels := frame.Layout.Channels()
	if width == 0 || channels <= 0 {
		return
	}
	if avutil.IsPlanarSample(frame.SampleFmt) {
		for c := 0; c < channels && c < 8; c++ {
			for i := 0; i < frame.NbSamples; i++ {
				fn(frame.Data[c][i*width : (i+1)*width])
			}
		}
		return
	}
	for i := 0; i < frame.NbSamples*channels; i++ {
		fn(frame.Data[0][i*width : (i+1)*width])
	}
}

func loadAudio(frame *avutil.Frame, index, ch int) float64 {
	b := sampleBytes(frame, index, ch)
	switch avutil.PackedSample(frame.SampleFmt) {
	case avutil.SampleU8:
		return float64(int(b[0])-128) / 128
	case avutil.SampleS16:
		return float64(int16(binary.LittleEndian.Uint16(b))) / 32768
	case avutil.SampleS32:
		return float64(int32(binary.LittleEndian.Uint32(b))) / 2147483648
	case avutil.SampleFLT:
		return float64(math.Float32frombits(binary.LittleEndian.Uint32(b)))
	case avutil.SampleDBL:
		return math.Float64frombits(binary.LittleEndian.Uint64(b))
	default:
		return 0
	}
}

func storeAudio(frame *avutil.Frame, index, ch int, v float64) {
	b := sampleBytes(frame, index, ch)
	switch avutil.PackedSample(frame.SampleFmt) {
	case avutil.SampleU8:
		b[0] = avutil.ClipUint8(int(math.RoundToEven(v*128 + 128)))
	case avutil.SampleS16:
		binary.LittleEndian.PutUint16(b, uint16(avutil.ClipInt16(int(math.RoundToEven(v*32768)))))
	case avutil.SampleS32:
		binary.LittleEndian.PutUint32(b, uint32(clipS32(v*2147483648)))
	case avutil.SampleFLT:
		binary.LittleEndian.PutUint32(b, math.Float32bits(float32(v)))
	case avutil.SampleDBL:
		binary.LittleEndian.PutUint64(b, math.Float64bits(v))
	}
}

func sampleBytes(frame *avutil.Frame, index, ch int) []byte {
	width := avutil.BytesPerSample(frame.SampleFmt)
	if avutil.IsPlanarSample(frame.SampleFmt) {
		return frame.Data[ch][index*width : (index+1)*width]
	}
	channels := frame.Layout.Channels()
	at := (index*channels + ch) * width
	return frame.Data[0][at : at+width]
}

func clipS32(v float64) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(math.RoundToEven(v))
}
