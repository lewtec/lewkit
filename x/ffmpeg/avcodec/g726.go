package avcodec

import (
	"encoding/binary"
	"fmt"
	"math/bits"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// g726Codec encodes and decodes mono G.726. Code size follows -b, default 4.
type g726Codec struct {
	pcm  []int16
	code int
	rate int
	le   bool
}

func newG726(par Parameters) *g726Codec {
	rate := par.SampleRate
	if rate <= 0 {
		rate = 8000
	}
	return &g726Codec{code: g726Code(rate, par.BitRate), rate: rate}
}

func newG726LE(par Parameters) *g726Codec {
	c := newG726(par)
	c.le = true
	return c
}

func (c *g726Codec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil || frame.NbSamples == 0 {
		return nil, nil
	}
	if frame.SampleFmt != avutil.SampleS16 || frame.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: g726", avutil.ErrInvalidData)
	}
	if len(frame.Data) == 0 || len(frame.Data[0]) < frame.NbSamples*2 {
		return nil, fmt.Errorf("%w: g726 samples", avutil.ErrInvalidData)
	}
	for i := 0; i < frame.NbSamples; i++ {
		c.pcm = append(c.pcm, int16(binary.LittleEndian.Uint16(frame.Data[0][i*2:])))
	}
	return nil, nil
}

func (c *g726Codec) Flush() ([]*avutil.Packet, error) {
	n := len(c.pcm)
	var raw []byte
	var err error
	if c.le {
		raw, err = G726LEEncode(c.pcm, c.code)
	} else {
		raw, err = G726Encode(c.pcm, c.code)
	}
	c.pcm = nil
	if err != nil || len(raw) == 0 {
		return nil, err
	}
	return []*avutil.Packet{{Data: raw, Duration: int64(n)}}, nil
}

func (c *g726Codec) Decode(pkt *avutil.Packet) ([]*avutil.Frame, error) {
	if pkt == nil || len(pkt.Data) == 0 {
		return nil, nil
	}
	var samples []int16
	var err error
	if c.le {
		samples, err = G726LEDecode(pkt.Data, c.code)
	} else {
		samples, err = G726Decode(pkt.Data, c.code)
	}
	if err != nil || len(samples) == 0 {
		return nil, err
	}
	rate := c.rate
	if rate <= 0 {
		rate = 8000
	}
	out, err := avutil.NewAudio(avutil.SampleS16, rate, avutil.Mono(), len(samples))
	if err != nil {
		return nil, err
	}
	for i, s := range samples {
		binary.LittleEndian.PutUint16(out.Data[0][i*2:], uint16(s))
	}
	out.PTS = pkt.PTS
	out.Duration = int64(len(samples))
	out.TimeBase = avutil.Q(1, int32(rate))
	return []*avutil.Frame{out}, nil
}

// G.726 at 16, 24, 32, or 40 kbit/s. The default is 4 bits per sample.
// The bitstream is the big-endian packing ffmpeg names g726.

type g726Float struct {
	sign uint8
	exp  uint8
	mant uint8
}

type g726Tables struct {
	quant  []int
	iquant []int16
	w      []int16
	f      []uint8
}

type g726State struct {
	tbls g726Tables
	sr   [2]g726Float
	dq   [6]g726Float
	a    [2]int
	b    [6]int
	pk   [2]int
	ap   int
	yu   int
	yl   int
	dms  int
	dml  int
	td   int
	se   int
	sez  int
	y    int
	code int
}

func g726Log2_16(v int) int {
	v &= 0xffff
	if v == 0 {
		return 0
	}
	return bits.Len(uint(v)) - 1
}

func g726I2F(i int, f *g726Float) {
	f.sign = 0
	if i < 0 {
		f.sign = 1
		i = -i
	}
	f.exp = uint8(g726Log2_16(i))
	if i != 0 {
		f.exp++
	}
	if i != 0 {
		f.mant = uint8((i << 6) >> f.exp)
	} else {
		f.mant = 1 << 5
	}
}

func g726Mult(f1, f2 g726Float) int16 {
	exp := int(f1.exp) + int(f2.exp)
	res := ((int(f1.mant) * int(f2.mant)) + 0x30) >> 4
	if exp > 19 {
		res <<= exp - 19
	} else {
		res >>= 19 - exp
	}
	if f1.sign^f2.sign != 0 {
		res = -res
	}
	return int16(res)
}

func g726Sgn(v int) int {
	if v < 0 {
		return -1
	}
	return 1
}

