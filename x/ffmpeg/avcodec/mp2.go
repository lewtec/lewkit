package avcodec

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/bits"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

const (
	mp2FrameSamples = 1152
	mp2History      = 4096
	mp2FracBits     = 15
	mp2WindowShift  = 14 // WFRAC_BITS + 15 - FRAC_BITS
	mp2ScaleP       = 15
	mp2SqrtHalf     = 23170 // FIX(M_SQRT2/2) at 15 fractional bits
)

// mp2Codec is the fixed-point MPEG-1/2 Layer II encoder (mp2fixed).
// The analysis window starts from a zeroed 512-sample history, which is
// the delay FFmpeg reports as initial_padding 481. Input is not pre-padded.
type mp2Codec struct {
	par      Parameters
	ready    bool
	err      error
	channels int
	rate     int

	lsf           int
	bitrateIndex  int
	freqIndex     int
	frameBits     int
	frameFrac     int
	frameFracIncr int
	doPadding     int
	sblimit       int
	alloc         []byte

	history   [2][mp2History]int16
	historyAt [2]int
	bank      [512]int16

	scaleTab   [64]int32
	scaleShift [64]int8
	scaleMult  [64]uint16
	scaleDiff  [128]byte
	totalBits  [17]int

	pending []int16
	pts     int64
}

func newMP2(par Parameters) *mp2Codec {
	codec := &mp2Codec{par: par}
	if par.SampleRate != 0 && par.Layout.Channels() != 0 {
		codec.err = codec.configure(par.SampleRate, par.Layout.Channels(), par.BitRate)
	}
	return codec
}

func (c *mp2Codec) configure(rate, channels, bitRate int) error {
	if channels < 1 || channels > 2 {
		return fmt.Errorf("%w: mp2 channels %d", avutil.ErrUnsupported, channels)
	}
	lsf := 0
	freqIndex := -1
	for i, freq := range mp2Freq {
		if freq == rate {
			freqIndex = i
			break
		}
		if freq/2 == rate {
			lsf = 1
			freqIndex = i
			break
		}
	}
	if freqIndex < 0 {
		return fmt.Errorf("%w: mp2 rate %d", avutil.ErrUnsupported, rate)
	}
	kbps := bitRate / 1000
	bitrateIndex := 15
	for i := 1; i < 15; i++ {
		if mp2Bitrate[lsf][1][i] == kbps {
			bitrateIndex = i
			break
		}
	}
	if bitrateIndex == 15 && bitRate == 0 {
		bitrateIndex = 14
		kbps = mp2Bitrate[lsf][1][bitrateIndex]
	}
	if bitrateIndex == 15 {
		return fmt.Errorf("%w: mp2 bitrate %d", avutil.ErrUnsupported, bitRate)
	}
	table := mp2Layer2Table(kbps, channels, rate, lsf)
	num := float32(kbps * 1000 * mp2FrameSamples)
	frame := float32(float64(num) / (float64(rate) * 8.0))
	c.channels = channels
	c.rate = rate
	c.lsf = lsf
	c.bitrateIndex = bitrateIndex
	c.freqIndex = freqIndex
	c.frameBits = int(frame) * 8
	c.frameFrac = 0
	c.frameFracIncr = int((float64(frame) - math.Floor(float64(frame))) * 65536.0)
	c.sblimit = mp2SBLimit[table]
	c.alloc = mp2Alloc[table]
	c.buildTables()
	c.ready = true
	c.err = nil
	return nil
}

func mp2Layer2Table(bitrate, channels, freq, lsf int) int {
	perChannel := bitrate / channels
	if lsf != 0 {
		return 4
	}
	if (freq == 48000 && perChannel >= 56) || (perChannel >= 56 && perChannel <= 80) {
		return 0
	}
	if freq != 48000 && perChannel >= 96 {
		return 1
	}
	if freq != 32000 && perChannel <= 48 {
		return 2
	}
	return 3
}

