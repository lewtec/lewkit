package avcodec

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

const (
	shineGranule     = 576
	shineSBLimit     = 32
	shineHanSize     = 512
	shineMPEGI       = 3
	shineMPEGII      = 2
	shineMPEG25      = 0
	shineLayerIII    = 1
	shineModeMono    = 3
	shineMaxChannels = 2
	shineMaxGranules = 2
	shineLN2         = 0.69314718
)

// shineFFmpegID3 is the ID3v2.4 TSSE tag the mp3 muxer writes on a pipe
// (Lavf59.27.100, ten bytes of padding). Xing is omitted because a pipe
// is not seekable.
var shineFFmpegID3 = []byte{
	0x49, 0x44, 0x33, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x23,
	0x54, 0x53, 0x53, 0x45, 0x00, 0x00, 0x00, 0x0f, 0x00, 0x00,
	0x03, 0x4c, 0x61, 0x76, 0x66, 0x35, 0x39, 0x2e, 0x32, 0x37,
	0x2e, 0x31, 0x30, 0x30, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00,
}

var shineSamplerates = [9]int{44100, 48000, 32000, 22050, 24000, 16000, 11025, 12000, 8000}

// Columns are MPEG-2.5, reserved, MPEG-II, MPEG-I.
var shineBitrates = [16][4]int{
	{-1, -1, -1, -1},
	{8, -1, 8, 32},
	{16, -1, 16, 40},
	{24, -1, 24, 48},
	{32, -1, 32, 56},
	{40, -1, 40, 64},
	{48, -1, 48, 80},
	{56, -1, 56, 96},
	{64, -1, 64, 112},
	{-1, -1, 80, 128},
	{-1, -1, 96, 160},
	{-1, -1, 112, 192},
	{-1, -1, 128, 224},
	{-1, -1, 144, 256},
	{-1, -1, 160, 320},
	{-1, -1, -1, -1},
}

var shineGranulesForVersion = [4]int{1, -1, 1, 2}

var shineSlen1 = [16]int{0, 0, 0, 0, 3, 1, 1, 1, 2, 2, 2, 3, 3, 3, 4, 4}
var shineSlen2 = [16]int{0, 1, 2, 3, 0, 1, 2, 3, 1, 2, 3, 1, 2, 3, 2, 3}

var shineScaleBands = [9][23]int{
	{0, 4, 8, 12, 16, 20, 24, 30, 36, 44, 52, 62, 74, 90, 110, 134, 162, 196, 238, 288, 342, 418, 576},
	{0, 4, 8, 12, 16, 20, 24, 30, 36, 42, 50, 60, 72, 88, 106, 128, 156, 190, 230, 276, 330, 384, 576},
	{0, 4, 8, 12, 16, 20, 24, 30, 36, 44, 54, 66, 82, 102, 126, 156, 194, 240, 296, 364, 448, 550, 576},
	{0, 6, 12, 18, 24, 30, 36, 44, 54, 66, 80, 96, 116, 140, 168, 200, 238, 284, 336, 396, 464, 522, 576},
	{0, 6, 12, 18, 24, 30, 36, 44, 54, 66, 80, 96, 114, 136, 162, 194, 232, 278, 330, 394, 464, 540, 576},
	{0, 6, 12, 18, 24, 30, 36, 44, 45, 66, 80, 96, 116, 140, 168, 200, 238, 248, 336, 396, 464, 522, 576},
	{0, 6, 12, 18, 24, 30, 36, 44, 54, 66, 80, 96, 116, 140, 168, 200, 238, 284, 336, 396, 464, 522, 576},
	{0, 6, 12, 18, 24, 30, 36, 44, 54, 66, 80, 96, 116, 140, 168, 200, 238, 284, 336, 396, 464, 522, 576},
	{0, 12, 24, 36, 48, 60, 72, 88, 108, 132, 160, 192, 232, 280, 336, 400, 476, 566, 568, 570, 572, 574, 576},
}

var shineSCFSIBand = [5]int{0, 6, 11, 16, 21}

var shineSubdivide = [23]struct{ region0, region1 int }{
	{0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 1}, {1, 1}, {1, 1},
	{1, 2}, {2, 2}, {2, 3}, {2, 3}, {3, 4}, {3, 4}, {3, 4}, {4, 5},
	{4, 5}, {4, 6}, {5, 6}, {5, 6}, {5, 7}, {6, 7}, {6, 7},
}

var shineMPAFreq = [3]int{44100, 48000, 32000}

var shineMPABitrate = [2][3][15]int{
	{
		{0, 32, 64, 96, 128, 160, 192, 224, 256, 288, 320, 352, 384, 416, 448},
		{0, 32, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384},
		{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320},
	},
	{
		{0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256},
		{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},
		{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},
	},
}

// shineCodec encodes mono MP3 with libshine's fixed-point encoder.
// The packet is the pipe mp3 file: the lavf ID3 tag plus complete frames.
type shineCodec struct {
	pcm  []int16
	rate int
	kbps int
}

