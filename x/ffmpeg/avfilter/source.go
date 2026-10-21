package avfilter

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strconv"
	"sync"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
	"github.com/lewtec/lewkit/x/ffmpeg/swscale"
)

const gradientSize = 6 * 256

type testSource struct {
	w, h     int
	tb       avutil.Rational
	sar      avutil.Rational
	duration int64
	decimals int
	pts      int64
	nb       int
	format   avutil.PixelFormat
	color    [4]byte
	cached   *avutil.Frame
	flags    swscale.Flags
	kind     string
}

func (s *testSource) Next() (*avutil.Frame, error) {
	if s.duration >= 0 && avutil.RescaleQ(s.pts, s.tb, avutil.Q(1, 1000000)) >= s.duration {
		return nil, io.EOF
	}
	var frame *avutil.Frame
	var err error
	switch s.kind {
	case "testsrc":
		frame, err = avutil.NewVideo(avutil.PixRGB24, s.w, s.h)
		if err != nil {
			return nil, err
		}
		fillTestSrc(frame, s.nb, s.tb, s.decimals)
	case "color":
		frame, err = s.colorFrame()
	default:
		format := s.format
		if format == 0 {
			format = avutil.PixYUV420P
		}
		frame, err = avutil.NewVideo(format, s.w, s.h)
	}
	if err != nil {
		return nil, err
	}
	frame.PTS = s.pts
	frame.Duration = 1
	frame.TimeBase = s.tb
	frame.SAR = s.sar
	frame.Key = true
	s.pts++
	s.nb++
	return frame, nil
}

func (s *testSource) colorFrame() (*avutil.Frame, error) {
	format := s.format
	if format == 0 {
		format = avutil.PixYUV420P
	}
	if s.cached != nil {
		return cloneVideo(s.cached)
	}
	rgb, err := avutil.NewVideo(avutil.PixRGB24, s.w, s.h)
	if err != nil {
		return nil, err
	}
	fillRGB(rgb, s.color)
	out, err := fromRGB(rgb, format, s.flags)
	if err != nil {
		return nil, err
	}
	s.cached = out
	return cloneVideo(out)
}

func newTestSrc(args argSet) (Link, error) {
	s, err := newPictureSource("testsrc", args)
	if err != nil {
		return nil, err
	}
	if text, ok := args.get("decimals", "n"); ok {
		n, err := strconv.Atoi(text)
		if err != nil || n < 0 || n > 17 {
			return nil, fmt.Errorf("%w: decimals %s", avutil.ErrOption, text)
		}
		s.decimals = n
	}
	return s, nil
}

func newNullSrc(args argSet) (Link, error) {
	s, err := newPictureSource("nullsrc", args)
	if err != nil {
		return nil, err
	}
	if text, ok := args.get("format", "pix_fmt"); ok {
		format, err := avutil.ParsePixelFormat(text)
		if err != nil {
			return nil, err
		}
		s.format = format
	}
	return s, nil
}

func newColor(args argSet, flags swscale.Flags) (Link, error) {
	s, err := newPictureSource("color", args)
	if err != nil {
		return nil, err
	}
	s.flags = flags
	s.color = [4]byte{0, 0, 0, 255}
	if text, ok := args.at(0, "color", "c"); ok {
		c, err := avutil.ParseColor(text)
		if err != nil {
			return nil, err
		}
		s.color = c
	}
	if text, ok := args.get("format", "pix_fmt"); ok {
		format, err := avutil.ParsePixelFormat(text)
		if err != nil {
			return nil, err
		}
		s.format = format
	}
	return s, nil
}

func newPictureSource(kind string, args argSet) (*testSource, error) {
	w, h := 320, 240
	if text, ok := args.at(0, "size", "s"); ok && kind != "color" || false {
		var err error
		w, h, err = avutil.ParseSize(text)
		if err != nil {
			return nil, err
		}
	}
	if text, ok := args.get("size", "s"); ok {
		var err error
		w, h, err = avutil.ParseSize(text)
		if err != nil {
			return nil, err
		}
	}
	rate := avutil.Q(25, 1)
	if text, ok := args.get("rate", "r"); ok {
		var err error
		rate, err = avutil.ParseRate(text)
		if err != nil {
			return nil, err
		}
	}
	duration := int64(-1)
	if text, ok := args.get("duration", "d"); ok {
		if text == "-1" {
			duration = -1
		} else {
			us, err := avutil.ParseTimeUS(text)
			if err != nil {
				return nil, err
			}
			duration = us
		}
	}
	sar := avutil.Q(1, 1)
	if text, ok := args.get("sar"); ok {
		r, err := avutil.ParseRate(text)
		if err != nil {
			return nil, err
		}
		sar = r
	}
	tb := rate.Inv()
	if tb.Num == 0 || tb.Den == 0 {
		return nil, fmt.Errorf("%w: rate", avutil.ErrOption)
	}
	return &testSource{kind: kind, w: w, h: h, tb: tb, sar: sar, duration: duration}, nil
}