func (c *mp2Codec) buildTables() {
	for i := 0; i < 257; i++ {
		value := (mp2Enwindow[i] + 2) >> 2
		c.bank[i] = int16(value)
		if i&63 != 0 {
			value = -value
		}
		if i != 0 {
			c.bank[512-i] = int16(value)
		}
	}
	for i := 0; i < 64; i++ {
		value := int32(math.Exp2(float64(3-i)/3.0) * float64(int32(1<<20)))
		if value <= 0 {
			value = 1
		}
		c.scaleTab[i] = value
		c.scaleShift[i] = int8(21 - mp2ScaleP - i/3)
		c.scaleMult[i] = uint16(float64(int32(1<<mp2ScaleP)) * math.Exp2(float64(i%3)/3.0))
	}
	for i := 0; i < 128; i++ {
		value := i - 64
		switch {
		case value <= -3:
			value = 0
		case value < 0:
			value = 1
		case value == 0:
			value = 2
		case value < 3:
			value = 3
		default:
			value = 4
		}
		c.scaleDiff[i] = byte(value)
	}
	for i := 0; i < 17; i++ {
		bitsPerGroup := mp2QuantBits[i]
		if bitsPerGroup < 0 {
			bitsPerGroup = -bitsPerGroup
		} else {
			bitsPerGroup *= 3
		}
		c.totalBits[i] = 12 * bitsPerGroup
	}
}

func (c *mp2Codec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if c.err != nil {
		return nil, c.err
	}
	if frame == nil || frame.NbSamples == 0 {
		return nil, nil
	}
	if err := c.prepare(frame); err != nil {
		return nil, err
	}
	samples, err := c.readSamples(frame)
	if err != nil {
		return nil, err
	}
	c.pending = append(c.pending, samples...)
	return c.emit(false)
}

func (c *mp2Codec) Flush() ([]*avutil.Packet, error) {
	if c.err != nil {
		return nil, c.err
	}
	if !c.ready || len(c.pending) == 0 {
		return nil, nil
	}
	return c.emit(true)
}

func (c *mp2Codec) prepare(frame *avutil.Frame) error {
	if frame.SampleFmt != avutil.SampleS16 {
		return fmt.Errorf("%w: mp2 %s", avutil.ErrUnsupported, frame.SampleFmt)
	}
	if !c.ready {
		rate := c.par.SampleRate
		if rate == 0 {
			rate = frame.SampleRate
		}
		channels := c.par.Layout.Channels()
		if channels == 0 {
			channels = frame.Layout.Channels()
		}
		if frame.SampleRate != 0 && c.par.SampleRate != 0 && frame.SampleRate != c.par.SampleRate {
			return fmt.Errorf("%w: mp2 sample rate", avutil.ErrInvalidData)
		}
		if frame.Layout.Channels() != 0 && c.par.Layout.Channels() != 0 && frame.Layout.Channels() != channels {
			return fmt.Errorf("%w: mp2 channels", avutil.ErrInvalidData)
		}
		if err := c.configure(rate, channels, c.par.BitRate); err != nil {
			c.err = err
			return err
		}
		return nil
	}
	if frame.SampleRate != 0 && frame.SampleRate != c.rate {
		return fmt.Errorf("%w: mp2 sample rate", avutil.ErrInvalidData)
	}
	if frame.Layout.Channels() != 0 && frame.Layout.Channels() != c.channels {
		return fmt.Errorf("%w: mp2 channels", avutil.ErrInvalidData)
	}
	return nil
}

func (c *mp2Codec) readSamples(frame *avutil.Frame) ([]int16, error) {
	count := frame.NbSamples * c.channels
	if frame.NbSamples < 0 || len(frame.Data[0]) < count*2 {
		return nil, fmt.Errorf("%w: mp2 frame", avutil.ErrInvalidData)
	}
	out := make([]int16, count)
	for i := range out {
		out[i] = int16(binary.LittleEndian.Uint16(frame.Data[0][i*2:]))
	}
	return out, nil
}