func newShineCodec(par Parameters) (*shineCodec, error) {
	rate := par.SampleRate
	if rate <= 0 {
		rate = 44100
	}
	kbps := 128
	if par.BitRate > 0 {
		kbps = par.BitRate / 1000
	}
	if shineFindSamplerate(rate) < 0 {
		return nil, fmt.Errorf("%w: shine rate %d", avutil.ErrUnsupported, rate)
	}
	version := shineMPEGVersion(shineFindSamplerate(rate))
	if shineFindBitrate(kbps, version) < 0 {
		return nil, fmt.Errorf("%w: shine bitrate %d", avutil.ErrUnsupported, kbps)
	}
	return &shineCodec{rate: rate, kbps: kbps}, nil
}

func (c *shineCodec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil || frame.NbSamples == 0 {
		return nil, nil
	}
	if frame.SampleFmt != avutil.SampleS16 || frame.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: shine", avutil.ErrInvalidData)
	}
	if len(frame.Data) == 0 || len(frame.Data[0]) < frame.NbSamples*2 {
		return nil, fmt.Errorf("%w: shine samples", avutil.ErrInvalidData)
	}
	for i := 0; i < frame.NbSamples; i++ {
		c.pcm = append(c.pcm, int16(binary.LittleEndian.Uint16(frame.Data[0][i*2:])))
	}
	return nil, nil
}

func (c *shineCodec) Flush() ([]*avutil.Packet, error) {
	n := len(c.pcm)
	raw, err := ShineEncodeMono(c.pcm, c.rate, c.kbps)
	c.pcm = nil
	if err != nil || len(raw) == 0 {
		return nil, err
	}
	return []*avutil.Packet{{Data: raw, Duration: int64(n)}}, nil
}

// ShineEncodeMono encodes mono PCM the way libshine 3.1.1 does for
// shine_set_config_mpeg_defaults, mode MONO, then shine_encode_buffer
// and shine_flush. The bytes match ffmpeg -c:a libshine -f mp3 on a pipe:
// the lavf ID3 tag followed by every complete MPEG frame the libshine
// wrapper would emit.
func ShineEncodeMono(pcm []int16, rate, bitrateKbps int) ([]byte, error) {
	enc, err := newShineEncoder(rate, bitrateKbps)
	if err != nil {
		return nil, err
	}
	samplesPerFrame := enc.granules * shineGranule
	frames := 0
	if len(pcm) > 0 {
		frames = (len(pcm) + samplesPerFrame - 1) / samplesPerFrame
	}
	var stream []byte
	for frame := 0; frame < frames; frame++ {
		var block [1152]int16
		start := frame * samplesPerFrame
		end := start + samplesPerFrame
		if end > len(pcm) {
			end = len(pcm)
		}
		copy(block[:], pcm[start:end])
		stream = append(stream, enc.encodeFrame(block[:samplesPerFrame])...)
	}
	stream = append(stream, enc.flush()...)
	audio, err := shineCompleteFrames(stream, frames)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(shineFFmpegID3)+len(audio))
	out = append(out, shineFFmpegID3...)
	out = append(out, audio...)
	return out, nil
}

type shineGR struct {
	part23Length      uint32
	bigValues         uint32
	count1            uint32
	globalGain        uint32
	scalefacCompress  uint32
	tableSelect       [3]uint32
	region0Count      uint32
	region1Count      uint32
	preflag           uint32
	scalefacScale     uint32
	count1TableSelect uint32
	part2Length       uint32
	sfbLMax           uint32
	address1          uint32
	address2          uint32
	address3          uint32
	quantizerStepSize int
	slen              [4]uint32
}

type shineBits struct {
	buf   []byte
	pos   int
	cache uint32
	free  int
}

type shineEnc struct {
	channels        int
	sampleRate      int
	bitrate         int
	version         int
	granules        int
	mode            int
	emphasis        int
	copyright       int
	original        int
	padding         int
	bitsPerFrame    int
	bitsPerSlot     int
	fracSlots       float64
	slotLag         float64
	wholeSlots      int
	bitrateIndex    int
	sampleRateIndex int
	sideInfoLen     int
	meanBits        int
	resvSize        int
	resvMax         int

	bs          shineBits
	privateBits uint32
	scfsi       [shineMaxChannels][4]uint32
	gr          [shineMaxGranules][shineMaxChannels]shineGR

	l3Enc    [shineMaxChannels][shineMaxGranules][shineGranule]int
	sbSample [shineMaxChannels][shineMaxGranules + 1][18][shineSBLimit]int32
	mdctFreq [shineMaxChannels][shineMaxGranules][shineGranule]int32

	xr     []int32
	xrsq   [shineGranule]int32
	xrabs  [shineGranule]int32
	xrmax  int32
	enTot  [shineMaxGranules]int32
	en     [shineMaxGranules][21]int32
	xm     [shineMaxGranules][21]int32
	xrmaxl [shineMaxGranules]int32
	pe     [shineMaxChannels][shineMaxGranules]float64

	scalefacL [shineMaxGranules][shineMaxChannels][22]int32

	subOff [shineMaxChannels]int
	subX   [shineMaxChannels][shineHanSize]int32
}

