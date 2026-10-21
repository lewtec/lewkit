package avcodec

import (
	"encoding/binary"
	"fmt"
	"math"
	"sync"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// twolameCodec encodes mono MP2 with the TwoLAME 0.4.0 psycho model.
type twolameCodec struct {
	pcm  []int16
	rate int
	kbps int
}

func newTwolame(par Parameters) (*twolameCodec, error) {
	rate := par.SampleRate
	if rate <= 0 {
		rate = 44100
	}
	kbps := 0
	if par.BitRate > 0 {
		kbps = par.BitRate / 1000
	}
	if _, err := newTLEncoder(rate, kbps); err != nil {
		return nil, err
	}
	return &twolameCodec{rate: rate, kbps: kbps}, nil
}

func (c *twolameCodec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil || frame.NbSamples == 0 {
		return nil, nil
	}
	if frame.SampleFmt != avutil.SampleS16 || frame.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: twolame", avutil.ErrInvalidData)
	}
	if len(frame.Data) == 0 || len(frame.Data[0]) < frame.NbSamples*2 {
		return nil, fmt.Errorf("%w: twolame samples", avutil.ErrInvalidData)
	}
	for i := 0; i < frame.NbSamples; i++ {
		c.pcm = append(c.pcm, int16(binary.LittleEndian.Uint16(frame.Data[0][i*2:])))
	}
	return nil, nil
}

func (c *twolameCodec) Flush() ([]*avutil.Packet, error) {
	n := len(c.pcm)
	raw, err := TwolameEncodeMono(c.pcm, c.rate, c.kbps)
	c.pcm = nil
	if err != nil || len(raw) == 0 {
		return nil, err
	}
	return []*avutil.Packet{{Data: raw, Duration: int64(n)}}, nil
}

// TwolameEncodeMono encodes interleaved mono s16 PCM as MPEG-1/2 Layer II
// the way ffmpeg's libtwolame wrapper does: psycho model 3, CBR, no CRC,
// no energy extension, copyright 0, original 0, padding off.
// A bitrate of 0 selects 192 kbit/s at 28 kHz and above, otherwise 80.
// A trailing partial frame is zero-padded to 1152 samples, matching
// twolame_encode_flush. Exact multiples flush to nothing.
func TwolameEncodeMono(pcm []int16, rate, bitrateKbps int) ([]byte, error) {
	enc, err := newTLEncoder(rate, bitrateKbps)
	if err != nil {
		return nil, err
	}
	var out []byte
	for len(pcm) > 0 {
		n := tlSamplesPerFrame - enc.samplesIn
		if n > len(pcm) {
			n = len(pcm)
		}
		copy(enc.pcm[enc.samplesIn:], pcm[:n])
		enc.samplesIn += n
		pcm = pcm[n:]
		if enc.samplesIn == tlSamplesPerFrame {
			out = append(out, enc.encodeFrame()...)
			enc.samplesIn = 0
		}
	}
	if enc.samplesIn > 0 {
		for i := enc.samplesIn; i < tlSamplesPerFrame; i++ {
			enc.pcm[i] = 0
		}
		out = append(out, enc.encodeFrame()...)
		enc.samplesIn = 0
	}
	return out, nil
}

const tlSamplesPerFrame = 1152

type tlStream struct {
	buf     []byte
	byteIdx int
	bitIdx  int
	totbit  int
}

func (b *tlStream) put1(bit int) {
	b.totbit++
	b.buf[b.byteIdx] |= byte((bit & 1) << uint(b.bitIdx-1))
	b.bitIdx--
	if b.bitIdx == 0 {
		b.bitIdx = 8
		b.byteIdx++
		if b.byteIdx < len(b.buf) {
			b.buf[b.byteIdx] = 0
		}
	}
}

func (b *tlStream) putbits(val uint32, n int) {
	putmask := [9]uint32{0x0, 0x1, 0x3, 0x7, 0xf, 0x1f, 0x3f, 0x7f, 0xff}
	j := n
	for j > 0 {
		k := j
		if k > b.bitIdx {
			k = b.bitIdx
		}
		tmp := val >> uint(j-k)
		b.buf[b.byteIdx] |= byte((tmp & putmask[k]) << uint(b.bitIdx-k))
		b.bitIdx -= k
		b.totbit += k
		if b.bitIdx == 0 {
			b.bitIdx = 8
			b.byteIdx++
			if b.byteIdx < len(b.buf) {
				b.buf[b.byteIdx] = 0
			}
		}
		j -= k
	}
}

type tlEncoder struct {
	rate          int
	bitrate       int
	version       int
	bitrateIndex  int
	samplerateIdx int
	tablenum      int
	sblimit       int
	jsbound       int

	samplesIn int
	pcm       [tlSamplesPerFrame]int16

	x    [2][512]float64
	m    [16][32]float64
	off  [2]int
	half [2]int

	psyOff     [2]int
	fftBuf     [2][tlFFTBUF]float64
	bark       [tlHBLKSIZE]float64
	ath        [tlHBLKSIZE]float64
	freqSubset [tlSUBSIZE]int
	cbandIndex [40]int
	cbands     int
	dbtable    [tlDBTAB]float64

	sb       [2][3][tlSCALEBLK][tlSBLIMIT]float64
	scalar   [2][3][tlSBLIMIT]int
	scfsi    [2][tlSBLIMIT]int
	bitAlloc [2][tlSBLIMIT]int
	maxSC    [2][tlSBLIMIT]float64
	smr      [2][tlSBLIMIT]float64
	coded    [2][3][tlSCALEBLK][tlSBLIMIT]uint32
}