func (c *mp2Codec) emit(pad bool) ([]*avutil.Packet, error) {
	need := mp2FrameSamples * c.channels
	var packets []*avutil.Packet
	for len(c.pending) >= need {
		packet, err := c.encodeFrame(c.pending[:need])
		if err != nil {
			return nil, err
		}
		c.pending = c.pending[need:]
		packets = append(packets, packet)
	}
	if pad && len(c.pending) > 0 {
		buf := make([]int16, need)
		copy(buf, c.pending)
		c.pending = nil
		packet, err := c.encodeFrame(buf)
		if err != nil {
			return nil, err
		}
		packets = append(packets, packet)
	}
	return packets, nil
}

func (c *mp2Codec) encodeFrame(samples []int16) (*avutil.Packet, error) {
	var subbands [2][36][32]int32
	var scaleCode [2][32]byte
	var scaleFactors [2][32][3]byte
	var smr [2][32]int16
	for ch := 0; ch < c.channels; ch++ {
		c.analyze(ch, samples[ch:], subbands[ch][:])
		scaleCode[ch], scaleFactors[ch] = c.scaleFactors(subbands[ch][:])
		for band := 0; band < c.sblimit; band++ {
			smr[ch][band] = mp2SMR[band]
		}
	}
	allocation, padding := c.allocate(smr, scaleCode)
	var writer mp2Bits
	c.writeFrame(&writer, subbands, scaleCode, scaleFactors, allocation, padding)
	packet := &avutil.Packet{
		Data:     writer.bytes(),
		PTS:      c.pts,
		Duration: mp2FrameSamples,
		TimeBase: avutil.Q(1, int32(c.rate)),
		Key:      true,
	}
	c.pts += mp2FrameSamples
	return packet, nil
}

func (c *mp2Codec) analyze(ch int, samples []int16, subbands [][32]int32) {
	at := c.historyAt[ch]
	pos := 0
	for block := 0; block < 36; block++ {
		for i := 0; i < 32; i++ {
			c.history[ch][at+(31-i)] = samples[pos]
			pos += c.channels
		}
		var windowed [64]int32
		for i := 0; i < 64; i++ {
			var sum int32
			for tap := 0; tap < 8; tap++ {
				sum += int32(c.history[ch][at+i+tap*64]) * int32(c.bank[i+tap*64])
			}
			windowed[i] = sum
		}
		var folded [32]int32
		folded[0] = windowed[16] >> mp2WindowShift
		for i := 1; i <= 16; i++ {
			folded[i] = (windowed[i+16] + windowed[16-i]) >> mp2WindowShift
		}
		for i := 17; i <= 31; i++ {
			folded[i] = (windowed[i+16] - windowed[80-i]) >> mp2WindowShift
		}
		mp2IDCT(subbands[block][:], folded[:])
		at -= 32
		if at < 0 {
			copy(c.history[ch][mp2History-(512-32):], c.history[ch][:512-32])
			at = mp2History - 512
		}
	}
	c.historyAt[ch] = at
}