func newShineEncoder(rate, bitrateKbps int) (*shineEnc, error) {
	sampleRateIndex := shineFindSamplerate(rate)
	if sampleRateIndex < 0 {
		return nil, fmt.Errorf("%w: shine sample rate %d", avutil.ErrUnsupported, rate)
	}
	version := shineMPEGVersion(sampleRateIndex)
	bitrateIndex := shineFindBitrate(bitrateKbps, version)
	if bitrateIndex < 0 {
		return nil, fmt.Errorf("%w: shine bitrate %d", avutil.ErrUnsupported, bitrateKbps)
	}
	enc := &shineEnc{
		channels:        1,
		sampleRate:      rate,
		bitrate:         bitrateKbps,
		version:         version,
		granules:        shineGranulesForVersion[version],
		mode:            shineModeMono,
		emphasis:        0,
		copyright:       0,
		original:        1,
		bitsPerSlot:     8,
		bitrateIndex:    bitrateIndex,
		sampleRateIndex: sampleRateIndex,
		bs: shineBits{
			buf:  make([]byte, 4096),
			free: 32,
		},
	}
	avg := (float64(enc.granules) * float64(shineGranule) / float64(enc.sampleRate)) *
		(1000 * float64(enc.bitrate) / float64(enc.bitsPerSlot))
	enc.wholeSlots = int(avg)
	enc.fracSlots = avg - float64(enc.wholeSlots)
	enc.slotLag = -enc.fracSlots
	if enc.fracSlots == 0 {
		enc.padding = 0
	}
	if enc.granules == 2 {
		if enc.channels == 1 {
			enc.sideInfoLen = 8 * (4 + 17)
		} else {
			enc.sideInfoLen = 8 * (4 + 32)
		}
	} else if enc.channels == 1 {
		enc.sideInfoLen = 8 * (4 + 9)
	} else {
		enc.sideInfoLen = 8 * (4 + 17)
	}
	return enc, nil
}

func shineFindSamplerate(freq int) int {
	for i, rate := range shineSamplerates {
		if freq == rate {
			return i
		}
	}
	return -1
}

func shineFindBitrate(bitrate, version int) int {
	for i := 0; i < 16; i++ {
		if bitrate == shineBitrates[i][version] {
			return i
		}
	}
	return -1
}

func shineMPEGVersion(sampleRateIndex int) int {
	if sampleRateIndex < 3 {
		return shineMPEGI
	}
	if sampleRateIndex < 6 {
		return shineMPEGII
	}
	return shineMPEG25
}

func (e *shineEnc) encodeFrame(pcm []int16) []byte {
	if e.fracSlots != 0 {
		if e.slotLag <= e.fracSlots-1 {
			e.padding = 1
		} else {
			e.padding = 0
		}
		e.slotLag += float64(e.padding) - e.fracSlots
	}
	e.bitsPerFrame = 8 * (e.wholeSlots + e.padding)
	e.meanBits = (e.bitsPerFrame - e.sideInfoLen) / e.granules

	e.mdctSub(pcm)
	e.iterationLoop()
	e.formatBitstream()

	out := append([]byte(nil), e.bs.buf[:e.bs.pos]...)
	e.bs.pos = 0
	return out
}

func (e *shineEnc) flush() []byte {
	out := append([]byte(nil), e.bs.buf[:e.bs.pos]...)
	e.bs.pos = 0
	return out
}

func (e *shineEnc) mdctSub(pcm []int16) {
	for ch := e.channels - 1; ch >= 0; ch-- {
		pos := 0
		for gr := 0; gr < e.granules; gr++ {
			for k := 0; k < 18; k += 2 {
				e.windowFilter(ch, pcm, &pos, e.sbSample[ch][gr+1][k][:])
				e.windowFilter(ch, pcm, &pos, e.sbSample[ch][gr+1][k+1][:])
				for band := 1; band < shineSBLimit; band += 2 {
					e.sbSample[ch][gr+1][k+1][band] *= -1
				}
			}
			freq := e.mdctFreq[ch][gr][:]
			for band := 0; band < shineSBLimit; band++ {
				var mdctIn [36]int32
				for k := 17; k >= 0; k-- {
					mdctIn[k] = e.sbSample[ch][gr][k][band]
					mdctIn[k+18] = e.sbSample[ch][gr+1][k][band]
				}
				for k := 17; k >= 0; k-- {
					vm := shineMul(mdctIn[35], shineMDCTCos[k*36+35])
					for j := 34; j >= 0; j-- {
						vm += shineMul(mdctIn[j], shineMDCTCos[k*36+j])
					}
					freq[band*18+k] = vm
				}
				if band != 0 {
					prev := band*18 - 18
					cur := band * 18
					for t := 0; t < 8; t++ {
						cs := shineMDCTCS[t]
						ca := shineMDCTCA[t]
						are := freq[cur+t]
						aim := freq[prev+17-t]
						freq[cur+t] = int32((int64(are)*int64(cs) - int64(aim)*int64(ca)) >> 31)
						freq[prev+17-t] = int32((int64(are)*int64(ca) + int64(aim)*int64(cs)) >> 31)
					}
				}
			}
		}
		e.sbSample[ch][0] = e.sbSample[ch][e.granules]
	}
}

func (e *shineEnc) windowFilter(ch int, pcm []int16, pos *int, dst []int32) {
	x := e.subX[ch][:]
	off := e.subOff[ch]
	for i := 31; i >= 0; i-- {
		x[i+off] = int32(pcm[*pos]) << 16
		*pos++
	}
	var y [64]int32
	for i := 63; i >= 0; i-- {
		s := shineMul(x[(off+i)&(shineHanSize-1)], shineEnwindow[i])
		for k := 1; k < 8; k++ {
			s += shineMul(x[(off+i+(k<<6))&(shineHanSize-1)], shineEnwindow[i+(k<<6)])
		}
		y[i] = s
	}
	e.subOff[ch] = (off + 480) & (shineHanSize - 1)
	for i := shineSBLimit - 1; i >= 0; i-- {
		s := shineMul(shineSubbandFL[i*64+63], y[63])
		for j := 62; j >= 0; j-- {
			s += shineMul(shineSubbandFL[i*64+j], y[j])
		}
		dst[i] = s
	}
}