func newTLEncoder(rate, bitrateKbps int) (*tlEncoder, error) {
	version := tlVersionForRate(rate)
	if version < 0 {
		return nil, fmt.Errorf("%w: twolame rate %d", avutil.ErrUnsupported, rate)
	}
	if bitrateKbps == 0 {
		if rate >= 28000 {
			bitrateKbps = 192
		} else {
			bitrateKbps = 80
		}
	}
	if bitrateKbps < 0 {
		return nil, fmt.Errorf("%w: twolame bitrate %d", avutil.ErrUnsupported, bitrateKbps)
	}
	brIndex := tlBitrateIndex(bitrateKbps, version)
	if brIndex < 0 {
		return nil, fmt.Errorf("%w: twolame bitrate %d", avutil.ErrUnsupported, bitrateKbps)
	}
	// MPEG-1 mono CBR refuses anything above 192 kbit/s.
	if version == 1 && bitrateKbps > 192 {
		return nil, fmt.Errorf("%w: twolame bitrate %d", avutil.ErrUnsupported, bitrateKbps)
	}
	srIdx := tlSamplerateIndex(rate)
	e := &tlEncoder{
		rate:          rate,
		bitrate:       bitrateKbps,
		version:       version,
		bitrateIndex:  brIndex,
		samplerateIdx: srIdx,
	}
	e.pickTable()
	e.initSubband()
	e.initPsycho()
	return e, nil
}

func tlVersionForRate(rate int) int {
	switch rate {
	case 48000, 44100, 32000:
		return 1
	case 24000, 22050, 16000:
		return 0
	default:
		return -1
	}
}

func tlSamplerateIndex(rate int) int {
	switch rate {
	case 44100, 22050:
		return 0
	case 48000, 24000:
		return 1
	case 32000, 16000:
		return 2
	default:
		return -1
	}
}

func tlBitrateIndex(bitrate, version int) int {
	if version != 0 && version != 1 {
		return -1
	}
	for index := 1; index < 15; index++ {
		if tlBitrateTable[version][index] == bitrate {
			return index
		}
	}
	return -1
}

func (e *tlEncoder) pickTable() {
	brPerCh := e.bitrate
	sfrq := int(float64(e.rate) / 1000.0)
	if e.version == 1 {
		if (sfrq == 48 && brPerCh >= 56) || (brPerCh >= 56 && brPerCh <= 80) {
			e.tablenum = 0
		} else if sfrq != 48 && brPerCh >= 96 {
			e.tablenum = 1
		} else if sfrq != 32 && brPerCh <= 48 {
			e.tablenum = 2
		} else {
			e.tablenum = 3
		}
	} else {
		e.tablenum = 4
	}
	e.sblimit = tlSBlimit[e.tablenum]
	e.jsbound = e.sblimit
}

func (e *tlEncoder) initSubband() {
	for i := 0; i < 16; i++ {
		for k := 0; k < 32; k++ {
			// PI64 is the macro PI/64, so the product is ((2i+1)*k*PI)/64.
			y := 1e9 * math.Cos(float64((2*i+1)*k)*tlPI/64)
			e.m[i][k] = math.Round(y) * 1e-9
		}
	}
}

func (e *tlEncoder) initPsycho() {
	e.psyOff[0] = 256
	e.psyOff[1] = 256
	for i := 0; i < tlDBTAB; i++ {
		x := float64(i) / 10.0
		e.dbtable[i] = 10*math.Log10(1+math.Pow(10.0, x/10.0)) - x
	}
	sfreq := float64(e.rate)
	for i := 1; i < tlHBLKSIZE; i++ {
		freq := float64(i) * sfreq / float64(tlBLKSIZE)
		e.bark[i] = tlFreqToBark(freq)
		e.ath[i] = tlATHDB(freq, 0)
	}
	cbase := 0
	cbands := 0
	e.cbandIndex[0] = 1
	for i := 1; i < tlHBLKSIZE; i++ {
		if e.bark[i]-e.bark[cbase] > 1.0 {
			cbase = i
			cbands++
			e.cbandIndex[cbands] = cbase
		}
	}
	cbands++
	e.cbandIndex[cbands] = 513
	e.cbands = cbands

	idx := 0
	i := 1
	for ; i < (3*16)+1; i++ {
		e.freqSubset[idx] = i
		idx++
	}
	for ; i < (6*16)+1; i += 2 {
		e.freqSubset[idx] = i
		idx++
	}
	for ; i < (12*16)+1; i += 4 {
		e.freqSubset[idx] = i
		idx++
	}
	for ; i < (32*16)+1; i += 8 {
		e.freqSubset[idx] = i
		idx++
	}
}