func mp2IDCT(out, tab []int32) {
	for j := 31; j >= 3; j -= 2 {
		tab[j] += tab[j-2]
	}
	for t := 30; t != 2; t -= 4 {
		tab[t] += tab[t-4]
		tab[t+1] += tab[t-3]
	}
	for t := 28; t != 4; t -= 8 {
		tab[t] += tab[t-8]
		tab[t+1] += tab[t-7]
		tab[t+2] += tab[t-6]
		tab[t+3] += tab[t-5]
	}
	for t := 0; t < 32; t += 16 {
		tab[t+3] = -tab[t+3]
		tab[t+6] = -tab[t+6]
		tab[t+11] = -tab[t+11]
		tab[t+12] = -tab[t+12]
		tab[t+13] = -tab[t+13]
		tab[t+15] = -tab[t+15]
	}
	for t := 0; t < 8; t++ {
		x3 := mp2Mul(tab[t+16], mp2SqrtHalf)
		x4 := tab[t] - x3
		x3 = tab[t] + x3
		sum := tab[t+24] + tab[t+8]
		x2 := mp2Mul(-sum, mp2SqrtHalf)
		x1 := mp2Mul(tab[t+8]-x2, mp2Costab[0])
		x2 = mp2Mul(tab[t+8]+x2, mp2Costab[1])
		tab[t] = x3 + x1
		tab[t+8] = x4 - x2
		tab[t+16] = x4 + x2
		tab[t+24] = x3 - x1
	}
	coef := 2
	for t := 0; t < 4; t++ {
		xr := mp2Mul(tab[t+28], mp2Costab[coef])
		tab[t+28] = tab[t] - xr
		tab[t] = tab[t] + xr
		xr = mp2Mul(tab[t+4], mp2Costab[coef+1])
		tab[t+4] = tab[t+24] - xr
		tab[t+24] = tab[t+24] + xr
		xr = mp2Mul(tab[t+20], mp2Costab[coef+2])
		tab[t+20] = tab[t+8] - xr
		tab[t+8] = tab[t+8] + xr
		xr = mp2Mul(tab[t+12], mp2Costab[coef+3])
		tab[t+12] = tab[t+16] - xr
		tab[t+16] = tab[t+16] + xr
	}
	coef = 6
	for i := 0; i < 4; i++ {
		xr := mp2Mul(tab[30-i*4], mp2Costab[coef])
		tab[30-i*4] = tab[i*4] - xr
		tab[i*4] = tab[i*4] + xr
		xr = mp2Mul(tab[2+i*4], mp2Costab[coef+1])
		tab[2+i*4] = tab[28-i*4] - xr
		tab[28-i*4] = tab[28-i*4] + xr
		xr = mp2Mul(tab[31-i*4], mp2Costab[coef])
		tab[31-i*4] = tab[1+i*4] - xr
		tab[1+i*4] = tab[1+i*4] + xr
		xr = mp2Mul(tab[3+i*4], mp2Costab[coef+1])
		tab[3+i*4] = tab[29-i*4] - xr
		tab[29-i*4] = tab[29-i*4] + xr
		coef += 2
	}
	hi := 30
	lo := 1
	for hi >= 0 {
		xr := mp2Mul(tab[lo], mp2Costab[coef])
		tab[lo] = tab[hi] - xr
		tab[hi] = tab[hi] + xr
		hi -= 2
		lo += 2
		coef++
	}
	for i := 0; i < 32; i++ {
		out[i] = tab[mp2BitRev[i]]
	}
}

func mp2Mul(a, b int32) int32 {
	return int32((int64(a) * int64(b)) >> mp2FracBits)
}

func (c *mp2Codec) scaleFactors(subbands [][32]int32) ([32]byte, [32][3]byte) {
	var code [32]byte
	var factors [32][3]byte
	for band := 0; band < c.sblimit; band++ {
		var factor [3]byte
		for granule := 0; granule < 3; granule++ {
			peak := mp2Abs(subbands[granule*12][band])
			for sample := 1; sample < 12; sample++ {
				value := mp2Abs(subbands[granule*12+sample][band])
				if value > peak {
					peak = value
				}
			}
			index := 62
			if peak > 1 {
				msb := bits.Len32(uint32(peak)) - 1
				index = (21-msb)*3 - 3
				if index >= 0 {
					for peak <= c.scaleTab[index+1] {
						index++
					}
				} else {
					index = 0
				}
			}
			factor[granule] = byte(index)
		}
		d1 := c.scaleDiff[int(factor[0])-int(factor[1])+64]
		d2 := c.scaleDiff[int(factor[1])-int(factor[2])+64]
		var pattern byte
		switch int(d1)*5 + int(d2) {
		case 0*5 + 0, 0*5 + 4, 3*5 + 4, 4*5 + 0, 4*5 + 4:
			pattern = 0
		case 0*5 + 1, 0*5 + 2, 4*5 + 1, 4*5 + 2:
			pattern = 3
			factor[2] = factor[1]
		case 0*5 + 3, 4*5 + 3:
			pattern = 3
			factor[1] = factor[2]
		case 1*5 + 0, 1*5 + 4, 2*5 + 4:
			pattern = 1
			factor[1] = factor[0]
		case 1*5 + 1, 1*5 + 2, 2*5 + 0, 2*5 + 1, 2*5 + 2:
			pattern = 2
			factor[1] = factor[0]
			factor[2] = factor[0]
		case 2*5 + 3, 3*5 + 3:
			pattern = 2
			factor[0] = factor[2]
			factor[1] = factor[2]
		case 3*5 + 0, 3*5 + 1, 3*5 + 2:
			pattern = 2
			factor[0] = factor[1]
			factor[2] = factor[1]
		case 1*5 + 3:
			pattern = 2
			if factor[0] > factor[2] {
				factor[0] = factor[2]
			}
			factor[1] = factor[0]
			factor[2] = factor[0]
		default:
			pattern = 0
		}
		code[band] = pattern
		factors[band] = factor
	}
	return code, factors
}