func fillTestSrc(frame *avutil.Frame, nb int, tb avutil.Rational, decimals int) {
	width := frame.Width
	height := frame.Height
	stride := frame.Linesize[0]
	data := frame.Data[0]
	radius := (width + height) / 4
	quad0 := int32(width)*int32(width)/4 + int32(height)*int32(height)/4 - int32(radius)*int32(radius)
	dquadY := int32(1 - height)
	for y := 0; y < height; y++ {
		p := y * stride
		color := 0
		colorRest := 0
		quad := quad0
		dquadX := int32(1 - width)
		for x := 0; x < width; x++ {
			icolor := color
			if quad < 0 {
				icolor ^= 7
			}
			quad += dquadX
			dquadX += 2
			if icolor&1 != 0 {
				data[p] = 255
			} else {
				data[p] = 0
			}
			p++
			if icolor&2 != 0 {
				data[p] = 255
			} else {
				data[p] = 0
			}
			p++
			if icolor&4 != 0 {
				data[p] = 255
			} else {
				data[p] = 0
			}
			p++
			colorRest += 8
			if colorRest >= width {
				colorRest -= width
				color++
			}
		}
		quad0 += dquadY
		dquadY += 2
	}
	base := (height * 3 / 4) * stride
	grad := (256 * nb * int(tb.Num) / int(tb.Den)) % gradientSize
	rgrad := 0
	dgrad := gradientSize / width
	drgrad := gradientSize % width
	p := base
	for x := 0; x < width; x++ {
		data[p] = gradByte(grad, 0)
		data[p+1] = gradByte(grad, 1)
		data[p+2] = gradByte(grad, 2)
		p += 3
		grad += dgrad
		rgrad += drgrad
		if rgrad >= gradientSize {
			grad++
			rgrad -= gradientSize
		}
		if grad >= gradientSize {
			grad -= gradientSize
		}
	}
	p = base
	for y := height / 8; y > 0; y-- {
		copy(data[p+stride:p+stride+3*width], data[p:p+3*width])
		p += stride
	}
	seg := width / 80
	if seg < 1 || height < 13*seg {
		return
	}
	pow := math.Pow10(decimals)
	if tb.Float64()*float64(nb)*pow >= math.MaxInt32 {
		return
	}
	p10 := int64(1)
	for i := 0; i < decimals; i++ {
		p10 *= 10
	}
	second := int(avutil.RescaleRnd(int64(nb)*int64(tb.Num), p10, int64(tb.Den), avutil.RoundZero))
	x := width - (width-seg*64)/2
	y := (height - seg*13) / 2
	at := x*3 + y*stride
	for i := 0; i < 8; i++ {
		at -= 3 * 8 * seg
		drawDigit(second%10, data[at:], stride, seg)
		second /= 10
		if second == 0 {
			break
		}
	}
}

func gradByte(grad, ch int) byte {
	switch ch {
	case 0:
		switch {
		case grad < 256 || grad >= 5*256:
			return 255
		case grad >= 2*256 && grad < 4*256:
			return 0
		case grad < 2*256:
			return byte(2*256 - 1 - grad)
		default:
			return byte(grad - 4*256)
		}
	case 1:
		switch {
		case grad >= 4*256:
			return 0
		case grad >= 256 && grad < 3*256:
			return 255
		case grad < 256:
			return byte(grad)
		default:
			return byte(4*256 - 1 - grad)
		}
	default:
		switch {
		case grad < 2*256:
			return 0
		case grad >= 3*256 && grad < 5*256:
			return 255
		case grad < 3*256:
			return byte(grad - 2*256)
		default:
			return byte(6*256 - 1 - grad)
		}
	}
}