func tlATHDB(f, value float64) float64 {
	if f < -0.3 {
		f = 3410
	}
	f /= 1000
	if f < 0.01 {
		f = 0.01
	}
	if f > 18.0 {
		f = 18.0
	}
	ath := 3.640*math.Pow(f, -0.8) -
		6.800*math.Exp(-0.6*math.Pow(f-3.4, 2.0)) +
		6.000*math.Exp(-0.15*math.Pow(f-8.7, 2.0)) +
		0.6*0.001*math.Pow(f, 4.0)
	return ath + value
}

func tlFreqToBark(freq float64) float64 {
	if freq < 0 {
		freq = 0
	}
	freq *= 0.001
	return 13.0*math.Atan(.76*freq) + 3.5*math.Atan(freq*freq/(7.5*7.5))
}

var tlHann = sync.OnceValue(func() *[tlBLKSIZE]float64 {
	var w [tlBLKSIZE]float64
	sqrt8over3 := math.Pow(8.0/3.0, 0.5)
	for i := 0; i < tlBLKSIZE; i++ {
		w[i] = sqrt8over3 * 0.5 * (1 - math.Cos(2.0*tlPI*float64(i)/float64(tlBLKSIZE))) / float64(tlBLKSIZE)
	}
	return &w
})

func (e *tlEncoder) frameBytes() int {
	average := (1152.0 / (float64(e.rate) / 1000.0)) * (float64(e.bitrate) / 8.0)
	return int(average)
}

func (e *tlEncoder) encodeFrame() []byte {
	for gr := 0; gr < 3; gr++ {
		for bl := 0; bl < tlSCALEBLK; bl++ {
			off := (gr*tlSCALEBLK + bl) * 32
			var s [tlSBLIMIT]float64
			e.filter32(e.pcm[off:off+32], &s)
			e.sb[0][gr][bl] = s
		}
	}
	e.calcScale()
	e.findSFMax()
	e.psycho3()
	e.transmission()
	adb := e.frameBytes() * 8
	adb = e.allocate(adb)

	buf := make([]byte, e.frameBytes()+8)
	bs := tlStream{buf: buf, bitIdx: 8}
	e.writeHeader(&bs)
	e.writeBitAlloc(&bs)
	e.writeScale(&bs)
	e.quantize()
	e.writeSamples(&bs)
	for i := 0; i < adb; i++ {
		bs.put1(0)
	}
	n := bs.totbit / 8
	if n > len(buf) {
		n = len(buf)
	}
	return buf[:n]
}

func (e *tlEncoder) filter32(pcm []int16, s *[tlSBLIMIT]float64) {
	ch := 0
	base := e.half[ch]*256 + e.off[ch]
	for i := 0; i < 32; i++ {
		e.x[ch][base+(31-i)*8] = float64(pcm[i]) / tlSCALE
	}
	pa := e.off[ch]
	pb := (pa + 1) % 8
	pc := (pa + 2) % 8
	pd := (pa + 3) % 8
	pe := (pa + 4) % 8
	pf := (pa + 5) % 8
	pg := (pa + 6) % 8
	ph := (pa + 7) % 8
	halfBase := e.half[ch] * 256
	var y [64]float64
	for i := 0; i < 32; i++ {
		dp := halfBase + i*8
		t := e.x[ch][dp+pa] * tlEnwindow[i]
		t += e.x[ch][dp+pb] * tlEnwindow[i+64]
		t += e.x[ch][dp+pc] * tlEnwindow[i+128]
		t += e.x[ch][dp+pd] * tlEnwindow[i+192]
		t += e.x[ch][dp+pe] * tlEnwindow[i+256]
		t += e.x[ch][dp+pf] * tlEnwindow[i+320]
		t += e.x[ch][dp+pg] * tlEnwindow[i+384]
		t += e.x[ch][dp+ph] * tlEnwindow[i+448]
		y[i] = t
	}
	var yprime [32]float64
	yprime[0] = y[16]

	other := 256
	if e.half[ch] != 0 {
		other = 0
		pa = (e.off[ch] + 1) & 7
	} else {
		pa = e.off[ch]
	}
	pb = (pa + 1) % 8
	pc = (pa + 2) % 8
	pd = (pa + 3) % 8
	pe = (pa + 4) % 8
	pf = (pa + 5) % 8
	pg = (pa + 6) % 8
	ph = (pa + 7) % 8
	for i := 0; i < 32; i++ {
		dp := other + i*8
		en := i + 32
		t := e.x[ch][dp+pa] * tlEnwindow[en]
		t += e.x[ch][dp+pb] * tlEnwindow[en+64]
		t += e.x[ch][dp+pc] * tlEnwindow[en+128]
		t += e.x[ch][dp+pd] * tlEnwindow[en+192]
		t += e.x[ch][dp+pe] * tlEnwindow[en+256]
		t += e.x[ch][dp+pf] * tlEnwindow[en+320]
		t += e.x[ch][dp+pg] * tlEnwindow[en+384]
		t += e.x[ch][dp+ph] * tlEnwindow[en+448]
		y[i+32] = t
		if i > 0 && i < 17 {
			yprime[i] = y[i+16] + y[16-i]
		}
	}
	for i := 17; i < 32; i++ {
		yprime[i] = y[i+16] - y[80-i]
	}
	for i := 15; i >= 0; i-- {
		s0 := 0.0
		s1 := 0.0
		mp := 0
		row := e.m[i]
		for j := 0; j < 8; j++ {
			s0 += row[mp] * yprime[mp]
			s1 += row[mp+1] * yprime[mp+1]
			s0 += row[mp+2] * yprime[mp+2]
			s1 += row[mp+3] * yprime[mp+3]
			mp += 4
		}
		s[i] = s0 + s1
		s[31-i] = s0 - s1
	}
	e.half[ch] = (e.half[ch] + 1) & 1
	if e.half[ch] == 1 {
		e.off[ch] = (e.off[ch] + 7) & 7
	}
}