func mp2Abs(value int32) int32 {
	if value < 0 {
		return -value
	}
	return value
}

func (c *mp2Codec) allocate(smrIn [2][32]int16, scaleCode [2][32]byte) ([2][32]byte, int) {
	var smr [2][32]int16
	var status [2][32]byte
	var allocation [2][32]byte
	for ch := 0; ch < c.channels; ch++ {
		smr[ch] = smrIn[ch]
	}
	maxFrame := c.frameBits
	c.frameFrac += c.frameFracIncr
	if c.frameFrac >= 65536 {
		c.frameFrac -= 65536
		c.doPadding = 1
		maxFrame += 8
	} else {
		c.doPadding = 0
	}
	current := 32
	offset := 0
	for band := 0; band < c.sblimit; band++ {
		width := int(c.alloc[offset])
		current += width * c.channels
		offset += 1 << width
	}
	for {
		bestBand := -1
		bestChannel := -1
		best := math.MinInt32
		for ch := 0; ch < c.channels; ch++ {
			for band := 0; band < c.sblimit; band++ {
				if int(smr[ch][band]) > best && status[ch][band] != 2 {
					best = int(smr[ch][band])
					bestBand = band
					bestChannel = ch
				}
			}
		}
		if bestBand < 0 {
			break
		}
		offset = 0
		for band := 0; band < bestBand; band++ {
			offset += 1 << int(c.alloc[offset])
		}
		var extra int
		if status[bestChannel][bestBand] == 0 {
			extra = 2 + mp2ScaleCount[scaleCode[bestChannel][bestBand]]*6
			extra += c.totalBits[c.alloc[offset+1]]
		} else {
			level := int(allocation[bestChannel][bestBand])
			extra = c.totalBits[c.alloc[offset+level+1]] - c.totalBits[c.alloc[offset+level]]
		}
		if current+extra <= maxFrame {
			allocation[bestChannel][bestBand]++
			level := int(allocation[bestChannel][bestBand])
			current += extra
			smr[bestChannel][bestBand] = smrIn[bestChannel][bestBand] - int16(mp2QuantSNR[c.alloc[offset+level]])
			if level == (1<<int(c.alloc[offset]))-1 {
				status[bestChannel][bestBand] = 2
			} else {
				status[bestChannel][bestBand] = 1
			}
		} else {
			status[bestChannel][bestBand] = 2
		}
	}
	return allocation, maxFrame - current
}