func (e *shineEnc) iterationLoop() {
	for ch := e.channels - 1; ch >= 0; ch-- {
		for gr := 0; gr < e.granules; gr++ {
			ix := e.l3Enc[ch][gr][:]
			e.xr = e.mdctFreq[ch][gr][:]
			e.xrmax = 0
			for i := shineGranule - 1; i >= 0; i-- {
				xr := e.xr[i]
				e.xrsq[i] = shineMulsr(xr, xr)
				ax := shineAbs32(xr)
				e.xrabs[i] = ax
				if ax > e.xrmax {
					e.xrmax = ax
				}
			}
			gi := &e.gr[gr][ch]
			gi.sfbLMax = 21
			e.calcSCFSI(ch, gr)
			maxBits := e.maxReservoir(e.pe[ch][gr])
			for i := 3; i >= 0; i-- {
				gi.slen[i] = 0
			}
			gi.part23Length = 0
			gi.bigValues = 0
			gi.count1 = 0
			gi.scalefacCompress = 0
			gi.tableSelect[0] = 0
			gi.tableSelect[1] = 0
			gi.tableSelect[2] = 0
			gi.region0Count = 0
			gi.region1Count = 0
			gi.part2Length = 0
			gi.preflag = 0
			gi.scalefacScale = 0
			gi.count1TableSelect = 0
			if e.xrmax != 0 {
				gi.part23Length = uint32(e.outerLoop(maxBits, ix, gr, ch))
			}
			e.resvAdjust(gi)
			gi.globalGain = uint32(gi.quantizerStepSize + 210)
		}
	}
	e.resvFrameEnd()
}

func (e *shineEnc) calcSCFSI(ch, gr int) {
	if e.version != shineMPEGI {
		return
	}
	bands := shineScaleBands[e.sampleRateIndex][:]
	e.xrmaxl[gr] = e.xrmax
	var temp int32
	for i := shineGranule - 1; i >= 0; i-- {
		temp += e.xrsq[i] >> 10
	}
	if temp != 0 {
		e.enTot[gr] = int32(math.Log(float64(temp)*4.768371584e-7) / shineLN2)
	} else {
		e.enTot[gr] = 0
	}
	for sfb := 20; sfb >= 0; sfb-- {
		temp = 0
		for i := bands[sfb]; i < bands[sfb+1]; i++ {
			temp += e.xrsq[i] >> 10
		}
		if temp != 0 {
			e.en[gr][sfb] = int32(math.Log(float64(temp)*4.768371584e-7) / shineLN2)
		} else {
			e.en[gr][sfb] = 0
		}
		e.xm[gr][sfb] = 0
	}
	if gr != 1 {
		return
	}
	condition := 0
	for gr2 := 1; gr2 >= 0; gr2-- {
		if e.xrmaxl[gr2] != 0 {
			condition++
		}
		condition++
	}
	if shineAbs(int(e.enTot[0]-e.enTot[1])) < 10 {
		condition++
	}
	tp := 0
	for sfb := 20; sfb >= 0; sfb-- {
		tp += shineAbs(int(e.en[0][sfb] - e.en[1][sfb]))
	}
	if tp < 100 {
		condition++
	}
	if condition == 6 {
		for band := 0; band < 4; band++ {
			sum0 := 0
			sum1 := 0
			start := shineSCFSIBand[band]
			end := shineSCFSIBand[band+1]
			for sfb := start; sfb < end; sfb++ {
				sum0 += shineAbs(int(e.en[0][sfb] - e.en[1][sfb]))
				sum1 += shineAbs(int(e.xm[0][sfb] - e.xm[1][sfb]))
			}
			if sum0 < 10 && sum1 < 10 {
				e.scfsi[ch][band] = 1
			} else {
				e.scfsi[ch][band] = 0
			}
		}
		return
	}
	for band := 0; band < 4; band++ {
		e.scfsi[ch][band] = 0
	}
}

func (e *shineEnc) part2Length(gr, ch int) int {
	gi := &e.gr[gr][ch]
	slen1 := shineSlen1[gi.scalefacCompress]
	slen2 := shineSlen2[gi.scalefacCompress]
	bits := 0
	if gr == 0 || e.scfsi[ch][0] == 0 {
		bits += 6 * slen1
	}
	if gr == 0 || e.scfsi[ch][1] == 0 {
		bits += 5 * slen1
	}
	if gr == 0 || e.scfsi[ch][2] == 0 {
		bits += 5 * slen2
	}
	if gr == 0 || e.scfsi[ch][3] == 0 {
		bits += 5 * slen2
	}
	return bits
}