func (e *tlEncoder) calcScale() {
	sblimit := e.sblimit
	for gr := 0; gr < 3; gr++ {
		for sb := 0; sb < sblimit; sb++ {
			curMax := math.Abs(e.sb[0][gr][tlSCALEBLK-1][sb])
			for j := tlSCALEBLK - 2; j >= 0; j-- {
				temp := math.Abs(e.sb[0][gr][j][sb])
				if temp > curMax {
					curMax = temp
				}
			}
			l := 16
			scaleFac := 32
			for l != 0 {
				if curMax <= tlScale[scaleFac] {
					scaleFac += l
				} else {
					scaleFac -= l
				}
				l >>= 1
			}
			if curMax > tlScale[scaleFac] {
				scaleFac--
			}
			e.scalar[0][gr][sb] = scaleFac
		}
	}
}

func (e *tlEncoder) findSFMax() {
	sblimit := e.sblimit
	for sb := 0; sb < sblimit; sb++ {
		lowest := e.scalar[0][0][sb]
		if lowest > e.scalar[0][1][sb] {
			lowest = e.scalar[0][1][sb]
		}
		if lowest > e.scalar[0][2][sb] {
			lowest = e.scalar[0][2][sb]
		}
		e.maxSC[0][sb] = tlScale[lowest]
	}
	for sb := sblimit; sb < tlSBLIMIT; sb++ {
		e.maxSC[0][sb] = 1e-20
		e.maxSC[1][sb] = 1e-20
	}
}

func (e *tlEncoder) transmission() {
	sblimit := e.sblimit
	for i := 0; i < sblimit; i++ {
		dscf0 := e.scalar[0][0][i] - e.scalar[0][1][i]
		dscf1 := e.scalar[0][1][i] - e.scalar[0][2][i]
		c0 := tlSCFClass(dscf0)
		c1 := tlSCFClass(dscf1)
		switch tlSCFPattern[c0][c1] {
		case 0x123:
			e.scfsi[0][i] = 0
		case 0x122:
			e.scfsi[0][i] = 3
			e.scalar[0][2][i] = e.scalar[0][1][i]
		case 0x133:
			e.scfsi[0][i] = 3
			e.scalar[0][1][i] = e.scalar[0][2][i]
		case 0x113:
			e.scfsi[0][i] = 1
			e.scalar[0][1][i] = e.scalar[0][0][i]
		case 0x111:
			e.scfsi[0][i] = 2
			e.scalar[0][1][i] = e.scalar[0][0][i]
			e.scalar[0][2][i] = e.scalar[0][0][i]
		case 0x222:
			e.scfsi[0][i] = 2
			e.scalar[0][0][i] = e.scalar[0][1][i]
			e.scalar[0][2][i] = e.scalar[0][1][i]
		case 0x333:
			e.scfsi[0][i] = 2
			e.scalar[0][0][i] = e.scalar[0][2][i]
			e.scalar[0][1][i] = e.scalar[0][2][i]
		case 0x444:
			e.scfsi[0][i] = 2
			if e.scalar[0][0][i] > e.scalar[0][2][i] {
				e.scalar[0][0][i] = e.scalar[0][2][i]
			}
			e.scalar[0][1][i] = e.scalar[0][0][i]
			e.scalar[0][2][i] = e.scalar[0][0][i]
		}
	}
}

func tlSCFClass(d int) int {
	if d <= -3 {
		return 0
	} else if d > -3 && d < 0 {
		return 1
	} else if d == 0 {
		return 2
	} else if d > 0 && d < 3 {
		return 3
	}
	return 4
}