func drawDigit(digit int, dst []byte, linesize, seg int) {
	type rect struct{ x, y, w, h int }
	segs := []rect{
		{1, 0, 5, 1}, {1, 6, 5, 1}, {1, 12, 5, 1},
		{0, 1, 1, 5}, {0, 7, 1, 5}, {6, 1, 1, 5}, {6, 7, 1, 5},
	}
	const (
		top = 1 << 0
		mid = 1 << 1
		bot = 1 << 2
		lt  = 1 << 3
		lb  = 1 << 4
		rt  = 1 << 5
		rb  = 1 << 6
	)
	masks := [10]byte{
		top | bot | lt | lb | rt | rb,
		rt | rb,
		top | mid | bot | lb | rt,
		top | mid | bot | rt | rb,
		mid | lt | rt | rb,
		top | bot | mid | lt | rb,
		top | bot | mid | lt | lb | rb,
		top | rt | rb,
		top | bot | mid | lt | lb | rt | rb,
		top | bot | mid | lt | rt | rb,
	}
	drawRect(0, dst, linesize, seg, 0, 0, 8, 13)
	mask := masks[digit]
	for i, r := range segs {
		if mask&(1<<uint(i)) != 0 {
			drawRect(255, dst, linesize, seg, r.x, r.y, r.w, r.h)
		}
	}
}

func drawRect(val byte, dst []byte, linesize, seg, x, y, w, h int) {
	off := seg * (3*x + y*linesize)
	bw := w * seg * 3
	bh := h * seg
	for i := 0; i < bh; i++ {
		for j := 0; j < bw; j++ {
			dst[off+j] = val
		}
		off += linesize
	}
}

const (
	sineLogPeriod = 15
	sineAmplitude = 4095
	sineShift     = 3
)

var sineSamples = sync.OnceValue(makeSineTable)

func makeSineTable() []int16 {
	halfPi := 1 << (sineLogPeriod - 2)
	table := make([]int16, 1<<sineLogPeriod)
	ampls := uint32(sineAmplitude << sineShift)
	unit2 := uint64(ampls*ampls) << 32
	table[0] = 0
	table[halfPi] = int16(ampls)
	for step := halfPi; step > 1; step /= 2 {
		k := uint32(0x10000)
		for i := 0; i < halfPi/2; i += step {
			s := uint32(int32(table[i]) + int32(table[i+step]))
			c := uint32(int32(table[halfPi-i]) + int32(table[halfPi-i-step]))
			n2 := s*s + c*c
			for {
				div := uint64(k) * uint64(n2)
				newK := uint32((uint64(k) + unit2/div + 1) >> 1)
				if k == newK {
					break
				}
				k = newK
			}
			table[i+step/2] = int16((k*s + 0x7FFF) >> 16)
			table[halfPi-i-step/2] = int16((k*c + 0x8000) >> 16)
		}
	}
	for i := 0; i <= halfPi; i++ {
		table[i] = int16((int32(table[i]) + (1 << (sineShift - 1))) >> sineShift)
	}
	for i := 0; i < halfPi; i++ {
		table[halfPi*2-i] = table[i]
	}
	for i := 0; i < 2*halfPi; i++ {
		table[i+2*halfPi] = -table[i]
	}
	return table
}

type sineSource struct {
	table    []int16
	rate     int
	freq     float64
	beep     float64
	duration int64
	expr     *avutil.Expr
	phi      uint32
	dphi     uint32
	phiBeep  uint32
	dphiBeep uint32
	beepLen  uint32
	beepPer  uint32
	beepIdx  uint32
	pts      int64
	n        int64
}

func newSine(args argSet) (Link, error) {
	rate := 44100
	if text, ok := args.get("sample_rate", "r"); ok {
		n, err := strconv.Atoi(text)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("%w: sample_rate %s", avutil.ErrOption, text)
		}
		rate = n
	}
	freq := 440.0
	if text, ok := args.get("frequency", "f"); ok {
		v, err := strconv.ParseFloat(text, 64)
		if err != nil || v < 0 {
			return nil, fmt.Errorf("%w: frequency %s", avutil.ErrOption, text)
		}
		freq = v
	}
	beep := 0.0
	if text, ok := args.get("beep_factor", "b"); ok {
		v, err := strconv.ParseFloat(text, 64)
		if err != nil || v < 0 {
			return nil, fmt.Errorf("%w: beep_factor %s", avutil.ErrOption, text)
		}
		beep = v
	}
	var duration int64
	if text, ok := args.get("duration", "d"); ok && text != "0" {
		us, err := avutil.ParseTimeUS(text)
		if err != nil {
			return nil, err
		}
		duration = avutil.Rescale(us, int64(rate), 1000000)
	}
	spec := "1024"
	if text, ok := args.get("samples_per_frame"); ok {
		spec = text
	}
	expr, err := avutil.ParseExpr(spec, "n", "pts", "t", "TB")
	if err != nil {
		return nil, err
	}
	s := &sineSource{
		table: sineSamples(), rate: rate, freq: freq, beep: beep,
		duration: duration, expr: expr,
		dphi: uint32(math.Ldexp(freq, 32)/float64(rate) + 0.5),
	}
	if beep != 0 {
		s.beepPer = uint32(rate)
		s.beepLen = s.beepPer / 25
		s.dphiBeep = uint32(math.Ldexp(beep*freq, 32)/float64(rate) + 0.5)
	}
	return s, nil
}