func (e *shineEnc) outerLoop(maxBits int, ix []int, gr, ch int) int {
	gi := &e.gr[gr][ch]
	gi.quantizerStepSize = e.binSearchStep(maxBits, ix, gi)
	gi.part2Length = uint32(e.part2Length(gr, ch))
	huffBits := maxBits - int(gi.part2Length)
	bits := e.innerLoop(ix, huffBits, gi)
	gi.part23Length = gi.part2Length + uint32(bits)
	return int(gi.part23Length)
}

func (e *shineEnc) innerLoop(ix []int, maxBits int, gi *shineGR) int {
	if maxBits < 0 {
		gi.quantizerStepSize--
	}
	var bits int
	for {
		for {
			gi.quantizerStepSize++
			if e.quantize(ix, gi.quantizerStepSize) <= 8192 {
				break
			}
		}
		shineRunlen(ix, gi)
		bits = shineCount1Bits(ix, gi)
		e.subdivide(gi)
		shineBigvSelect(ix, gi)
		bits += shineBigvBits(ix, gi)
		if bits <= maxBits {
			return bits
		}
	}
}

func (e *shineEnc) quantize(ix []int, stepsize int) int {
	scalei := shineSteptabi[stepsize+127]
	if shineMulr(e.xrmax, scalei) > 165140 {
		return 16384
	}
	max := 0
	for i := 0; i < shineGranule; i++ {
		ln := int(shineMulr(shineAbs32(e.xr[i]), scalei))
		var q int
		if ln < 10000 {
			q = shineInt2idx[ln]
		} else {
			scale := shineSteptab[stepsize+127]
			dbl := float64(e.xrabs[i]) * scale * 4.656612875e-10
			q = int(math.Sqrt(math.Sqrt(dbl) * dbl))
		}
		ix[i] = q
		if max < q {
			max = q
		}
	}
	return max
}

func shineRunlen(ix []int, gi *shineGR) {
	i := shineGranule
	for i > 1 {
		if ix[i-1] == 0 && ix[i-2] == 0 {
			i -= 2
			continue
		}
		break
	}
	gi.count1 = 0
	for i > 3 {
		if ix[i-1] <= 1 && ix[i-2] <= 1 && ix[i-3] <= 1 && ix[i-4] <= 1 {
			gi.count1++
			i -= 4
			continue
		}
		break
	}
	gi.bigValues = uint32(i >> 1)
}

func shineCount1Bits(ix []int, gi *shineGR) int {
	sum0 := 0
	sum1 := 0
	i := int(gi.bigValues) << 1
	for k := uint32(0); k < gi.count1; k++ {
		v := ix[i]
		w := ix[i+1]
		x := ix[i+2]
		y := ix[i+3]
		p := v + (w << 1) + (x << 2) + (y << 3)
		sign := 0
		if v != 0 {
			sign++
		}
		if w != 0 {
			sign++
		}
		if x != 0 {
			sign++
		}
		if y != 0 {
			sign++
		}
		sum0 += sign + int(shineHuff[32].hlen[p])
		sum1 += sign + int(shineHuff[33].hlen[p])
		i += 4
	}
	if sum0 < sum1 {
		gi.count1TableSelect = 0
		return sum0
	}
	gi.count1TableSelect = 1
	return sum1
}

func (e *shineEnc) subdivide(gi *shineGR) {
	if gi.bigValues == 0 {
		gi.region0Count = 0
		gi.region1Count = 0
		return
	}
	bands := shineScaleBands[e.sampleRateIndex]
	bigRegion := int(gi.bigValues) * 2
	scfb := 0
	for bands[scfb] < bigRegion {
		scfb++
	}
	this := shineSubdivide[scfb].region0
	for this != 0 {
		if bands[this+1] <= bigRegion {
			break
		}
		this--
	}
	gi.region0Count = uint32(this)
	gi.address1 = uint32(bands[this+1])
	off := int(gi.region0Count) + 1
	this = shineSubdivide[scfb].region1
	for this != 0 {
		if bands[off+this+1] <= bigRegion {
			break
		}
		this--
	}
	gi.region1Count = uint32(this)
	gi.address2 = uint32(bands[off+this+1])
	gi.address3 = uint32(bigRegion)
}

func shineBigvSelect(ix []int, gi *shineGR) {
	gi.tableSelect[0] = 0
	gi.tableSelect[1] = 0
	gi.tableSelect[2] = 0
	if gi.address1 > 0 {
		gi.tableSelect[0] = uint32(shineChooseTable(ix, 0, int(gi.address1)))
	}
	if gi.address2 > gi.address1 {
		gi.tableSelect[1] = uint32(shineChooseTable(ix, int(gi.address1), int(gi.address2)))
	}
	if gi.bigValues<<1 > gi.address2 {
		gi.tableSelect[2] = uint32(shineChooseTable(ix, int(gi.address2), int(gi.bigValues)<<1))
	}
}