func (e *tlEncoder) allocate(adb int) int {
	nch := 1
	sblimit := e.sblimit
	jsbound := e.jsbound
	bbal := 0
	for sb := 0; sb < jsbound; sb++ {
		bbal += nch * tlNBal[tlAllocLine[e.tablenum][sb]]
	}
	for sb := jsbound; sb < sblimit; sb++ {
		bbal += tlNBal[tlAllocLine[e.tablenum][sb]]
	}
	adb -= bbal + 32
	ad := adb
	var mnr [2][tlSBLIMIT]float64
	var used [2][tlSBLIMIT]int
	for sb := 0; sb < sblimit; sb++ {
		for ch := 0; ch < nch; ch++ {
			mnr[ch][sb] = tlSNR[0] - e.smr[ch][sb]
			e.bitAlloc[ch][sb] = 0
			used[ch][sb] = 0
		}
	}
	bspl, bscf, bsel := 0, 0, 0
	for {
		minSB := -1
		minCH := -1
		small := 999999.0
		for ch := 0; ch < nch; ch++ {
			for sb := 0; sb < sblimit; sb++ {
				if used[ch][sb] != 2 && small > mnr[ch][sb] {
					small = mnr[ch][sb]
					minSB = sb
					minCH = ch
				}
			}
		}
		if minSB <= -1 {
			break
		}
		thisline := tlAllocLine[e.tablenum][minSB]
		nextStep := tlStepIndex[thisline][e.bitAlloc[minCH][minSB]+1]
		increment := tlSCALEBLK * tlGroup[nextStep] * tlBitCount[nextStep]
		if used[minCH][minSB] != 0 {
			cur := tlStepIndex[thisline][e.bitAlloc[minCH][minSB]]
			increment -= tlSCALEBLK * tlGroup[cur] * tlBitCount[cur]
		}
		oth := 1 - minCH
		scale, seli := 0, 0
		if used[minCH][minSB] == 0 {
			seli = 2
			scale = 6 * tlSFsPerSCFSI[e.scfsi[minCH][minSB]]
			if nch == 2 && minSB >= jsbound {
				seli += 2
				scale += 6 * tlSFsPerSCFSI[e.scfsi[oth][minSB]]
			}
		}
		if ad >= bspl+bscf+bsel+seli+scale+increment {
			e.bitAlloc[minCH][minSB]++
			ba := e.bitAlloc[minCH][minSB]
			bspl += increment
			bscf += scale
			bsel += seli
			used[minCH][minSB] = 1
			step := tlStepIndex[thisline][ba]
			mnr[minCH][minSB] = tlSNR[step] - e.smr[minCH][minSB]
			if ba >= (1<<tlNBal[tlAllocLine[e.tablenum][minSB]])-1 {
				used[minCH][minSB] = 2
			}
		} else {
			used[minCH][minSB] = 2
		}
		if minSB >= jsbound && nch == 2 {
			ba := e.bitAlloc[minCH][minSB]
			e.bitAlloc[oth][minSB] = ba
			used[oth][minSB] = used[minCH][minSB]
			step := tlStepIndex[thisline][ba]
			mnr[oth][minSB] = tlSNR[step] - e.smr[oth][minSB]
		}
	}
	ad -= bspl + bscf + bsel
	for ch := 0; ch < nch; ch++ {
		for sb := sblimit; sb < tlSBLIMIT; sb++ {
			e.bitAlloc[ch][sb] = 0
		}
	}
	return ad
}

func (e *tlEncoder) writeHeader(b *tlStream) {
	b.putbits(0xfff, 12)
	b.put1(e.version)
	b.putbits(2, 2) // 4 - layer
	b.put1(1)       // no CRC
	b.putbits(uint32(e.bitrateIndex), 4)
	b.putbits(uint32(e.samplerateIdx), 2)
	b.put1(0) // padding
	b.put1(0) // private
	b.putbits(3, 2)
	b.putbits(0, 2)
	b.put1(0) // copyright
	b.put1(0) // original, ffmpeg clears the libtwolame default
	b.putbits(0, 2)
}

func (e *tlEncoder) writeBitAlloc(b *tlStream) {
	nch := 1
	for sb := 0; sb < e.sblimit; sb++ {
		chlim := nch
		if sb >= e.jsbound {
			chlim = 1
		}
		for ch := 0; ch < chlim; ch++ {
			n := tlNBal[tlAllocLine[e.tablenum][sb]]
			b.putbits(uint32(e.bitAlloc[ch][sb]), n)
		}
	}
}

func (e *tlEncoder) writeScale(b *tlStream) {
	nch := 1
	for sb := 0; sb < e.sblimit; sb++ {
		for ch := 0; ch < nch; ch++ {
			if e.bitAlloc[ch][sb] != 0 {
				b.putbits(uint32(e.scfsi[ch][sb]), 2)
			}
		}
	}
	for sb := 0; sb < e.sblimit; sb++ {
		for ch := 0; ch < nch; ch++ {
			if e.bitAlloc[ch][sb] == 0 {
				continue
			}
			switch e.scfsi[ch][sb] {
			case 0:
				for gr := 0; gr < 3; gr++ {
					b.putbits(uint32(e.scalar[ch][gr][sb]), 6)
				}
			case 1, 3:
				b.putbits(uint32(e.scalar[ch][0][sb]), 6)
				b.putbits(uint32(e.scalar[ch][2][sb]), 6)
			case 2:
				b.putbits(uint32(e.scalar[ch][0][sb]), 6)
			}
		}
	}
}

