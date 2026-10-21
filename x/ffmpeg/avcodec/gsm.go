package avcodec

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// gsmCodec is GSM 06.10 (libgsm) at 13 kbit/s. Decode follows the FFmpeg 5.1.9
// decoder. Encode follows libgsm with FAST and WAV49 off.
type gsmCodec struct {
	par     Parameters
	ref     [280]int16
	v       [9]int32
	lar     [2][8]int32
	larIdx  int
	msr     int32
	buf     []byte
	pcm     []int16
	enc     gsmEnc
	encBase int64
	encOff  int64
	encHave bool
	ms      bool
}

func newGSMCodec(par Parameters) *gsmCodec {
	return &gsmCodec{par: par, enc: gsmEnc{nrp: 40}}
}

func newGSMMS(par Parameters) *gsmCodec {
	c := newGSMCodec(par)
	c.ms = true
	return c
}

func (g *gsmCodec) Decode(pkt *avutil.Packet) ([]*avutil.Frame, error) {
	if pkt == nil || len(pkt.Data) == 0 {
		return nil, nil
	}
	g.buf = append(g.buf, pkt.Data...)
	var samples []int16
	for len(g.buf) >= 33 {
		frame, err := g.decodeFrame(g.buf[:33])
		if err != nil {
			return nil, err
		}
		samples = append(samples, frame...)
		g.buf = g.buf[33:]
	}
	if len(samples) == 0 {
		return nil, nil
	}
	out, err := avutil.NewAudio(avutil.SampleS16, 8000, avutil.Mono(), len(samples))
	if err != nil {
		return nil, err
	}
	for i, s := range samples {
		binary.LittleEndian.PutUint16(out.Data[0][i*2:], uint16(s))
	}
	out.PTS = pkt.PTS
	out.Duration = int64(len(samples))
	out.TimeBase = avutil.Q(1, 8000)
	return []*avutil.Frame{out}, nil
}

func (g *gsmCodec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil || frame.NbSamples == 0 {
		return nil, nil
	}
	if frame.SampleFmt != avutil.SampleS16 {
		return nil, fmt.Errorf("%w: gsm %s", avutil.ErrInvalidData, frame.SampleFmt)
	}
	if ch := frame.Layout.Channels(); ch > 1 {
		return nil, fmt.Errorf("%w: gsm channels", avutil.ErrRange)
	}
	if len(frame.Data) == 0 || len(frame.Data[0]) < frame.NbSamples*2 {
		return nil, fmt.Errorf("%w: gsm samples", avutil.ErrInvalidData)
	}
	if !g.encHave {
		g.encBase = frame.PTS
		g.encHave = true
	}
	for i := 0; i < frame.NbSamples; i++ {
		g.pcm = append(g.pcm, int16(binary.LittleEndian.Uint16(frame.Data[0][i*2:])))
	}
	return g.emitGSM()
}

func (g *gsmCodec) Flush() ([]*avutil.Packet, error) {
	if len(g.pcm) == 0 {
		return nil, nil
	}
	need := 160
	if g.ms {
		need = 320
	}
	for len(g.pcm) < need {
		g.pcm = append(g.pcm, 0)
	}
	return g.emitGSM()
}

func (g *gsmCodec) emitGSM() ([]*avutil.Packet, error) {
	if g.ms {
		return g.emitGSMMS()
	}
	var pkts []*avutil.Packet
	for len(g.pcm) >= 160 {
		raw, err := g.encodeFrame(g.pcm[:160])
		if err != nil {
			return nil, err
		}
		g.pcm = append([]int16(nil), g.pcm[160:]...)
		pts := g.encBase + g.encOff
		g.encOff += 160
		pkts = append(pkts, &avutil.Packet{
			Data: append([]byte(nil), raw[:]...),
			PTS:  pts, DTS: pts, Duration: 160,
			TimeBase: avutil.Q(1, 8000),
		})
	}
	return pkts, nil
}

func (g *gsmCodec) emitGSMMS() ([]*avutil.Packet, error) {
	var pkts []*avutil.Packet
	for len(g.pcm) >= 320 {
		raw, err := g.encodeMS(g.pcm[:320])
		if err != nil {
			return nil, err
		}
		g.pcm = append([]int16(nil), g.pcm[320:]...)
		pts := g.encBase + g.encOff
		g.encOff += 320
		pkts = append(pkts, &avutil.Packet{
			Data: append([]byte(nil), raw[:]...),
			PTS:  pts, DTS: pts, Duration: 320,
			TimeBase: avutil.Q(1, 8000),
		})
	}
	return pkts, nil
}

