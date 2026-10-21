package avcodec

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// g722Codec is the 64 kbit/s G.722 encoder and decoder, trellis off.
type g722Codec struct {
	pcm []int16
}

func newG722Codec(Parameters) *g722Codec { return &g722Codec{} }

func (c *g722Codec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil || frame.NbSamples == 0 {
		return nil, nil
	}
	if frame.SampleFmt != avutil.SampleS16 || frame.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: g722", avutil.ErrInvalidData)
	}
	if len(frame.Data) == 0 || len(frame.Data[0]) < frame.NbSamples*2 {
		return nil, fmt.Errorf("%w: g722 samples", avutil.ErrInvalidData)
	}
	for i := 0; i < frame.NbSamples; i++ {
		c.pcm = append(c.pcm, int16(binary.LittleEndian.Uint16(frame.Data[0][i*2:])))
	}
	return nil, nil
}

func (c *g722Codec) Flush() ([]*avutil.Packet, error) {
	raw, err := G722Encode(c.pcm)
	c.pcm = nil
	if err != nil || len(raw) == 0 {
		return nil, err
	}
	return []*avutil.Packet{{Data: raw, Duration: int64(len(raw) * 2)}}, nil
}

func (c *g722Codec) Decode(pkt *avutil.Packet) ([]*avutil.Frame, error) {
	if pkt == nil || len(pkt.Data) == 0 {
		return nil, nil
	}
	samples, err := G722Decode(pkt.Data)
	if err != nil || len(samples) == 0 {
		return nil, err
	}
	out, err := avutil.NewAudio(avutil.SampleS16, 16000, avutil.Mono(), len(samples))
	if err != nil {
		return nil, err
	}
	for i, s := range samples {
		binary.LittleEndian.PutUint16(out.Data[0][i*2:], uint16(s))
	}
	out.PTS = pkt.PTS
	out.Duration = int64(len(samples))
	out.TimeBase = avutil.Q(1, 16000)
	return []*avutil.Frame{out}, nil
}

// G.722 at 64 kbit/s with trellis off. The predictor follows FFmpeg 5.1.9,
// whose decoder is the ITU bit-exact one. Each byte holds one 2-bit high-band
// code above a 6-bit low-band code. There is no packet header.

const g722PrevSamples = 1024

type g722Band struct {
	predictor   int16
	zero        int32
	partReconst [2]int8
	prevReconst int16
	pole        [2]int16
	diffMem     [6]int32
	zeroMem     [6]int16
	logFactor   int16
	scale       int16
}

type g722State struct {
	prev [g722PrevSamples]int16
	pos  int
	band [2]g722Band
}

func newG722() *g722State {
	s := &g722State{pos: 22}
	s.band[0].scale = 8
	s.band[1].scale = 2
	return s
}

// G722Encode packs 16 kHz mono s16 samples into raw G.722 bytes.
// An odd trailing sample is repeated, matching the encoder's last-frame rule.
func G722Encode(pcm []int16) ([]byte, error) {
	if len(pcm) == 0 {
		return nil, nil
	}
	s := newG722()
	out := make([]byte, (len(pcm)+1)/2)
	i := 0
	for ; i+1 < len(pcm); i += 2 {
		out[i/2] = s.encodePair(pcm[i], pcm[i+1])
	}
	if i < len(pcm) {
		out[i/2] = s.encodePair(pcm[i], pcm[i])
	}
	return out, nil
}

// G722Decode expands raw G.722 bytes into 16 kHz mono s16 samples.
func G722Decode(data []byte) ([]int16, error) {
	if len(data) == 0 {
		return nil, nil
	}
	s := newG722()
	out := make([]int16, len(data)*2)
	for i, b := range data {
		out[i*2], out[i*2+1] = s.decodeByte(b)
	}
	return out, nil
}

func (s *g722State) encodePair(a, b int16) byte {
	x0, x1 := s.qmf(a, b)
	// C parses "xout[0] + xout[1] >> 14" as (xout[0] + xout[1]) >> 14.
	xlow := (x0 + x1) >> 14
	xhigh := (x0 - x1) >> 14
	ihigh := g722EncodeHigh(&s.band[1], xhigh)
	ilow := g722EncodeLow(&s.band[0], xlow)
	dhigh := (int32(s.band[1].scale) * int32(g722HighInvQuant[ihigh])) >> 10
	s.band[1].updateHigh(dhigh, ihigh)
	s.band[0].updateLow(ilow >> 2)
	return byte(ihigh<<6 | ilow)
}

func (s *g722State) decodeByte(v byte) (int16, int16) {
	ihigh := int(v >> 6)
	ilow := int(v & 0x3f)
	rlow := g722ClipP2((int32(s.band[0].scale)*int32(g722LowInvQuant6[ilow])>>10)+int32(s.band[0].predictor), 14)
	s.band[0].updateLow(ilow >> 2)
	dhigh := (int32(s.band[1].scale) * int32(g722HighInvQuant[ihigh])) >> 10
	rhigh := g722ClipP2(dhigh+int32(s.band[1].predictor), 14)
	s.band[1].updateHigh(dhigh, ihigh)
	x0, x1 := s.qmf(int16(rlow+rhigh), int16(rlow-rhigh))
	return int16(g722Clip16(x0 >> 11)), int16(g722Clip16(x1 >> 11))
}