func (c *mp2Codec) writeFrame(w *mp2Bits, subbands [2][36][32]int32, scaleCode [2][32]byte, scaleFactors [2][32][3]byte, allocation [2][32]byte, padding int) {
	w.u(0xfff, 12)
	w.u(uint32(1-c.lsf), 1)
	w.u(2, 2)
	w.u(1, 1)
	w.u(uint32(c.bitrateIndex), 4)
	w.u(uint32(c.freqIndex), 2)
	w.u(uint32(c.doPadding), 1)
	w.u(0, 1)
	if c.channels == 2 {
		w.u(0, 2)
	} else {
		w.u(3, 2)
	}
	w.u(0, 2)
	w.u(0, 1)
	w.u(1, 1)
	w.u(0, 2)

	offset := 0
	for band := 0; band < c.sblimit; band++ {
		width := int(c.alloc[offset])
		for ch := 0; ch < c.channels; ch++ {
			w.u(uint32(allocation[ch][band]), width)
		}
		offset += 1 << width
	}
	for band := 0; band < c.sblimit; band++ {
		for ch := 0; ch < c.channels; ch++ {
			if allocation[ch][band] != 0 {
				w.u(uint32(scaleCode[ch][band]), 2)
			}
		}
	}
	for band := 0; band < c.sblimit; band++ {
		for ch := 0; ch < c.channels; ch++ {
			if allocation[ch][band] == 0 {
				continue
			}
			factor := scaleFactors[ch][band]
			switch scaleCode[ch][band] {
			case 0:
				w.u(uint32(factor[0]), 6)
				w.u(uint32(factor[1]), 6)
				w.u(uint32(factor[2]), 6)
			case 1, 3:
				w.u(uint32(factor[0]), 6)
				w.u(uint32(factor[2]), 6)
			case 2:
				w.u(uint32(factor[0]), 6)
			}
		}
	}
	for granule := 0; granule < 3; granule++ {
		for group := 0; group < 12; group += 3 {
			offset = 0
			for band := 0; band < c.sblimit; band++ {
				width := int(c.alloc[offset])
				for ch := 0; ch < c.channels; ch++ {
					level := int(allocation[ch][band])
					if level == 0 {
						continue
					}
					quant := int(c.alloc[offset+level])
					steps := mp2QuantSteps[quant]
					var coded [3]int
					for sample := 0; sample < 3; sample++ {
						value := subbands[ch][granule*12+group+sample][band]
						shift := c.scaleShift[scaleFactors[ch][band][granule]]
						mult := int32(c.scaleMult[scaleFactors[ch][band][granule]])
						var scaled int32
						if shift < 0 {
							scaled = value * (1 << int(-shift))
						} else {
							scaled = value >> int(shift)
						}
						scaled = (scaled * mult) >> mp2ScaleP
						scaled += 1 << mp2ScaleP
						if scaled < 0 {
							scaled = 0
						}
						code := int((uint32(scaled) * uint32(steps)) >> (mp2ScaleP + 1))
						if code >= steps {
							code = steps - 1
						}
						coded[sample] = code
					}
					bits := mp2QuantBits[quant]
					if bits < 0 {
						packed := coded[0] + steps*(coded[1]+steps*coded[2])
						w.u(uint32(packed), -bits)
					} else {
						w.u(uint32(coded[0]), bits)
						w.u(uint32(coded[1]), bits)
						w.u(uint32(coded[2]), bits)
					}
				}
				offset += 1 << width
			}
		}
	}
	for padding >= 16 {
		w.u(0, 16)
		padding -= 16
	}
	if padding > 0 {
		w.u(0, padding)
	}
}

// mp2Bits writes MPEG audio bits, most significant bit first.
type mp2Bits struct {
	buf []byte
	acc uint32
	n   int
}

func (w *mp2Bits) u(value uint32, n int) {
	for n > 0 {
		take := 8 - w.n
		if take > n {
			take = n
		}
		shift := n - take
		w.acc = (w.acc << take) | ((value >> shift) & ((1 << take) - 1))
		w.n += take
		n -= take
		if w.n == 8 {
			w.buf = append(w.buf, byte(w.acc))
			w.acc = 0
			w.n = 0
		}
	}
}

func (w *mp2Bits) bytes() []byte {
	if w.n > 0 {
		w.buf = append(w.buf, byte(w.acc<<(8-w.n)))
		w.acc = 0
		w.n = 0
	}
	return w.buf
}