func (s *sineSource) Next() (*avutil.Frame, error) {
	tb := avutil.Q(1, int32(s.rate))
	nb := int(math.RoundToEven(s.expr.Eval(map[string]float64{
		"n":   float64(s.n),
		"pts": float64(s.pts),
		"t":   float64(s.pts) * tb.Float64(),
		"TB":  tb.Float64(),
	})))
	if nb <= 0 {
		nb = 1024
	}
	if s.duration > 0 {
		left := s.duration - s.pts
		if left <= 0 {
			return nil, io.EOF
		}
		if int64(nb) > left {
			nb = int(left)
		}
	}
	frame, err := avutil.NewAudio(avutil.SampleS16, s.rate, avutil.Mono(), nb)
	if err != nil {
		return nil, err
	}
	for i := 0; i < nb; i++ {
		sample := int32(s.table[s.phi>>(32-sineLogPeriod)])
		s.phi += s.dphi
		if s.beep != 0 && s.beepIdx < s.beepLen {
			sample += int32(s.table[s.phiBeep>>(32-sineLogPeriod)]) * 2
			s.phiBeep += s.dphiBeep
		}
		if s.beepPer != 0 {
			s.beepIdx++
			if s.beepIdx == s.beepPer {
				s.beepIdx = 0
			}
		}
		binary.LittleEndian.PutUint16(frame.Data[0][i*2:], uint16(int16(sample)))
	}
	frame.PTS = s.pts
	frame.Duration = int64(nb)
	frame.TimeBase = tb
	s.pts += int64(nb)
	s.n++
	return frame, nil
}

type silence struct {
	rate     int
	layout   avutil.ChannelLayout
	format   avutil.SampleFormat
	count    int
	duration int64
	pts      int64
}

func newAnull(args argSet) (Link, error) {
	s := &silence{
		rate: 44100, layout: avutil.Stereo(), format: avutil.SampleS16, count: 1024,
	}
	if text, ok := args.get("sample_rate", "r"); ok {
		n, err := strconv.Atoi(text)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("%w: sample_rate %s", avutil.ErrOption, text)
		}
		s.rate = n
	}
	if text, ok := args.get("channel_layout", "cl"); ok {
		layout, err := avutil.ParseLayout(text)
		if err != nil {
			return nil, err
		}
		s.layout = layout
	}
	if text, ok := args.get("nb_samples", "n"); ok {
		n, err := strconv.Atoi(text)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("%w: nb_samples %s", avutil.ErrOption, text)
		}
		s.count = n
	}
	if text, ok := args.get("sample_fmt"); ok {
		format, err := avutil.ParseSampleFormat(text)
		if err != nil {
			return nil, err
		}
		s.format = format
	}
	if text, ok := args.get("duration", "d"); ok && text != "0" && text != "-1" {
		us, err := avutil.ParseTimeUS(text)
		if err != nil {
			return nil, err
		}
		s.duration = avutil.Rescale(us, int64(s.rate), 1000000)
	}
	return s, nil
}

func (s *silence) Next() (*avutil.Frame, error) {
	n := s.count
	if s.duration > 0 {
		left := s.duration - s.pts
		if left <= 0 {
			return nil, io.EOF
		}
		if int64(n) > left {
			n = int(left)
		}
	}
	frame, err := avutil.NewAudio(s.format, s.rate, s.layout, n)
	if err != nil {
		return nil, err
	}
	frame.PTS = s.pts
	frame.Duration = int64(n)
	frame.TimeBase = avutil.Q(1, int32(s.rate))
	s.pts += int64(n)
	return frame, nil
}