func (e *tlEncoder) quantize() {
	nch := 1
	sblimit := e.sblimit
	jsbound := e.jsbound
	for gr := 0; gr < 3; gr++ {
		for j := 0; j < tlSCALEBLK; j++ {
			for sb := 0; sb < sblimit; sb++ {
				chlim := nch
				if sb >= jsbound {
					chlim = 1
				}
				for ch := 0; ch < chlim; ch++ {
					if e.bitAlloc[ch][sb] == 0 {
						continue
					}
					d := e.sb[ch][gr][j][sb] / tlScale[e.scalar[ch][gr][sb]]
					line := tlAllocLine[e.tablenum][sb]
					q := tlStepIndex[line][e.bitAlloc[ch][sb]]
					d = d*tlQA[q] + tlQB[q]
					sig := 0
					if d >= 0 {
						sig = 1
					} else {
						d += 1.0
					}
					mag := d * float64(tlSteps2n[q])
					var code uint32
					if mag > 0 {
						code = uint32(mag)
					}
					if sig != 0 {
						code |= uint32(tlSteps2n[q])
					}
					e.coded[ch][gr][j][sb] = code
				}
			}
		}
	}
}

func (e *tlEncoder) writeSamples(b *tlStream) {
	nch := 1
	sblimit := e.sblimit
	jsbound := e.jsbound
	for gr := 0; gr < 3; gr++ {
		for j := 0; j < tlSCALEBLK; j += 3 {
			for sb := 0; sb < sblimit; sb++ {
				chlim := nch
				if sb >= jsbound {
					chlim = 1
				}
				for ch := 0; ch < chlim; ch++ {
					if e.bitAlloc[ch][sb] == 0 {
						continue
					}
					line := tlAllocLine[e.tablenum][sb]
					step := tlStepIndex[line][e.bitAlloc[ch][sb]]
					if tlGroup[step] == 3 {
						for x := 0; x < 3; x++ {
							b.putbits(e.coded[ch][gr][j+x][sb], tlBitCount[step])
						}
					} else {
						y := uint32(tlSteps[step])
						temp := e.coded[ch][gr][j][sb] + e.coded[ch][gr][j+1][sb]*y + e.coded[ch][gr][j+2][sb]*y*y
						b.putbits(temp, tlBitCount[step])
					}
				}
			}
		}
	}
}

func (e *tlEncoder) addDB(a, b float64) float64 {
	fdiff := 10.0 * (a - b)
	if fdiff > 990.0 {
		return a
	}
	if fdiff < -990.0 {
		return b
	}
	idiff := int(fdiff)
	if idiff >= 0 {
		return a + e.dbtable[idiff]
	}
	return b + e.dbtable[-idiff]
}

func (e *tlEncoder) psycho3() {
	var sample [tlBLKSIZE]float64
	var energy [tlBLKSIZE]float64
	var power [tlHBLKSIZE]float64
	var xtm, xnm [tlHBLKSIZE]float64
	var tonelabel, noiselabel [tlHBLKSIZE]int
	var ltg [tlSUBSIZE]float64
	var lsb [tlSBLIMIT]float64

	k := 0
	ok := e.psyOff[k] % tlFFTBUF
	for i := 0; i < tlSamplesPerFrame; i++ {
		e.fftBuf[k][ok] = float64(e.pcm[i]) / tlSCALE
		ok++
		if ok >= tlFFTBUF {
			ok = 0
		}
	}
	ok = (e.psyOff[k] + 1216) % tlFFTBUF
	for i := 0; i < tlBLKSIZE; i++ {
		sample[i] = e.fftBuf[k][ok]
		ok++
		if ok >= tlFFTBUF {
			ok = 0
		}
	}
	e.psyOff[k] += tlSamplesPerFrame
	e.psyOff[k] %= tlFFTBUF

	tlPsychoFFT(&sample, &energy)
	for i := 1; i < tlHBLKSIZE; i++ {
		if energy[i] < 1e-20 {
			power[i] = tlDBMIN + tlPOWERNORM
		} else {
			power[i] = 10*math.Log10(energy[i]) + tlPOWERNORM
		}
	}
	var xmax [tlSBLIMIT]float64
	for i := 0; i < tlSBLIMIT; i++ {
		xmax[i] = tlDBMIN
	}
	for i := 1; i < tlHBLKSIZE; i++ {
		index := (i - 1) >> 4
		if xmax[index] < power[i] {
			xmax[index] = power[i]
		}
	}
	for i := 0; i < tlSBLIMIT; i++ {
		val := 20*math.Log10(e.maxSC[k][i]*tlSCALE) - 10
		if xmax[i] > val {
			lsb[i] = xmax[i]
		} else {
			lsb[i] = val
		}
	}
	e.labelTones(&power, &tonelabel, &xtm)
	e.labelNoise(&power, &energy, &noiselabel, &xnm)
	for i := 1; i < tlHBLKSIZE; i++ {
		if noiselabel[i] == tlNOISE && xnm[i] < e.ath[i] {
			xnm[i] = tlDBMIN
			noiselabel[i] = 0
		}
		if tonelabel[i] == tlTONE && xtm[i] < e.ath[i] {
			xtm[i] = tlDBMIN
			tonelabel[i] = 0
		}
	}
	e.threshold(&ltg, &tonelabel, &xtm, &noiselabel, &xnm)
	var ltmin [tlSBLIMIT]float64
	for i := 0; i < tlSBLIMIT; i++ {
		ltmin[i] = 999999.9
	}
	for i := 0; i < tlSUBSIZE; i++ {
		index := e.freqSubset[i] >> 4
		if ltmin[index] > ltg[i] {
			ltmin[index] = ltg[i]
		}
	}
	for i := 0; i < tlSBLIMIT; i++ {
		e.smr[k][i] = lsb[i] - ltmin[i]
	}
}