func (g *gsmCodec) decodeFrame(frame []byte) ([]int16, error) {
	if len(frame) < 33 {
		return nil, fmt.Errorf("%w: gsm frame", avutil.ErrInvalidData)
	}
	r := bitr{b: frame}
	magic, err := r.u(4)
	if err != nil {
		return nil, err
	}
	if magic != 0xd {
		return nil, fmt.Errorf("%w: gsm magic", avutil.ErrInvalidData)
	}
	lar := g.lar[g.larIdx][:]
	params := []struct {
		n      int
		factor int32
		offset int32
	}{
		{6, 13107, 1 << 15},
		{6, 13107, 1 << 15},
		{5, 13107, (1 << 14) + 2048*2},
		{5, 13107, (1 << 14) - 2560*2},
		{4, 19223, (1 << 13) + 94*2},
		{4, 17476, (1 << 13) - 1792*2},
		{3, 31454, (1 << 12) - 341*2},
		{3, 29708, (1 << 12) - 1144*2},
	}
	for i, p := range params {
		coded, err := r.u(p.n)
		if err != nil {
			return nil, err
		}
		lar[i] = decodeLogArea(int32(coded), p.factor, p.offset)
	}
	ref := g.ref[120:]
	for s := 0; s < 4; s++ {
		lag, err := r.u(7)
		if err != nil {
			return nil, err
		}
		gain, err := r.u(2)
		if err != nil {
			return nil, err
		}
		offset, err := r.u(2)
		if err != nil {
			return nil, err
		}
		if lag < 40 {
			lag = 40
		}
		if lag > 120 {
			lag = 120
		}
		dst := ref[s*40 : s*40+40]
		src := g.ref[120+s*40-int(lag):]
		gsmMult40(dst, src, gsmLongTermGain[gain])
		if err := apcmDequant(&r, dst[offset:]); err != nil {
			return nil, err
		}
	}
	copy(g.ref[:120], g.ref[160:280])
	samples := make([]int16, 160)
	g.shortTerm(samples, g.ref[120:])
	g.msr = postprocess(samples, g.msr)
	return samples, nil
}

func decodeLogArea(coded, factor, offset int32) int32 {
	coded = (coded << 10) - offset
	return gsmMult(coded, factor) * 2
}

func gsmMult(a, b int32) int32 {
	return (a*b + (1 << 14)) >> 15
}

func gsmMult40(dst, src []int16, gain int16) {
	if gsmMult40Asm(dst, src, gain) {
		return
	}
	for i := 0; i < 40 && i < len(dst) && i < len(src); i++ {
		dst[i] = int16(gsmMult(int32(gain), int32(src[i])))
	}
}

func apcmDequant(r *bitr, dst []int16) error {
	maxidx, err := r.u(6)
	if err != nil {
		return err
	}
	if int(maxidx) >= len(gsmDequant) {
		return fmt.Errorf("%w: gsm xmax", avutil.ErrInvalidData)
	}
	tab := gsmDequant[maxidx]
	for i := 0; i < 13; i++ {
		val, err := r.u(3)
		if err != nil {
			return err
		}
		dst[3*i] += tab[gsmRequant3[val]]
	}
	return nil
}

func (g *gsmCodec) shortTerm(dst, src []int16) {
	lar := g.lar[g.larIdx]
	prev := g.lar[g.larIdx^1]
	var rrp [8]int32
	apply := func(from, to int, mix func(a, b int32) int32) {
		for i := 0; i < 8; i++ {
			rrp[i] = rrpWeight(mix(prev[i], lar[i]))
		}
		for i := from; i < to; i++ {
			dst[i] = int16(filterValue(int32(src[i]), &rrp, &g.v))
		}
	}
	apply(0, 13, func(a, b int32) int32 { return (a >> 2) + (a >> 1) + (b >> 2) })
	apply(13, 27, func(a, b int32) int32 { return (a >> 1) + (b >> 1) })
	apply(27, 40, func(a, b int32) int32 { return (a >> 2) + (b >> 1) + (b >> 2) })
	apply(40, 160, func(_, b int32) int32 { return b })
	g.larIdx ^= 1
}

func rrpWeight(filtered int32) int32 {
	abs := filtered
	if abs < 0 {
		abs = -abs
	}
	switch {
	case abs < 11059:
		abs <<= 1
	case abs < 20070:
		abs += 11059
	default:
		abs = (abs >> 2) + 26112
	}
	if filtered < 0 {
		return -abs
	}
	return abs
}

func filterValue(in int32, rrp *[8]int32, v *[9]int32) int32 {
	for i := 7; i >= 0; i-- {
		in -= gsmMult(rrp[i], v[i])
		v[i+1] = v[i] + gsmMult(rrp[i], in)
	}
	v[0] = in
	return in
}

func postprocess(data []int16, msr int32) int32 {
	for i := range data {
		msr = clip16(int32(data[i]) + gsmMult(msr, 28180))
		data[i] = int16(clip16(msr*2) &^ 7)
	}
	return msr
}

func clip16(v int32) int32 {
	if v > 32767 {
		return 32767
	}
	if v < -32768 {
		return -32768
	}
	return v
}

var gsmLongTermGain = [4]int16{3277, 11469, 21299, 32767}

var gsmRequant3 = [8]int{0, 1, 2, 3, 4, 5, 6, 7}