func shineChooseTable(ix []int, begin, end int) int {
	max := 0
	for i := begin; i < end; i++ {
		if max < ix[i] {
			max = ix[i]
		}
	}
	if max == 0 {
		return 0
	}
	choice0 := 0
	if max < 15 {
		for i := 13; i >= 0; i-- {
			if int(shineHuff[i].xlen) > max {
				choice0 = i
				break
			}
		}
		sum0 := shineCountBits(ix, begin, end, choice0)
		switch choice0 {
		case 2:
			if shineCountBits(ix, begin, end, 3) <= sum0 {
				choice0 = 3
			}
		case 5:
			if shineCountBits(ix, begin, end, 6) <= sum0 {
				choice0 = 6
			}
		case 7:
			sum1 := shineCountBits(ix, begin, end, 8)
			if sum1 <= sum0 {
				choice0 = 8
				sum0 = sum1
			}
			if shineCountBits(ix, begin, end, 9) <= sum0 {
				choice0 = 9
			}
		case 10:
			sum1 := shineCountBits(ix, begin, end, 11)
			if sum1 <= sum0 {
				choice0 = 11
				sum0 = sum1
			}
			if shineCountBits(ix, begin, end, 12) <= sum0 {
				choice0 = 12
			}
		case 13:
			if shineCountBits(ix, begin, end, 15) <= sum0 {
				choice0 = 15
			}
		}
		return choice0
	}
	max -= 15
	choice1 := 0
	for i := 15; i < 24; i++ {
		if int(shineHuff[i].linmax) >= max {
			choice0 = i
			break
		}
	}
	for i := 24; i < 32; i++ {
		if int(shineHuff[i].linmax) >= max {
			choice1 = i
			break
		}
	}
	sum0 := shineCountBits(ix, begin, end, choice0)
	sum1 := shineCountBits(ix, begin, end, choice1)
	if sum1 < sum0 {
		return choice1
	}
	return choice0
}

func shineBigvBits(ix []int, gi *shineGR) int {
	bits := 0
	if table := int(gi.tableSelect[0]); table != 0 {
		bits += shineCountBits(ix, 0, int(gi.address1), table)
	}
	if table := int(gi.tableSelect[1]); table != 0 {
		bits += shineCountBits(ix, int(gi.address1), int(gi.address2), table)
	}
	if table := int(gi.tableSelect[2]); table != 0 {
		bits += shineCountBits(ix, int(gi.address2), int(gi.address3), table)
	}
	return bits
}

func shineCountBits(ix []int, start, end, table int) int {
	if table == 0 {
		return 0
	}
	h := &shineHuff[table]
	sum := 0
	ylen := int(h.ylen)
	linbits := int(h.linbits)
	if table > 15 {
		for i := start; i < end; i += 2 {
			x := ix[i]
			y := ix[i+1]
			if x > 14 {
				x = 15
				sum += linbits
			}
			if y > 14 {
				y = 15
				sum += linbits
			}
			sum += int(h.hlen[x*ylen+y])
			if x != 0 {
				sum++
			}
			if y != 0 {
				sum++
			}
		}
		return sum
	}
	for i := start; i < end; i += 2 {
		x := ix[i]
		y := ix[i+1]
		sum += int(h.hlen[x*ylen+y])
		if x != 0 {
			sum++
		}
		if y != 0 {
			sum++
		}
	}
	return sum
}

func (e *shineEnc) binSearchStep(desired int, ix []int, gi *shineGR) int {
	next := -120
	count := 120
	for count > 1 {
		half := count / 2
		var bit int
		if e.quantize(ix, next+half) > 8192 {
			bit = 100000
		} else {
			shineRunlen(ix, gi)
			bit = shineCount1Bits(ix, gi)
			e.subdivide(gi)
			shineBigvSelect(ix, gi)
			bit += shineBigvBits(ix, gi)
		}
		if bit < desired {
			count = half
		} else {
			next += half
			count -= half
		}
	}
	return next
}

func (e *shineEnc) maxReservoir(pe float64) int {
	mean := e.meanBits / e.channels
	maxBits := mean
	if maxBits > 4095 {
		maxBits = 4095
	}
	if e.resvMax == 0 {
		return maxBits
	}
	more := int(pe*3.1 - float64(mean))
	add := 0
	if more > 100 {
		frac := (e.resvSize * 6) / 10
		if frac < more {
			add = frac
		} else {
			add = more
		}
	}
	over := e.resvSize - ((e.resvMax << 3) / 10) - add
	if over > 0 {
		add += over
	}
	maxBits += add
	if maxBits > 4095 {
		maxBits = 4095
	}
	return maxBits
}

func (e *shineEnc) resvAdjust(gi *shineGR) {
	e.resvSize += (e.meanBits / e.channels) - int(gi.part23Length)
}

func (e *shineEnc) resvFrameEnd() {
	if e.channels == 2 && e.meanBits&1 != 0 {
		e.resvSize++
	}
	over := e.resvSize - e.resvMax
	if over < 0 {
		over = 0
	}
	e.resvSize -= over
	stuffing := over
	if rem := e.resvSize % 8; rem != 0 {
		stuffing += rem
		e.resvSize -= rem
	}
	if stuffing == 0 {
		return
	}
	gi := &e.gr[0][0]
	if int(gi.part23Length)+stuffing < 4095 {
		gi.part23Length += uint32(stuffing)
		return
	}
	for gr := 0; gr < e.granules; gr++ {
		for ch := 0; ch < e.channels; ch++ {
			if stuffing == 0 {
				break
			}
			g := &e.gr[gr][ch]
			extra := 4095 - int(g.part23Length)
			bitsThis := stuffing
			if extra < stuffing {
				bitsThis = extra
			}
			g.part23Length += uint32(bitsThis)
			stuffing -= bitsThis
		}
	}
}