func tlPsychoFFT(sample *[tlBLKSIZE]float64, energy *[tlBLKSIZE]float64) {
	var x [tlBLKSIZE]float64
	for i := 0; i < tlBLKSIZE; i++ {
		x[i] = sample[i] * tlHann()[i]
	}
	tlFHT(&x)
	energy[0] = x[0] * x[0]
	for i, j := 1, tlBLKSIZE-1; i < tlBLKSIZE/2; i, j = i+1, j-1 {
		a := x[i]
		b := x[j]
		energy[i] = (a*a + b*b) / 2.0
	}
	energy[tlBLKSIZE/2] = x[tlBLKSIZE/2] * x[tlBLKSIZE/2]
}

func (e *tlEncoder) labelTones(power *[tlHBLKSIZE]float64, tonelabel *[tlHBLKSIZE]int, xtm *[tlHBLKSIZE]float64) {
	var maxima [tlHBLKSIZE]int
	maxima[0] = 0
	maxima[tlHBLKSIZE-1] = 0
	tonelabel[0] = 0
	tonelabel[tlHBLKSIZE-1] = 0
	xtm[0] = tlDBMIN
	xtm[tlHBLKSIZE-1] = tlDBMIN
	for i := 1; i < tlHBLKSIZE-1; i++ {
		tonelabel[i] = 0
		xtm[i] = tlDBMIN
		if power[i] > power[i-1] && power[i] > power[i+1] {
			maxima[i] = 1
		} else {
			maxima[i] = 0
		}
	}
	e.labelToneRange(power, tonelabel, &maxima, xtm, 2, 63, 2)
	e.labelToneRange(power, tonelabel, &maxima, xtm, 63, 127, 3)
	e.labelToneRange(power, tonelabel, &maxima, xtm, 127, 255, 6)
	e.labelToneRange(power, tonelabel, &maxima, xtm, 255, 500, 12)
}

func (e *tlEncoder) labelToneRange(power *[tlHBLKSIZE]float64, tonelabel *[tlHBLKSIZE]int, maxima *[tlHBLKSIZE]int, xtm *[tlHBLKSIZE]float64, start, end, srange int) {
	for k := start; k < end; k++ {
		if maxima[k] != 1 {
			continue
		}
		tonelabel[k] = tlTONE
		for j := -srange; j <= srange; j++ {
			if j < -1 || j > 1 {
				if power[k]-power[k+j] < 7.0 {
					tonelabel[k] = 0
				}
			}
		}
		if tonelabel[k] == tlTONE {
			temp := e.addDB(power[k-1], power[k])
			xtm[k] = e.addDB(temp, power[k+1])
			for j := -srange; j <= srange; j++ {
				power[k+j] = tlDBMIN
			}
		}
	}
}

func (e *tlEncoder) labelNoise(power *[tlHBLKSIZE]float64, energy *[tlBLKSIZE]float64, noiselabel *[tlHBLKSIZE]int, xnm *[tlHBLKSIZE]float64) {
	xnm[0] = tlDBMIN
	for i := 0; i < e.cbands; i++ {
		sum := tlDBMIN
		esum := 0.0
		centreweight := 0.0
		start := e.cbandIndex[i]
		stop := e.cbandIndex[i+1]
		for j := start; j < stop; j++ {
			xnm[j] = tlDBMIN
			if power[j] != tlDBMIN {
				sum = e.addDB(power[j], sum)
				esum += energy[j]
				centreweight += float64(j-start) * energy[j]
			}
		}
		var centre int
		if sum <= tlDBMIN || esum < 0.00001 {
			centre = (start + stop) / 2
		} else {
			centre = start + int(centreweight/esum)
		}
		xnm[centre] = sum
		noiselabel[centre] = tlNOISE
	}
}