// qmf pushes two samples and returns the 24-tap mirror-filter accumulators.
func (s *g722State) qmf(a, b int16) (int32, int32) {
	s.prev[s.pos] = a
	s.prev[s.pos+1] = b
	s.pos += 2
	x0, x1 := g722ApplyQMF(s.prev[s.pos-24 : s.pos])
	if s.pos >= g722PrevSamples {
		copy(s.prev[:22], s.prev[s.pos-22:s.pos])
		s.pos = 22
	}
	return x0, x1
}

func g722ApplyQMF(prev []int16) (x0, x1 int32) {
	for i := 0; i < 12; i++ {
		x1 += int32(prev[i*2]) * g722QMFEven[i]
		x0 += int32(prev[i*2+1]) * g722QMFOdd[i]
	}
	return x0, x1
}

func g722EncodeHigh(b *g722Band, xhigh int32) int {
	diff := g722Clip16(xhigh - int32(b.predictor))
	pred := (141 * int32(b.scale)) >> 8
	mag := diff ^ (diff >> 31)
	code := 0
	if mag < pred {
		code = 1
	}
	if diff >= 0 {
		code += 2
	}
	return code
}

func g722EncodeLow(b *g722Band, xlow int32) int {
	diff := g722Clip16(xlow - int32(b.predictor))
	// C parses "limit + 1 << 10" as (limit + 1) << 10.
	mag := diff ^ (diff >> 31)
	limit := (mag + 1) << 10
	i := 0
	scale := int32(b.scale)
	if limit > int32(g722LowQuant[8])*scale {
		i = 9
	}
	for i < 29 && limit > int32(g722LowQuant[i])*scale {
		i++
	}
	base := 61
	if diff < 0 {
		base = 33
		if i < 2 {
			base = 63
		}
	}
	return base - i
}

func (b *g722Band) updateLow(ilow int) {
	cur := (int32(b.scale) * int32(g722LowInvQuant4[ilow])) >> 10
	b.adapt(cur)
	b.logFactor = int16(g722Clip((int32(b.logFactor)*127)>>7+int32(g722LowLogStep[ilow]), 0, 18432))
	b.scale = int16(g722LinearScale(int32(b.logFactor) - (8 << 11)))
}

func (b *g722Band) updateHigh(dhigh int32, ihigh int) {
	b.adapt(dhigh)
	b.logFactor = int16(g722Clip((int32(b.logFactor)*127)>>7+int32(g722HighLogStep[ihigh&1]), 0, 22528))
	b.scale = int16(g722LinearScale(int32(b.logFactor) - (10 << 11)))
}

func (b *g722Band) adapt(curDiff int32) {
	curPart := int8(0)
	if b.zero+curDiff < 0 {
		curPart = 1
	}
	sg0 := g722Sign(curPart != b.partReconst[0])
	sg1 := g722Sign(curPart == b.partReconst[1])
	b.partReconst[1] = b.partReconst[0]
	b.partReconst[0] = curPart

	clipped := g722Clip(int32(b.pole[0]), -8191, 8191)
	b.pole[1] = int16(g722Clip((sg0*clipped)>>5+sg1*128+(int32(b.pole[1])*127)>>7, -12288, 12288))
	limit := 15360 - int32(b.pole[1])
	b.pole[0] = int16(g722Clip(-192*sg0+(int32(b.pole[0])*255)>>8, -limit, limit))

	b.reconstructZero(curDiff)

	curQtzd := g722Clip16((int32(b.predictor) + curDiff) * 2)
	b.predictor = int16(g722Clip16(b.zero +
		(int32(b.pole[0])*curQtzd)>>15 +
		(int32(b.pole[1])*int32(b.prevReconst))>>15))
	b.prevReconst = int16(curQtzd)
}

func (b *g722Band) reconstructZero(curDiff int32) {
	var sum int32
	d := int32(0)
	if curDiff != 0 {
		d = 1
	}
	acc := func(k int, x int32) {
		tmp := x
		step := int32(128)
		if b.diffMem[k]^curDiff < 0 {
			step = -128
		}
		b.zeroMem[k] = int16((int32(b.zeroMem[k])*255)>>8 + d*step)
		b.diffMem[k] = tmp
		sum += (tmp * int32(b.zeroMem[k])) >> 15
	}
	acc(5, b.diffMem[4])
	acc(4, b.diffMem[3])
	acc(3, b.diffMem[2])
	acc(2, b.diffMem[1])
	acc(1, b.diffMem[0])
	acc(0, curDiff*2)
	b.zero = sum
}

func g722LinearScale(logFactor int32) int32 {
	wd1 := int32(g722InvLog2[(logFactor>>6)&31])
	shift := logFactor >> 11
	if shift < 0 {
		return wd1 >> uint(-shift)
	}
	return wd1 << uint(shift)
}

func g722Sign(positive bool) int32 {
	if positive {
		return 1
	}
	return -1
}

func g722Clip(v, lo, hi int32) int32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func g722Clip16(v int32) int32 {
	if v > 32767 {
		return 32767
	}
	if v < -32768 {
		return -32768
	}
	return v
}

func g722ClipP2(v int32, bits uint) int32 {
	lim := int32(1) << bits
	if v < -lim {
		return -lim
	}
	if v >= lim {
		return lim - 1
	}
	return v
}