func (e *shineEnc) formatBitstream() {
	for ch := 0; ch < e.channels; ch++ {
		for gr := 0; gr < e.granules; gr++ {
			for i := 0; i < shineGranule; i++ {
				if e.mdctFreq[ch][gr][i] < 0 && e.l3Enc[ch][gr][i] > 0 {
					e.l3Enc[ch][gr][i] = -e.l3Enc[ch][gr][i]
				}
			}
		}
	}
	e.encodeSideInfo()
	e.encodeMainData()
}

func (e *shineEnc) encodeSideInfo() {
	b := &e.bs
	b.put(0x7ff, 11)
	b.put(uint32(e.version), 2)
	b.put(uint32(shineLayerIII), 2)
	b.put(1, 1)
	b.put(uint32(e.bitrateIndex), 4)
	b.put(uint32(e.sampleRateIndex%3), 2)
	b.put(uint32(e.padding), 1)
	b.put(0, 1)
	b.put(uint32(e.mode), 2)
	b.put(0, 2)
	b.put(uint32(e.copyright), 1)
	b.put(uint32(e.original), 1)
	b.put(uint32(e.emphasis), 2)
	if e.version == shineMPEGI {
		b.put(0, 9)
		if e.channels == 2 {
			b.put(e.privateBits, 3)
		} else {
			b.put(e.privateBits, 5)
		}
	} else {
		b.put(0, 8)
		if e.channels == 2 {
			b.put(e.privateBits, 2)
		} else {
			b.put(e.privateBits, 1)
		}
	}
	if e.version == shineMPEGI {
		for ch := 0; ch < e.channels; ch++ {
			for band := 0; band < 4; band++ {
				b.put(e.scfsi[ch][band], 1)
			}
		}
	}
	for gr := 0; gr < e.granules; gr++ {
		for ch := 0; ch < e.channels; ch++ {
			gi := &e.gr[gr][ch]
			b.put(gi.part23Length, 12)
			b.put(gi.bigValues, 9)
			b.put(gi.globalGain, 8)
			if e.version == shineMPEGI {
				b.put(gi.scalefacCompress, 4)
			} else {
				b.put(gi.scalefacCompress, 9)
			}
			b.put(0, 1)
			for region := 0; region < 3; region++ {
				b.put(gi.tableSelect[region], 5)
			}
			b.put(gi.region0Count, 4)
			b.put(gi.region1Count, 3)
			if e.version == shineMPEGI {
				b.put(gi.preflag, 1)
			}
			b.put(gi.scalefacScale, 1)
			b.put(gi.count1TableSelect, 1)
		}
	}
}

func (e *shineEnc) encodeMainData() {
	for gr := 0; gr < e.granules; gr++ {
		for ch := 0; ch < e.channels; ch++ {
			gi := &e.gr[gr][ch]
			slen1 := shineSlen1[gi.scalefacCompress]
			slen2 := shineSlen2[gi.scalefacCompress]
			if gr == 0 || e.scfsi[ch][0] == 0 {
				for sfb := 0; sfb < 6; sfb++ {
					e.bs.put(uint32(e.scalefacL[gr][ch][sfb]), slen1)
				}
			}
			if gr == 0 || e.scfsi[ch][1] == 0 {
				for sfb := 6; sfb < 11; sfb++ {
					e.bs.put(uint32(e.scalefacL[gr][ch][sfb]), slen1)
				}
			}
			if gr == 0 || e.scfsi[ch][2] == 0 {
				for sfb := 11; sfb < 16; sfb++ {
					e.bs.put(uint32(e.scalefacL[gr][ch][sfb]), slen2)
				}
			}
			if gr == 0 || e.scfsi[ch][3] == 0 {
				for sfb := 16; sfb < 21; sfb++ {
					e.bs.put(uint32(e.scalefacL[gr][ch][sfb]), slen2)
				}
			}
			e.huffmanBits(e.l3Enc[ch][gr][:], gi)
		}
	}
}

func (e *shineEnc) huffmanBits(ix []int, gi *shineGR) {
	bands := shineScaleBands[e.sampleRateIndex]
	startBits := e.bs.count()
	bigValues := int(gi.bigValues) << 1
	region1 := bands[gi.region0Count+1]
	region2 := bands[int(gi.region0Count)+int(gi.region1Count)+2]
	for i := 0; i < bigValues; i += 2 {
		idx := 0
		if i >= region1 {
			idx++
		}
		if i >= region2 {
			idx++
		}
		table := int(gi.tableSelect[idx])
		if table != 0 {
			e.bs.huffman(table, ix[i], ix[i+1])
		}
	}
	h := &shineHuff[gi.count1TableSelect+32]
	count1End := bigValues + int(gi.count1)<<2
	for i := bigValues; i < count1End; i += 4 {
		e.bs.huffmanCount1(h, ix[i], ix[i+1], ix[i+2], ix[i+3])
	}
	bits := e.bs.count() - startBits
	stuff := int(gi.part23Length) - int(gi.part2Length) - bits
	if stuff == 0 {
		return
	}
	words := stuff / 32
	remain := stuff % 32
	for words > 0 {
		e.bs.put(0xffffffff, 32)
		words--
	}
	if remain != 0 {
		e.bs.put(uint32(1)<<uint(remain)-1, remain)
	}
}

