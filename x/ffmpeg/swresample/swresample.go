package swresample

import (
	"fmt"
	"math"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// Context converts one PCM layout into another.
type Context struct {
	InRate, OutRate     int
	InFmt, OutFmt       avutil.SampleFormat
	InLayout, OutLayout avutil.ChannelLayout
}

// New builds a converter. Rates must be positive.
func New(inRate int, inFmt avutil.SampleFormat, inLayout avutil.ChannelLayout, outRate int, outFmt avutil.SampleFormat, outLayout avutil.ChannelLayout) (*Context, error) {
	if inRate <= 0 || outRate <= 0 || inLayout.Channels() <= 0 || outLayout.Channels() <= 0 {
		return nil, fmt.Errorf("%w: resample", avutil.ErrRange)
	}
	if avutil.BytesPerSample(inFmt) == 0 || avutil.BytesPerSample(outFmt) == 0 {
		return nil, fmt.Errorf("%w: sample format", avutil.ErrUnsupported)
	}
	return &Context{
		InRate: inRate, OutRate: outRate,
		InFmt: inFmt, OutFmt: outFmt,
		InLayout: inLayout, OutLayout: outLayout,
	}, nil
}

// Convert writes src into dst. dst.NbSamples is the output length.
// When the rate is unchanged, one input sample becomes one output sample.
func (c *Context) Convert(dst, src *avutil.Frame) error {
	if c == nil || dst == nil || src == nil || src.NbSamples < 0 {
		return fmt.Errorf("%w: resample", avutil.ErrInvalidData)
	}
	inCh := c.InLayout.Channels()
	outCh := c.OutLayout.Channels()
	outN := src.NbSamples
	if c.InRate != c.OutRate {
		if src.NbSamples == 0 {
			outN = 0
		} else {
			outN = int(int64(src.NbSamples)*int64(c.OutRate)/int64(c.InRate))
			if outN < 1 {
				outN = 1
			}
		}
	}
	if dst.NbSamples < outN || dst.SampleFmt != c.OutFmt {
		return fmt.Errorf("%w: resample destination", avutil.ErrInvalidData)
	}
	if c.InRate == c.OutRate && inCh == outCh {
		return c.direct(dst, src, outN, inCh)
	}
	return c.remix(dst, src, outN, inCh, outCh)
}

func (c *Context) direct(dst, src *avutil.Frame, n, channels int) error {
	for ch := 0; ch < channels; ch++ {
		sb := plane(src, ch)
		db := plane(dst, ch)
		for i := 0; i < n; i++ {
			convertOne(c.InFmt, c.OutFmt, sampleAt(c.InFmt, sb, i, channels, ch), sampleAt(c.OutFmt, db, i, channels, ch))
		}
	}
	dst.NbSamples = n
	return nil
}

func (c *Context) remix(dst, src *avutil.Frame, outN, inCh, outCh int) error {
	in := make([][]float64, inCh)
	for ch := 0; ch < inCh; ch++ {
		in[ch] = make([]float64, src.NbSamples)
		sb := plane(src, ch)
		for i := 0; i < src.NbSamples; i++ {
			in[ch][i] = loadFloat(c.InFmt, sampleAt(c.InFmt, sb, i, inCh, ch))
		}
	}
	mixed := make([][]float64, outCh)
	for ch := 0; ch < outCh; ch++ {
		mixed[ch] = make([]float64, src.NbSamples)
	}
	matrix := downmix(inCh, outCh)
	for i := 0; i < src.NbSamples; i++ {
		for oc := 0; oc < outCh; oc++ {
			var sum float64
			for ic := 0; ic < inCh; ic++ {
				sum += matrix[oc][ic] * in[ic][i]
			}
			mixed[oc][i] = sum
		}
	}
	resampled := mixed
	if c.InRate != c.OutRate {
		resampled = make([][]float64, outCh)
		for ch := 0; ch < outCh; ch++ {
			resampled[ch] = resampleLinear(mixed[ch], outN)
		}
	}
	for ch := 0; ch < outCh; ch++ {
		db := plane(dst, ch)
		for i := 0; i < outN; i++ {
			tmp := [8]byte{}
			storeFloat(c.OutFmt, tmp[:], resampled[ch][i])
			copy(sampleAt(c.OutFmt, db, i, outCh, ch), tmp[:avutil.BytesPerSample(c.OutFmt)])
		}
	}
	dst.NbSamples = outN
	return nil
}

func downmix(inCh, outCh int) [][]float64 {
	m := make([][]float64, outCh)
	for i := range m {
		m[i] = make([]float64, inCh)
	}
	if inCh == outCh {
		for i := 0; i < inCh; i++ {
			m[i][i] = 1
		}
		return m
	}
	if outCh == 1 {
		g := 1 / float64(inCh)
		for i := 0; i < inCh; i++ {
			m[0][i] = g
		}
		return m
	}
	if inCh == 1 {
		for i := 0; i < outCh; i++ {
			m[i][0] = 1
		}
		return m
	}
	n := inCh
	if outCh < n {
		n = outCh
	}
	for i := 0; i < n; i++ {
		m[i][i] = 1
	}
	return m
}

func resampleLinear(in []float64, outN int) []float64 {
	out := make([]float64, outN)
	if len(in) == 0 || outN == 0 {
		return out
	}
	if len(in) == 1 || outN == 1 {
		for i := range out {
			out[i] = in[0]
		}
		return out
	}
	scale := float64(len(in)-1) / float64(outN-1)
	for i := 0; i < outN; i++ {
		pos := float64(i) * scale
		j := int(math.Floor(pos))
		if j >= len(in)-1 {
			out[i] = in[len(in)-1]
			continue
		}
		f := pos - float64(j)
		out[i] = in[j]*(1-f) + in[j+1]*f
	}
	return out
}