func g726Clip(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func g726ClipP2(v, p int) int {
	lim := 1 << p
	if v < -lim || v >= lim {
		if v < 0 {
			return -lim
		}
		return lim - 1
	}
	return v
}

var g726Pool = []g726Tables{
	{
		[]int{260, int(^uint(0) >> 1)},
		[]int16{116, 365, 365, 116},
		[]int16{-22, 439, 439, -22},
		[]uint8{0, 7, 7, 0},
	},
	{
		[]int{7, 217, 330, int(^uint(0) >> 1)},
		[]int16{-32768, 135, 273, 373, 373, 273, 135, -32768},
		[]int16{-4, 30, 137, 582, 582, 137, 30, -4},
		[]uint8{0, 1, 2, 7, 7, 2, 1, 0},
	},
	{
		[]int{-125, 79, 177, 245, 299, 348, 399, int(^uint(0) >> 1)},
		[]int16{-32768, 4, 135, 213, 273, 323, 373, 425, 425, 373, 323, 273, 213, 135, 4, -32768},
		[]int16{-12, 18, 41, 64, 112, 198, 355, 1122, 1122, 355, 198, 112, 64, 41, 18, -12},
		[]uint8{0, 0, 0, 1, 1, 1, 3, 7, 7, 3, 1, 1, 1, 0, 0, 0},
	},
	{
		[]int{-122, -16, 67, 138, 197, 249, 297, 338, 377, 412, 444, 474, 501, 527, 552, int(^uint(0) >> 1)},
		[]int16{-32768, -66, 28, 104, 169, 224, 274, 318, 358, 395, 429, 459, 488, 514, 539, 566, 566, 539, 514, 488, 459, 429, 395, 358, 318, 274, 224, 169, 104, 28, -66, -32768},
		[]int16{14, 14, 24, 39, 40, 41, 58, 100, 141, 179, 219, 280, 358, 440, 529, 696, 696, 529, 440, 358, 280, 219, 179, 141, 100, 58, 41, 40, 39, 24, 14, 14},
		[]uint8{0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 2, 3, 4, 5, 6, 6, 6, 6, 5, 4, 3, 2, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0},
	},
}

func (c *g726State) reset(code int) {
	*c = g726State{code: code, tbls: g726Pool[code-2]}
	for i := 0; i < 2; i++ {
		c.sr[i].mant = 1 << 5
		c.pk[i] = 1
	}
	for i := 0; i < 6; i++ {
		c.dq[i].mant = 1 << 5
	}
	c.yu = 544
	c.yl = 34816
	c.y = 544
}

func (c *g726State) quant(d int) int {
	sign := 0
	if d < 0 {
		sign = 1
		d = -d
	}
	exp := g726Log2_16(d)
	dln := (exp << 7) + (((d << 7) >> exp) & 0x7f) - (c.y >> 2)
	i := 0
	for c.tbls.quant[i] < int(^uint(0)>>1) && c.tbls.quant[i] < dln {
		i++
	}
	if sign != 0 {
		i = ^i
	}
	if c.code != 2 && i == 0 {
		i = 0xff
	}
	return i
}

func (c *g726State) inverse(i int) int16 {
	dql := int(c.tbls.iquant[i]) + (c.y >> 2)
	dex := (dql >> 7) & 0xf
	dqt := (1 << 7) + (dql & 0x7f)
	if dql < 0 {
		return 0
	}
	return int16((dqt << dex) >> 7)
}

func (c *g726State) decode(i int) int16 {
	isig := i >> (c.code - 1)
	dq := int(c.inverse(i))
	ylint := c.yl >> 15
	ylfrac := (c.yl >> 10) & 0x1f
	thr2 := (0x20 + ylfrac) << ylint
	if ylint > 9 {
		thr2 = 0x1f << 10
	}
	tr := c.td == 1 && dq > ((3*thr2)>>2)
	if isig != 0 {
		dq = -dq
	}
	re := int16(c.se + dq)
	pk0 := 0
	if c.sez+dq != 0 {
		pk0 = g726Sgn(c.sez + dq)
	}
	dq0 := 0
	if dq != 0 {
		dq0 = g726Sgn(dq)
	}
	if tr {
		c.a[0], c.a[1] = 0, 0
		for i := 0; i < 6; i++ {
			c.b[i] = 0
		}
	} else {
		fa1 := g726ClipP2(((-c.a[0] * c.pk[0] * pk0) >> 5), 8)
		c.a[1] += 128*pk0*c.pk[1] + fa1 - (c.a[1] >> 7)
		c.a[1] = g726Clip(c.a[1], -12288, 12288)
		c.a[0] += 64*3*pk0*c.pk[0] - (c.a[0] >> 8)
		c.a[0] = g726Clip(c.a[0], -(15360 - c.a[1]), 15360-c.a[1])
		for i := 0; i < 6; i++ {
			sign := 1
			if c.dq[i].sign != 0 {
				sign = -1
			}
			c.b[i] += 128*dq0*sign - (c.b[i] >> 8)
		}
	}
	c.pk[1] = c.pk[0]
	if pk0 != 0 {
		c.pk[0] = pk0
	} else {
		c.pk[0] = 1
	}
	c.sr[1] = c.sr[0]
	g726I2F(int(re), &c.sr[0])
	for i := 5; i > 0; i-- {
		c.dq[i] = c.dq[i-1]
	}
	g726I2F(dq, &c.dq[0])
	c.dq[0].sign = uint8(isig)
	if c.a[1] < -11776 {
		c.td = 1
	} else {
		c.td = 0
	}
	c.dms += (int(c.tbls.f[i]) << 4) + ((-c.dms) >> 5)
	c.dml += (int(c.tbls.f[i]) << 4) + ((-c.dml) >> 7)
	if tr {
		c.ap = 256
	} else {
		c.ap += (-c.ap) >> 4
		if c.y <= 1535 || c.td != 0 || absInt((c.dms<<2)-c.dml) >= (c.dml>>3) {
			c.ap += 0x20
		}
	}
	c.yu = g726Clip(c.y+int(c.tbls.w[i])+((-c.y)>>5), 544, 5120)
	c.yl += c.yu + ((-c.yl) >> 6)
	al := c.ap >> 2
	if c.ap >= 256 {
		al = 1 << 6
	}
	c.y = (c.yl + (c.yu-(c.yl>>6))*al) >> 6
	c.se = 0
	var f g726Float
	for n := 0; n < 6; n++ {
		c.se += int(g726Mult(func() g726Float { g726I2F(c.b[n]>>2, &f); return f }(), c.dq[n]))
	}
	c.sez = c.se >> 1
	for n := 0; n < 2; n++ {
		c.se += int(g726Mult(func() g726Float { g726I2F(c.a[n]>>2, &f); return f }(), c.sr[n]))
	}
	c.se >>= 1
	out := int(re) * 4
	if out < -0xffff {
		out = -0xffff
	} else if out > 0xffff {
		out = 0xffff
	}
	return int16(out)
}

func (c *g726State) encodeSample(sig int16) int {
	i := c.quant(int(sig)/4-c.se) & ((1 << c.code) - 1)
	c.decode(i)
	return i
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

type g726LE struct {
	b   []byte
	acc uint32
	n   int
}

func (w *g726LE) u(v uint32, n int) {
	for k := 0; k < n; k++ {
		w.acc |= ((v >> k) & 1) << w.n
		w.n++
		if w.n == 8 {
			w.b = append(w.b, byte(w.acc))
			w.acc = 0
			w.n = 0
		}
	}
}

func (w *g726LE) bytes() []byte {
	if w.n > 0 {
		w.b = append(w.b, byte(w.acc))
		w.acc = 0
		w.n = 0
	}
	return w.b
}

type g726LER struct {
	b []byte
	i int
}

func (r *g726LER) u(n int) (uint32, error) {
	if n < 0 || r.i+n > len(r.b)*8 {
		return 0, fmt.Errorf("%w: g726le bits", avutil.ErrInvalidData)
	}
	var v uint32
	for k := 0; k < n; k++ {
		bit := (r.b[r.i/8] >> (r.i % 8)) & 1
		v |= uint32(bit) << k
		r.i++
	}
	return v, nil
}

// G726LEEncode packs mono s16 with little-endian G.726 codes.
func G726LEEncode(pcm []int16, code int) ([]byte, error) {
	if code == 0 {
		code = 4
	}
	if code < 2 || code > 5 {
		return nil, fmt.Errorf("%w: g726le code %d", avutil.ErrInvalidData, code)
	}
	var s g726State
	s.reset(code)
	var w g726LE
	for _, sample := range pcm {
		w.u(uint32(s.encodeSample(sample)), code)
	}
	return w.bytes(), nil
}

// G726LEDecode expands a little-endian G.726 bitstream.
func G726LEDecode(data []byte, code int) ([]int16, error) {
	if code == 0 {
		code = 4
	}
	if code < 2 || code > 5 {
		return nil, fmt.Errorf("%w: g726le code %d", avutil.ErrInvalidData, code)
	}
	var s g726State
	s.reset(code)
	r := g726LER{b: data}
	n := len(data) * 8 / code
	out := make([]int16, n)
	for i := 0; i < n; i++ {
		v, err := r.u(code)
		if err != nil {
			return nil, err
		}
		out[i] = s.decode(int(v))
	}
	return out, nil
}

// G726Encode packs mono s16 into a G.726 bitstream.
// code is 2, 3, 4, or 5 bits per sample. 0 selects 4.
func G726Encode(pcm []int16, code int) ([]byte, error) {
	if code == 0 {
		code = 4
	}
	if code < 2 || code > 5 {
		return nil, fmt.Errorf("%w: g726 code %d", avutil.ErrInvalidData, code)
	}
	var s g726State
	s.reset(code)
	var w bitw
	for _, sample := range pcm {
		w.u(uint32(s.encodeSample(sample)), code)
	}
	return w.bytes(), nil
}

// G726Decode expands a big-endian G.726 bitstream.
func G726Decode(data []byte, code int) ([]int16, error) {
	if code == 0 {
		code = 4
	}
	if code < 2 || code > 5 {
		return nil, fmt.Errorf("%w: g726 code %d", avutil.ErrInvalidData, code)
	}
	var s g726State
	s.reset(code)
	r := bitr{b: data}
	n := len(data) * 8 / code
	out := make([]int16, n)
	for i := 0; i < n; i++ {
		v, err := r.u(code)
		if err != nil {
			return nil, err
		}
		out[i] = s.decode(int(v))
	}
	return out, nil
}

func g726Code(rate, bitRate int) int {
	code := 4
	if bitRate > 0 && rate > 0 {
		code = (bitRate + rate/2) / rate
	}
	if code < 2 {
		code = 2
	} else if code > 5 {
		code = 5
	}
	return code
}