func (b *shineBits) put(val uint32, n int) {
	if n == 0 {
		return
	}
	if b.free > n {
		b.free -= n
		b.cache |= val << uint(b.free)
		return
	}
	n -= b.free
	b.cache |= val >> uint(n)
	if b.pos+4 > len(b.buf) {
		next := make([]byte, len(b.buf)+len(b.buf)/2+4)
		copy(next, b.buf)
		b.buf = next
	}
	binary.BigEndian.PutUint32(b.buf[b.pos:], b.cache)
	b.pos += 4
	b.free = 32 - n
	if n != 0 {
		b.cache = val << uint(b.free)
	} else {
		b.cache = 0
	}
}

func (b *shineBits) count() int {
	return b.pos*8 + 32 - b.free
}

func (b *shineBits) huffman(table, x, y int) {
	signX := shineAbsSign(&x)
	signY := shineAbsSign(&y)
	h := &shineHuff[table]
	ylen := int(h.ylen)
	if table > 15 {
		linbits := h.linbits
		var linX, linY uint32
		if x > 14 {
			linX = uint32(x - 15)
			x = 15
		}
		if y > 14 {
			linY = uint32(y - 15)
			y = 15
		}
		idx := x*ylen + y
		code := uint32(h.table[idx])
		cbits := int(h.hlen[idx])
		ext := uint32(0)
		xbits := 0
		if x > 14 {
			ext |= linX
			xbits += int(linbits)
		}
		if x != 0 {
			ext = (ext << 1) | signX
			xbits++
		}
		if y > 14 {
			ext = (ext << linbits) | linY
			xbits += int(linbits)
		}
		if y != 0 {
			ext = (ext << 1) | signY
			xbits++
		}
		b.put(code, cbits)
		b.put(ext, xbits)
		return
	}
	idx := x*ylen + y
	code := uint32(h.table[idx])
	cbits := int(h.hlen[idx])
	if x != 0 {
		code = (code << 1) | signX
		cbits++
	}
	if y != 0 {
		code = (code << 1) | signY
		cbits++
	}
	b.put(code, cbits)
}

func (b *shineBits) huffmanCount1(h *shineHuffTab, v, w, x, y int) {
	signV := shineAbsSign(&v)
	signW := shineAbsSign(&w)
	signX := shineAbsSign(&x)
	signY := shineAbsSign(&y)
	p := v + (w << 1) + (x << 2) + (y << 3)
	b.put(uint32(h.table[p]), int(h.hlen[p]))
	code := uint32(0)
	cbits := 0
	if v != 0 {
		code = signV
		cbits = 1
	}
	if w != 0 {
		code = (code << 1) | signW
		cbits++
	}
	if x != 0 {
		code = (code << 1) | signX
		cbits++
	}
	if y != 0 {
		code = (code << 1) | signY
		cbits++
	}
	b.put(code, cbits)
}

func shineAbsSign(x *int) uint32 {
	if *x > 0 {
		return 0
	}
	*x = -*x
	return 1
}

func shineMul(a, b int32) int32 {
	return int32((int64(a) * int64(b)) >> 32)
}

func shineMulr(a, b int32) int32 {
	return int32(((int64(a)*int64(b) + 0x80000000) >> 32))
}

func shineMulsr(a, b int32) int32 {
	return int32(((int64(a)*int64(b) + 0x40000000) >> 31))
}

func shineAbs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func shineAbs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func shineCompleteFrames(stream []byte, maxFrames int) ([]byte, error) {
	var out []byte
	for len(stream) >= 4 && maxFrames > 0 {
		header := binary.BigEndian.Uint32(stream[:4])
		size, err := shineFrameBytes(header)
		if err != nil {
			return nil, err
		}
		if size > len(stream) {
			break
		}
		out = append(out, stream[:size]...)
		stream = stream[size:]
		maxFrames--
	}
	return out, nil
}

func shineFrameBytes(header uint32) (int, error) {
	if header&0xffe00000 != 0xffe00000 ||
		header&(3<<19) == 1<<19 ||
		header&(3<<17) == 0 ||
		header&(0xf<<12) == 0xf<<12 ||
		header&(3<<10) == 3<<10 {
		return 0, fmt.Errorf("%w: shine frame header", avutil.ErrInvalidData)
	}
	var lsf, mpeg25 int
	if header&(1<<20) != 0 {
		if header&(1<<19) == 0 {
			lsf = 1
		}
	} else {
		lsf = 1
		mpeg25 = 1
	}
	layer := 4 - int((header>>17)&3)
	sampleIndex := int((header >> 10) & 3)
	if sampleIndex > 2 {
		sampleIndex = 0
	}
	sampleRate := shineMPAFreq[sampleIndex] >> (lsf + mpeg25)
	bitrateIndex := int((header >> 12) & 0xf)
	padding := int((header >> 9) & 1)
	if bitrateIndex == 0 {
		return 0, fmt.Errorf("%w: shine free format", avutil.ErrInvalidData)
	}
	bitrate := shineMPABitrate[lsf][layer-1][bitrateIndex]
	var frameSize int
	switch layer {
	case 1:
		frameSize = (bitrate * 12000) / sampleRate
		frameSize = (frameSize + padding) * 4
	case 2:
		frameSize = (bitrate*144000)/sampleRate + padding
	default:
		frameSize = (bitrate*144000)/(sampleRate<<lsf) + padding
	}
	return frameSize, nil
}