func (e *tlEncoder) threshold(ltg *[tlSUBSIZE]float64, tonelabel *[tlHBLKSIZE]int, xtm *[tlHBLKSIZE]float64, noiselabel *[tlHBLKSIZE]int, xnm *[tlHBLKSIZE]float64) {
	var lttm, ltnm [tlSUBSIZE]float64
	for i := 0; i < tlSUBSIZE; i++ {
		lttm[i] = tlDBMIN
		ltnm[i] = tlDBMIN
	}
	bitRate := e.bitrate // mono: bitrate/nch
	for k := 1; k < tlHBLKSIZE; k++ {
		if tonelabel[k] == tlTONE {
			for j := 0; j < tlSUBSIZE; j++ {
				dz := e.bark[e.freqSubset[j]] - e.bark[k]
				if dz >= -3.0 && dz < 8.0 {
					av := -1.525 - 0.275*e.bark[k] - 4.5 + xtm[k]
					var vf float64
					if dz < -1 {
						vf = 17*(dz+1) - (0.4*xtm[k] + 6)
					} else if dz < 0 {
						vf = (0.4*xtm[k] + 6) * dz
					} else if dz < 1 {
						vf = -17 * dz
					} else {
						vf = -(dz-1)*(17-0.15*xtm[k]) - 17
					}
					lttm[j] = e.addDB(lttm[j], av+vf)
				}
			}
		}
		if noiselabel[k] == tlNOISE {
			for j := 0; j < tlSUBSIZE; j++ {
				dz := e.bark[e.freqSubset[j]] - e.bark[k]
				if dz >= -3.0 && dz < 8.0 {
					av := -1.525 - 0.175*e.bark[k] - 0.5 + xnm[k]
					var vf float64
					if dz < -1 {
						vf = 17*(dz+1) - (0.4*xnm[k] + 6)
					} else if dz < 0 {
						vf = (0.4*xnm[k] + 6) * dz
					} else if dz < 1 {
						vf = -17 * dz
					} else {
						vf = -(dz-1)*(17-0.15*xnm[k]) - 17
					}
					ltnm[j] = e.addDB(ltnm[j], av+vf)
				}
			}
		}
	}
	for i := 0; i < tlSUBSIZE; i++ {
		ltg[i] = e.addDB(ltnm[i], lttm[i])
		if bitRate < 96 {
			ltg[i] = e.addDB(e.ath[e.freqSubset[i]], ltg[i])
		} else {
			ltg[i] = e.addDB(e.ath[e.freqSubset[i]]-12.0, ltg[i])
		}
	}
}

func tlFHT(fz *[tlBLKSIZE]float64) {
	for i := 0; i < len(tlK1K2); i++ {
		k1 := tlK1K2[i][0]
		k2 := tlK1K2[i][1]
		fz[k1], fz[k2] = fz[k2], fz[k1]
	}
	for fi := 0; fi < tlBLKSIZE; fi += 4 {
		f1 := fz[fi] - fz[fi+1]
		f0 := fz[fi] + fz[fi+1]
		f3 := fz[fi+2] - fz[fi+3]
		f2 := fz[fi+2] + fz[fi+3]
		fz[fi+2] = f0 - f2
		fz[fi] = f0 + f2
		fz[fi+3] = f1 - f3
		fz[fi+1] = f1 + f3
	}
	k := 0
	k4 := 0
	for {
		k += 2
		k1 := 1 << uint(k)
		k2 := k1 << 1
		k4 = k2 << 1
		k3 := k2 + k1
		kx := k1 >> 1
		fi := 0
		gi := kx
		for fi < tlBLKSIZE {
			f1 := fz[fi] - fz[fi+k1]
			f0 := fz[fi] + fz[fi+k1]
			f3 := fz[fi+k2] - fz[fi+k3]
			f2 := fz[fi+k2] + fz[fi+k3]
			fz[fi+k2] = f0 - f2
			fz[fi] = f0 + f2
			fz[fi+k3] = f1 - f3
			fz[fi+k1] = f1 + f3
			g1 := fz[gi] - fz[gi+k1]
			g0 := fz[gi] + fz[gi+k1]
			g3 := tlSqrt2 * fz[gi+k3]
			g2 := tlSqrt2 * fz[gi+k2]
			fz[gi+k2] = g0 - g2
			fz[gi] = g0 + g2
			fz[gi+k3] = g1 - g3
			fz[gi+k1] = g1 + g3
			gi += k4
			fi += k4
		}
		tC := tlCosTab[k]
		tS := tlSinTab[k]
		c1 := 1.0
		s1 := 0.0
		for i := 1; i < kx; i++ {
			t := c1
			c1 = t*tC - s1*tS
			s1 = t*tS + s1*tC
			c2 := c1*c1 - s1*s1
			s2 := 2 * (c1 * s1)
			fi = i
			gi = k1 - i
			for fi < tlBLKSIZE {
				b := s2*fz[fi+k1] - c2*fz[gi+k1]
				a := c2*fz[fi+k1] + s2*fz[gi+k1]
				f1 := fz[fi] - a
				f0 := fz[fi] + a
				g1 := fz[gi] - b
				g0 := fz[gi] + b
				b = s2*fz[fi+k3] - c2*fz[gi+k3]
				a = c2*fz[fi+k3] + s2*fz[gi+k3]
				f3 := fz[fi+k2] - a
				f2 := fz[fi+k2] + a
				g3 := fz[gi+k2] - b
				g2 := fz[gi+k2] + b
				b = s1*f2 - c1*g3
				a = c1*f2 + s1*g3
				fz[fi+k2] = f0 - a
				fz[fi] = f0 + a
				fz[gi+k3] = g1 - b
				fz[gi+k1] = g1 + b
				b = c1*g2 - s1*f3
				a = s1*g2 + c1*f3
				fz[gi+k2] = g0 - a
				fz[gi] = g0 + a
				fz[fi+k3] = f1 - b
				fz[fi+k1] = f1 + b
				gi += k4
				fi += k4
			}
		}
		if k4 >= tlBLKSIZE {
			break
		}
	}
}
