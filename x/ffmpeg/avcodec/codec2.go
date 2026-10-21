package avcodec

import (
	"math"
	"sync"
)

// Codec 2 mode 1300 (FreeDV 1600): 8 kHz mono, 320 samples (40 ms) to 52 bits
// packed in 7 bytes. Gray coding is on, matching libcodec2's ffmpeg wrapper.
// One codec state covers the whole buffer; a short tail is zero-padded to a
// frame because the encoder has no short-frame mode.

const (
	c2PI      = 3.141592654
	c2TwoPI   = 6.283185307
	c2Fs      = 8000
	c2NSamp   = 80
	c2MPitch  = 320
	c2PMin    = 20
	c2PMax    = 160
	c2NW      = 279
	c2TW      = 40
	c2FFT     = 512
	c2LPCOrd  = 10
	c2MaxAmp  = 160
	c2FrameN  = 320
	c2FrameB  = 7
	c2WOBits  = 7
	c2EBits   = 5
	c2EMin    = -10.0
	c2EMax    = 40.0
	c2Beta    = 0.2
	c2Gamma   = 0.5
	c2VThresh = 6.0

	c2NLPCoeff = 0.95
	c2Dec      = 5
	c2CNLP     = 0.3
	c2NLPTap   = 48
	c2PMaxS    = 0.0200
	c2Ln10     = 2.302585092994046
)

// C stores Wo_min/Wo_max as float after a double division.
var (
	c2WoMin = sync.OnceValue(func() float64 { return f32(c2TwoPI / float64(c2PMax)) })
	c2WoMax = sync.OnceValue(func() float64 { return f32(c2TwoPI / float64(c2PMin)) })
)

type cpx struct {
	r, i float64
}

type c2Model struct {
	wo     float64
	l      int
	a      [c2MaxAmp + 1]float64
	phi    [c2MaxAmp + 1]float64
	voiced int
}

type c2NLP struct {
	m      int
	w      []float64
	sq     []float64
	memX   float64
	memY   float64
	memFIR [c2NLPTap]float64
}

type c2Codec struct {
	w        []float64
	bigW     []float64
	sn       []float64
	pn       []float64
	snSynth  []float64
	nlp      *c2NLP
	prevF0   float64
	prev     c2Model
	prevLSP  [c2LPCOrd]float64
	prevE    float64
	exPhase  float64
	bgEst    float64
	randNext uint64
}

// Codec2Encode1300 packs 8 kHz mono s16 into codec2raw mode 1300.
// A partial tail is zero-padded to 320 samples, matching ffmpeg.
func Codec2Encode1300(pcm []int16) ([]byte, error) {
	if len(pcm) == 0 {
		return nil, nil
	}
	if len(pcm)%c2FrameN != 0 {
		padded := make([]int16, (len(pcm)/c2FrameN+1)*c2FrameN)
		copy(padded, pcm)
		pcm = padded
	}
	c := newC2()
	out := make([]byte, 0, len(pcm)/c2FrameN*c2FrameB)
	var buf [c2FrameB]byte
	for off := 0; off < len(pcm); off += c2FrameN {
		c.encode(buf[:], pcm[off:off+c2FrameN])
		out = append(out, buf[:]...)
	}
	return out, nil
}

// Codec2Decode1300 expands each 7-byte mode-1300 frame to 320 samples.
// A trailing partial frame is dropped.
func Codec2Decode1300(data []byte) ([]int16, error) {
	n := len(data) / c2FrameB
	if n == 0 {
		return nil, nil
	}
	c := newC2()
	// ffmpeg reads the first packet while probing. codec2_rand is process-global,
	// so the PCM on stdout is a second decode with the generator already advanced.
	probe := newC2()
	var scratch [c2FrameN]int16
	probe.decode(scratch[:], data[:c2FrameB])
	c.randNext = probe.randNext
	out := make([]int16, n*c2FrameN)
	for i := 0; i < n; i++ {
		c.decode(out[i*c2FrameN:(i+1)*c2FrameN], data[i*c2FrameB:(i+1)*c2FrameB])
	}
	return out, nil
}

func newC2() *c2Codec {
	c := &c2Codec{
		sn:       make([]float64, c2MPitch),
		snSynth:  make([]float64, 2*c2NSamp),
		prevF0:   1 / c2PMaxS,
		prevE:    1,
		randNext: 1,
	}
	for i := range c.sn {
		c.sn[i] = 1
	}
	c.makeAnalysisWindow()
	c.makeSynthWindow()
	c.nlp = newNLP()
	c.prev.wo = c2WoMin()
	c.prev.l = int(c2PI / c.prev.wo)
	for i := 0; i < c2LPCOrd; i++ {
		c.prevLSP[i] = f32(float64(i) * c2PI / float64(c2LPCOrd+1))
	}
	return c
}

func newNLP() *c2NLP {
	n := &c2NLP{
		m:  c2MPitch,
		w:  make([]float64, c2MPitch/c2Dec),
		sq: make([]float64, c2MPitch),
	}
	den := float64(c2MPitch/c2Dec - 1)
	for i := range n.w {
		n.w[i] = 0.5 - 0.5*math.Cos(2*c2PI*float64(i)/den)
	}
	return n
}

func (c *c2Codec) makeAnalysisWindow() {
	nw := c2NW
	c.w = make([]float64, c2MPitch)
	sum := f32(0)
	for i, j := c2MPitch/2-nw/2, 0; i < c2MPitch/2+nw/2; i, j = i+1, j+1 {
		// 0.5 is double and cosf returns float, so the mix is rounded on store.
		ang := f32(c2TwoPI * float64(j) / float64(nw-1))
		c.w[i] = f32(0.5 - 0.5*float64(cosf(ang)))
		sum = f32(sum + f32(c.w[i]*c.w[i]))
	}
	sum = f32(1.0 / float64(sqrtf(f32(sum*float64(c2FFT)))))
	for i := range c.w {
		c.w[i] = f32(c.w[i] * sum)
	}
	shift := make([]cpx, c2FFT)
	for i := 0; i < nw/2; i++ {
		shift[i].r = c.w[i+c2MPitch/2]
	}
	j := c2MPitch/2 - nw/2
	for i := c2FFT - nw/2; i < c2FFT; i++ {
		shift[i].r = c.w[j]
		j++
	}
	fftInPlace(shift, false)
	c.bigW = make([]float64, c2FFT)
	for i := 0; i < c2FFT/2; i++ {
		c.bigW[i] = shift[i+c2FFT/2].r
		c.bigW[i+c2FFT/2] = shift[i].r
	}
}

func (c *c2Codec) makeSynthWindow() {
	c.pn = make([]float64, 2*c2NSamp)
	step := 1 / float64(2*c2TW)
	win := 0.0
	for i := c2NSamp/2 - c2TW; i < c2NSamp/2+c2TW; i++ {
		c.pn[i] = win
		win += step
	}
	for i := c2NSamp/2 + c2TW; i < 3*c2NSamp/2-c2TW; i++ {
		c.pn[i] = 1
	}
	win = 1
	for i := 3*c2NSamp/2 - c2TW; i < 3*c2NSamp/2+c2TW; i++ {
		c.pn[i] = win
		win -= step
	}
}

func (c *c2Codec) encode(bits []byte, speech []int16) {
	for i := range bits {
		bits[i] = 0
	}
	var nbit uint
	var model c2Model
	for f := 0; f < 4; f++ {
		c.analyse(&model, speech[f*c2NSamp:(f+1)*c2NSamp])
		packBits(bits, &nbit, model.voiced, 1, true)
	}
	packBits(bits, &nbit, encodeWo(model.wo), c2WOBits, true)
	lsp := make([]float64, c2LPCOrd)
	ak := make([]float64, c2LPCOrd+1)
	e := speechToLSPs(lsp, ak, c.sn, c.w)
	packBits(bits, &nbit, encodeEnergy(e), c2EBits, true)
	var idx [c2LPCOrd]int
	encodeLSPScalar(idx[:], lsp)
	for i := 0; i < c2LPCOrd; i++ {
		packBits(bits, &nbit, idx[i], uint(c2LspCB[i].bits), true)
	}
}

func (c *c2Codec) decode(speech []int16, bits []byte) {
	var nbit uint
	var model [4]c2Model
	var lsp [4][c2LPCOrd]float64
	var e [4]float64
	var idx [c2LPCOrd]int
	for i := 0; i < 4; i++ {
		model[i].voiced = unpackBits(bits, &nbit, 1, true)
	}
	model[3].wo = decodeWo(unpackBits(bits, &nbit, c2WOBits, true))
	model[3].l = int(c2PI / model[3].wo)
	e[3] = decodeEnergy(unpackBits(bits, &nbit, c2EBits, true))
	for i := 0; i < c2LPCOrd; i++ {
		idx[i] = unpackBits(bits, &nbit, uint(c2LspCB[i].bits), true)
	}
	decodeLSPScalar(lsp[3][:], idx[:])
	checkLSPOrder(lsp[3][:])
	bwExpand(lsp[3][:], 50, 100)
	weight := 0.25
	for i := 0; i < 3; i++ {
		for k := 0; k < c2LPCOrd; k++ {
			// 1.0 is double, weight and the LSP samples are float.
			w := float32(weight)
			prev := float32(c.prevLSP[k])
			next := float32(lsp[3][k])
			term := (1.0-float64(w))*float64(prev) + float64(w*next)
			lsp[i][k] = f32(term)
		}
		interpWo(&model[i], &c.prev, &model[3], weight)
		e[i] = interpEnergy(c.prevE, e[3], weight)
		weight += 0.25
	}
	aw := make([]cpx, c2FFT/2+1)
	for i := 0; i < 4; i++ {
		ak := make([]float64, c2LPCOrd+1)
		lspToLPC(lsp[i][:], ak)
		aksToM2(ak, &model[i], e[i], aw)
		if model[i].wo < c2PI*150/4000 {
			model[i].a[1] *= 0.032
		}
		c.synthOne(&model[i], speech[i*c2NSamp:(i+1)*c2NSamp], aw)
	}
	c.prev = model[3]
	c.prevE = e[3]
	c.prevLSP = lsp[3]
}

func (c *c2Codec) analyse(model *c2Model, speech []int16) {
	copy(c.sn[:c2MPitch-c2NSamp], c.sn[c2NSamp:c2MPitch])
	for i := 0; i < c2NSamp; i++ {
		c.sn[i+c2MPitch-c2NSamp] = float64(speech[i])
	}
	sw := c.dftSpeech()
	pitch := c.nlp.pitch(c.sn, c2NSamp, &c.prevF0)
	model.wo = c2TwoPI / pitch
	model.l = int(c2PI / model.wo)
	twoStage(model, sw)
	estimateAmps(model, sw)
	estVoicing(model, sw, c.bigW)
}

func (c *c2Codec) dftSpeech() []cpx {
	nw := c2NW
	sw := make([]cpx, c2FFT)
	for i := 0; i < nw/2; i++ {
		sw[i].r = c.sn[i+c2MPitch/2] * c.w[i+c2MPitch/2]
	}
	for i := 0; i < nw/2; i++ {
		sw[c2FFT-nw/2+i].r = c.sn[i+c2MPitch/2-nw/2] * c.w[i+c2MPitch/2-nw/2]
	}
	fftInPlace(sw, false)
	return sw
}

func (n *c2NLP) pitch(sn []float64, hop int, prevF0 *float64) float64 {
	m := n.m
	for i := m - hop; i < m; i++ {
		n.sq[i] = sn[i] * sn[i]
	}
	for i := m - hop; i < m; i++ {
		notch := n.sq[i] - n.memX
		notch += c2NLPCoeff * n.memY
		n.memX = n.sq[i]
		n.memY = notch
		n.sq[i] = notch + 1
	}
	for i := m - hop; i < m; i++ {
		copy(n.memFIR[:c2NLPTap-1], n.memFIR[1:])
		n.memFIR[c2NLPTap-1] = n.sq[i]
		acc := 0.0
		for j := 0; j < c2NLPTap; j++ {
			acc += n.memFIR[j] * nlpFir[j]
		}
		n.sq[i] = acc
	}
	buf := make([]cpx, c2FFT)
	for i := 0; i < m/c2Dec; i++ {
		buf[i].r = n.sq[i*c2Dec] * n.w[i]
	}
	fftInPlace(buf, false)
	fw := make([]float64, c2FFT)
	for i := range fw {
		fw[i] = buf[i].r*buf[i].r + buf[i].i*buf[i].i
	}
	pmin := c2PMin
	pmax := c2PMax
	gmaxBin := c2FFT * c2Dec / pmax
	gmax := 0.0
	for i := gmaxBin; i <= c2FFT*c2Dec/pmin; i++ {
		if fw[i] > gmax {
			gmax = fw[i]
			gmaxBin = i
		}
	}
	best := postSubMultiples(fw, pmax, gmax, gmaxBin, *prevF0)
	copy(n.sq[:m-hop], n.sq[hop:m])
	*prevF0 = best
	return float64(c2Fs) / best
}

func postSubMultiples(fw []float64, pmax int, gmax float64, gmaxBin int, prevF0 float64) float64 {
	mult := 2
	minBin := c2FFT * c2Dec / pmax
	cmax := gmaxBin
	prevBin := int(prevF0 * float64(c2FFT*c2Dec) / float64(c2Fs))
	for gmaxBin/mult >= minBin {
		b := gmaxBin / mult
		bmin := int(0.8 * float64(b))
		bmax := int(1.2 * float64(b))
		if bmin < minBin {
			bmin = minBin
		}
		thresh := c2CNLP * gmax
		if prevBin > bmin && prevBin < bmax {
			thresh = c2CNLP * 0.5 * gmax
		}
		lmax := 0.0
		lmaxBin := bmin
		for bb := bmin; bb <= bmax; bb++ {
			if fw[bb] > lmax {
				lmax = fw[bb]
				lmaxBin = bb
			}
		}
		if lmax > thresh && lmax > fw[lmaxBin-1] && lmax > fw[lmaxBin+1] {
			cmax = lmaxBin
		}
		mult++
	}
	return float64(cmax) * float64(c2Fs) / float64(c2FFT*c2Dec)
}

func twoStage(model *c2Model, sw []cpx) {
	hsRefine(model, sw, c2TwoPI/model.wo-5, c2TwoPI/model.wo+5, 1)
	hsRefine(model, sw, c2TwoPI/model.wo-1, c2TwoPI/model.wo+1, 0.25)
	if model.wo < c2WoMin() {
		model.wo = c2WoMin()
	}
	if model.wo > c2WoMax() {
		model.wo = c2WoMax()
	}
	model.l = int(math.Floor(c2PI / model.wo))
	if model.wo*float64(model.l) >= 0.95*c2PI {
		model.l--
	}
}

func hsRefine(model *c2Model, sw []cpx, pmin, pmax, pstep float64) {
	model.l = int(c2PI / model.wo)
	wom := model.wo
	em := 0.0
	r := c2TwoPI / float64(c2FFT)
	oneOnR := 1 / r
	for p := pmin; p <= pmax; p += pstep {
		e := 0.0
		wo := c2TwoPI / p
		bFloat := wo * oneOnR
		cur := bFloat
		for m := 1; m <= model.l; m++ {
			b := int(cur + 0.5)
			e += sw[b].r*sw[b].r + sw[b].i*sw[b].i
			cur += bFloat
		}
		if e > em {
			em = e
			wom = wo
		}
	}
	model.wo = wom
}

func estimateAmps(model *c2Model, sw []cpx) {
	r := c2TwoPI / float64(c2FFT)
	oneOnR := 1 / r
	for m := 1; m <= model.l; m++ {
		den := 0.0
		am := int((float64(m)-0.5)*model.wo*oneOnR + 0.5)
		bm := int((float64(m)+0.5)*model.wo*oneOnR + 0.5)
		for i := am; i < bm; i++ {
			den += sw[i].r*sw[i].r + sw[i].i*sw[i].i
		}
		model.a[m] = math.Sqrt(den)
	}
}

func estVoicing(model *c2Model, sw []cpx, bigW []float64) {
	half := float64(c2Fs / 2)
	l1000 := int(float64(model.l) * 1000 / half)
	sig := 1e-4
	for l := 1; l <= l1000; l++ {
		sig += model.a[l] * model.a[l]
	}
	wo := model.wo
	err := 1e-4
	for l := 1; l <= l1000; l++ {
		var amR, amI, den float64
		al := int(math.Ceil((float64(l) - 0.5) * wo * float64(c2FFT) / c2TwoPI))
		bl := int(math.Ceil((float64(l) + 0.5) * wo * float64(c2FFT) / c2TwoPI))
		offset := int(float64(c2FFT/2) - float64(l)*wo*float64(c2FFT)/c2TwoPI + 0.5)
		for m := al; m < bl; m++ {
			amR += sw[m].r * bigW[offset+m]
			amI += sw[m].i * bigW[offset+m]
			den += bigW[offset+m] * bigW[offset+m]
		}
		amR /= den
		amI /= den
		for m := al; m < bl; m++ {
			er := sw[m].r - amR*bigW[offset+m]
			ei := sw[m].i - amI*bigW[offset+m]
			err += er*er + ei*ei
		}
	}
	if 10*math.Log10(sig/err) > c2VThresh {
		model.voiced = 1
	} else {
		model.voiced = 0
	}
	l2000 := int(float64(model.l) * 2000 / half)
	l4000 := int(float64(model.l) * 4000 / half)
	elow, ehigh := 1e-4, 1e-4
	for l := 1; l <= l2000; l++ {
		elow += model.a[l] * model.a[l]
	}
	for l := l2000; l <= l4000; l++ {
		ehigh += model.a[l] * model.a[l]
	}
	eratio := 10 * math.Log10(elow/ehigh)
	if model.voiced == 0 && eratio > 10 {
		model.voiced = 1
	}
	if model.voiced == 1 {
		if eratio < -10 {
			model.voiced = 0
		}
		if eratio < -4 && model.wo <= 60*c2TwoPI/float64(c2Fs) {
			model.voiced = 0
		}
	}
}

func speechToLSPs(lsp, ak, sn, w []float64) float64 {
	wn := make([]float64, c2MPitch)
	e := f32(0)
	for i := 0; i < c2MPitch; i++ {
		wn[i] = f32(sn[i] * w[i])
		e = f32(e + f32(wn[i]*wn[i]))
	}
	if e == 0 {
		for i := 0; i < c2LPCOrd; i++ {
			lsp[i] = c2PI / float64(c2LPCOrd) * float64(i)
		}
		return 0
	}
	r := make([]float64, c2LPCOrd+1)
	autocorrelate(wn, r)
	levinson(r, ak)
	energy := f32(0)
	for i := 0; i <= c2LPCOrd; i++ {
		energy = f32(energy + f32(ak[i]*r[i]))
	}
	for i := 0; i <= c2LPCOrd; i++ {
		ak[i] = f32(ak[i] * f32(math.Pow(0.994, float64(i))))
	}
	if lpcToLSP(ak, lsp) != c2LPCOrd {
		for i := 0; i < c2LPCOrd; i++ {
			lsp[i] = c2PI / float64(c2LPCOrd) * float64(i)
		}
	}
	return energy
}

// f32 rounds to IEEE binary32. Codec2 stores LPC state in float, and the
// Levinson clamp (|k|>1 -> 0) depends on that rounding.
func f32(x float64) float64 { return float64(float32(x)) }

func cosf(x float64) float32  { return float32(math.Cos(float64(float32(x)))) }
func sqrtf(x float64) float32 { return float32(math.Sqrt(float64(float32(x)))) }

func powf(a, b float32) float32 {
	return float32(math.Pow(float64(a), float64(b)))
}

func autocorrelate(sn, rn []float64) {
	for j := 0; j <= c2LPCOrd; j++ {
		acc := f32(0)
		for i := 0; i < len(sn)-j; i++ {
			acc = f32(acc + f32(sn[i]*sn[i+j]))
		}
		rn[j] = acc
	}
}

func levinson(r, lpcs []float64) {
	order := c2LPCOrd
	a := make([][]float64, order+1)
	for i := range a {
		a[i] = make([]float64, order+1)
	}
	e := f32(r[0])
	for i := 1; i <= order; i++ {
		sum := f32(0)
		for j := 1; j <= i-1; j++ {
			sum = f32(sum + f32(a[i-1][j]*r[i-j]))
		}
		// -1.0 is double in C, so the division is done in float64 then stored as float.
		k := f32(-1.0 * float64(f32(r[i]+sum)) / e)
		if math.Abs(k) > 1 {
			k = 0
		}
		a[i][i] = k
		for j := 1; j <= i-1; j++ {
			a[i][j] = f32(a[i-1][j] + f32(k*a[i-1][i-j]))
		}
		e = f32(e * f32(1-f32(k*k)))
	}
	lpcs[0] = 1
	for i := 1; i <= order; i++ {
		lpcs[i] = a[order][i]
	}
}

func chebEva(coef []float64, x float64) float64 {
	m := c2LPCOrd / 2
	t := make([]float64, m+1)
	t[0] = 1
	t[1] = x
	for i := 2; i <= m; i++ {
		t[i] = 2*x*t[i-1] - t[i-2]
	}
	sum := 0.0
	for i := 0; i <= m; i++ {
		sum += coef[m-i] * t[i]
	}
	return sum
}

func lpcToLSP(a, freq []float64) int {
	order := c2LPCOrd
	m := order / 2
	p := make([]float64, order+1)
	q := make([]float64, order+1)
	p[0], q[0] = 1, 1
	for i := 1; i <= m; i++ {
		p[i] = a[i] + a[order+1-i] - p[i-1]
		q[i] = a[i] - a[order+1-i] + q[i-1]
	}
	for i := 0; i < m; i++ {
		p[i] *= 2
		q[i] *= 2
	}
	xr, xl := 0.0, 1.0
	roots := 0
	const delta = 0.01
	const nb = 5
	for j := 0; j < order; j++ {
		pt := p
		if j%2 != 0 {
			pt = q
		}
		psuml := chebEva(pt, xl)
		flag := true
		var xm float64
		for flag && xr >= -1 {
			xr = xl - delta
			psumr := chebEva(pt, xr)
			tempR, tempX := psumr, xr
			if psumr*psuml < 0 || psumr == 0 {
				roots++
				for k := 0; k <= nb; k++ {
					xm = (xl + xr) / 2
					psumm := chebEva(pt, xm)
					if psumm*psuml > 0 {
						psuml = psumm
						xl = xm
					} else {
						psumr = psumm
						xr = xm
					}
				}
				freq[j] = xm
				xl = xm
				flag = false
			} else {
				psuml = tempR
				xl = tempX
			}
		}
	}
	for i := 0; i < order; i++ {
		freq[i] = math.Acos(freq[i])
	}
	return roots
}

func lspToLPC(lsp, ak []float64) {
	order := c2LPCOrd
	freq := make([]float32, order)
	for i := 0; i < order; i++ {
		freq[i] = c32cos(lsp[i])
	}
	m := order / 2
	wp := make([]float32, 4*m+2)
	var xin1, xin2 float32 = 1, 1
	n4 := 0
	for j := 0; j <= order; j++ {
		var xout1, xout2 float32
		for i := 0; i < m; i++ {
			n1 := i * 4
			n2 := n1 + 1
			n3 := n2 + 1
			n4 = n3 + 1
			xout1 = xin1 - 2*freq[2*i]*wp[n1] + wp[n2]
			xout2 = xin2 - 2*freq[2*i+1]*wp[n3] + wp[n4]
			wp[n2] = wp[n1]
			wp[n4] = wp[n3]
			wp[n1] = xin1
			wp[n3] = xin2
			xin1, xin2 = xout1, xout2
		}
		xout1 = xin1 + wp[n4+1]
		xout2 = xin2 - wp[n4+2]
		ak[j] = float64((xout1 + xout2) * 0.5)
		wp[n4+1] = xin1
		wp[n4+2] = xin2
		xin1, xin2 = 0, 0
	}
}

func encodeLSPScalar(idx []int, lsp []float64) {
	for i := 0; i < c2LPCOrd; i++ {
		hz := (4000 / c2PI) * lsp[i]
		best, bestE := 0, 1e32
		for j, c := range c2LspCB[i].cb {
			d := c - hz
			e := d * d
			if e < bestE {
				bestE = e
				best = j
			}
		}
		idx[i] = best
	}
}

func decodeLSPScalar(lsp []float64, idx []int) {
	for i := 0; i < c2LPCOrd; i++ {
		lsp[i] = f32((c2PI / 4000) * c2LspCB[i].cb[idx[i]])
	}
}

func checkLSPOrder(lsp []float64) {
	for i := 1; i < c2LPCOrd; i++ {
		if lsp[i] < lsp[i-1] {
			tmp := lsp[i-1]
			lsp[i-1] = lsp[i] - 0.1
			lsp[i] = tmp + 0.1
			i = 1
		}
	}
}

func bwExpand(lsp []float64, minLow, minHigh float64) {
	for i := 1; i < 4; i++ {
		sep := minLow * (c2PI / 4000)
		if lsp[i]-lsp[i-1] < sep {
			lsp[i] = f32(lsp[i-1] + sep)
		}
	}
	for i := 4; i < c2LPCOrd; i++ {
		sep := minHigh * (c2PI / 4000)
		if lsp[i]-lsp[i-1] < sep {
			lsp[i] = f32(lsp[i-1] + sep)
		}
	}
}

func encodeWo(wo float64) int {
	levels := 1 << c2WOBits
	norm := (wo - c2WoMin()) / (c2WoMax() - c2WoMin())
	index := int(math.Floor(float64(levels)*norm + 0.5))
	if index < 0 {
		return 0
	}
	if index > levels-1 {
		return levels - 1
	}
	return index
}

func decodeWo(index int) float64 {
	levels := 1 << c2WOBits
	step := f32((c2WoMax() - c2WoMin()) / float64(levels))
	return f32(c2WoMin() + f32(step*float64(index)))
}

func encodeEnergy(e float64) int {
	if e <= 0 {
		return 0
	}
	levels := 1 << c2EBits
	db := 10 * math.Log10(e)
	norm := (db - c2EMin) / (c2EMax - c2EMin)
	index := int(math.Floor(float64(levels)*norm + 0.5))
	if index < 0 {
		return 0
	}
	if index > levels-1 {
		return levels - 1
	}
	return index
}

func decodeEnergy(index int) float64 {
	levels := 1 << c2EBits
	step := (c2EMax - c2EMin) / float64(levels)
	db := c2EMin + step*float64(index)
	return c2Pow10(db / 10)
}

func c2Pow10(x float64) float64 {
	return math.Exp(c2Ln10 * x)
}

func interpWo(interp, prev, next *c2Model, weight float64) {
	if interp.voiced != 0 && prev.voiced == 0 && next.voiced == 0 {
		interp.voiced = 0
	}
	if interp.voiced != 0 {
		if prev.voiced != 0 && next.voiced != 0 {
			interp.wo = (1-weight)*prev.wo + weight*next.wo
		}
		if prev.voiced == 0 && next.voiced != 0 {
			interp.wo = next.wo
		}
		if prev.voiced != 0 && next.voiced == 0 {
			interp.wo = prev.wo
		}
	} else {
		interp.wo = c2WoMin()
	}
	interp.l = int(c2PI / interp.wo)
}

func interpEnergy(prevE, nextE, weight float64) float64 {
	return c2Pow10((1-weight)*math.Log10(prevE) + weight*math.Log10(nextE))
}

func aksToM2(ak []float64, model *c2Model, e float64, aw []cpx) {
	buf := make([]float64, c2FFT)
	for i := 0; i <= c2LPCOrd; i++ {
		buf[i] = ak[i]
	}
	spec := realFFT(buf)
	copy(aw, spec)
	pw := make([]float32, c2FFT/2)
	for i := range pw {
		re := float32(spec[i].r)
		im := float32(spec[i].i)
		// 1.0 and 1e-6 are double; the product of the bins is float.
		pw[i] = float32(1.0 / (float64(re*re+im*im) + 1e-6))
	}
	lpcPostFilter(pw, ak, float32(e))
	r := float32(c2TwoPI / float64(c2FFT))
	wo := float32(model.wo)
	for m := 1; m <= model.l; m++ {
		am := int((float64(m)-0.5)*float64(wo)/float64(r) + 0.5)
		bm := int((float64(m)+0.5)*float64(wo)/float64(r) + 0.5)
		if bm > c2FFT/2 {
			bm = c2FFT / 2
		}
		var em float32
		for i := am; i < bm; i++ {
			em += pw[i]
		}
		model.a[m] = float64(sqrtf(float64(em)))
	}
}

func lpcPostFilter(pw []float32, ak []float64, e float32) {
	x := make([]float64, c2FFT)
	x[0] = ak[0]
	coeff := float32(c2Gamma)
	for i := 1; i <= c2LPCOrd; i++ {
		x[i] = float64(float32(ak[i]) * coeff)
		coeff *= float32(c2Gamma)
	}
	ww := realFFT(x)
	rw := make([]float32, c2FFT/2)
	for i := range rw {
		re := float32(ww[i].r)
		im := float32(ww[i].i)
		mag2 := re*re + im*im
		rw[i] = sqrtf(float64(mag2 * pw[i]))
	}
	eBefore := float32(1e-4)
	for _, v := range pw {
		eBefore += v
	}
	eAfter := float32(1e-4)
	beta := float32(c2Beta)
	for i := range pw {
		pfw := powf(rw[i], beta)
		pw[i] *= pfw * pfw
		eAfter += pw[i]
	}
	gain := eBefore / eAfter * e
	for i := range pw {
		pw[i] *= gain
	}
	// 1.4*1.4 is a double product, then stored back to float.
	boost := 1.4 * 1.4
	for i := 0; i < c2FFT/8; i++ {
		pw[i] = float32(float64(pw[i]) * boost)
	}
}

func (c *c2Codec) synthOne(model *c2Model, speech []int16, aw []cpx) {
	h := make([]cpx, c2MaxAmp+1)
	r := c2TwoPI / float64(c2FFT)
	for m := 1; m <= model.l; m++ {
		b := int(float64(m)*model.wo/r + 0.5)
		h[m].r = aw[b].r
		h[m].i = -aw[b].i
	}
	c.phaseSynth(model, h)
	c.postFilter(model)
	c.synthesise(model)
	earProtect(c.snSynth[:c2NSamp])
	for i := 0; i < c2NSamp; i++ {
		speech[i] = c2ToInt16(c.snSynth[i])
	}
}

func (c *c2Codec) phaseSynth(model *c2Model, h []cpx) {
	// ex_phase is float. The floorf argument is a double sum narrowed to float.
	ex := float32(c.exPhase)
	ex += float32(model.wo) * float32(c2NSamp)
	floorArg := float64(ex)/c2TwoPI + 0.5
	floored := math.Floor(float64(float32(floorArg)))
	ex = float32(float64(ex) - c2TwoPI*floored)
	c.exPhase = float64(ex)
	for m := 1; m <= model.l; m++ {
		var er, ei float32
		if model.voiced != 0 {
			ang := ex * float32(m)
			er = c32cos(float64(ang))
			ei = c32sin(float64(ang))
		} else {
			phi := f32(c2TwoPI * float64(c.rand()) / 32767)
			er = c32cos(phi)
			ei = c32sin(phi)
		}
		hr := float32(h[m].r)
		hi := float32(h[m].i)
		ar := hr*er - hi*ei
		ai := hi*er + hr*ei
		model.phi[m] = float64(c32atan2(ai, float32(float64(ar)+1e-12)))
	}
}

func c32atan2(y, x float32) float32 {
	return float32(math.Atan2(float64(y), float64(x)))
}

func (c *c2Codec) postFilter(model *c2Model) {
	e := 1e-12
	for m := 1; m <= model.l; m++ {
		e += model.a[m] * model.a[m]
	}
	e = 10 * math.Log10(e/float64(model.l))
	if e < 40 && model.voiced == 0 {
		c.bgEst = c.bgEst*(1-0.1) + e*0.1
	}
	thresh := c2Pow10((c.bgEst + 6) / 20)
	if model.voiced != 0 {
		for m := 1; m <= model.l; m++ {
			if model.a[m] < thresh {
				model.phi[m] = f32((c2TwoPI / 32767) * float64(c.rand()))
			}
		}
	}
}

func (c *c2Codec) synthesise(model *c2Model) {
	copy(c.snSynth[:c2NSamp-1], c.snSynth[c2NSamp:2*c2NSamp-1])
	c.snSynth[c2NSamp-1] = 0
	spec := make([]cpx, c2FFT/2+1)
	for l := 1; l <= model.l; l++ {
		b := int(float64(l)*model.wo*float64(c2FFT)/c2TwoPI + 0.5)
		if b > c2FFT/2-1 {
			b = c2FFT/2 - 1
		}
		ar := float32(model.a[l]) * c32cos(model.phi[l])
		ai := float32(model.a[l]) * c32sin(model.phi[l])
		spec[b].r = float64(ar)
		spec[b].i = float64(ai)
	}
	sw := realIFFT(spec)
	for i := 0; i < c2NSamp-1; i++ {
		c.snSynth[i] = f32(c.snSynth[i] + f32(sw[c2FFT-c2NSamp+1+i]*c.pn[i]))
	}
	for i, j := c2NSamp-1, 0; i < 2*c2NSamp; i, j = i+1, j+1 {
		c.snSynth[i] = f32(sw[j] * c.pn[i])
	}
}

func earProtect(x []float64) {
	maxS := 0.0
	for _, v := range x {
		if v > maxS {
			maxS = v
		}
	}
	over := maxS / 30000
	if over > 1 {
		gain := 1 / (over * over)
		for i := range x {
			x[i] *= gain
		}
	}
}

func c2ToInt16(v float64) int16 {
	if v > 32767 {
		return 32767
	}
	if v < -32767 {
		return -32767
	}
	return int16(v)
}

// codec2_rand is a 64-bit LCG. unsigned long on this platform does not wrap at 32 bits.
func (c *c2Codec) rand() int {
	c.randNext = c.randNext*1103515245 + 12345
	return int(uint32(c.randNext>>16) % 32768)
}

func packBits(bits []byte, nbit *uint, field int, width uint, gray bool) {
	if gray {
		field = (field >> 1) ^ field
	}
	f := uint(field)
	w := width
	for {
		bI := *nbit
		left := 8 - (bI & 7)
		slice := w
		if left < slice {
			slice = left
		}
		bits[bI>>3] |= byte((f >> (w - slice)) << (left - slice))
		*nbit = bI + slice
		w -= slice
		if w == 0 {
			return
		}
	}
}

func unpackBits(bits []byte, nbit *uint, width uint, gray bool) int {
	var field uint
	w := width
	for {
		bI := *nbit
		left := 8 - (bI & 7)
		slice := w
		if left < slice {
			slice = left
		}
		got := (uint(bits[bI>>3]) >> (left - slice)) & ((1 << slice) - 1)
		field |= got << (w - slice)
		*nbit = bI + slice
		w -= slice
		if w == 0 {
			break
		}
	}
	if gray {
		field ^= field >> 8
		field ^= field >> 4
		field ^= field >> 2
		field ^= field >> 1
	}
	return int(field)
}

func fftInPlace(a []cpx, inverse bool) {
	n := len(a)
	j := 0
	for i := 1; i < n; i++ {
		bit := n >> 1
		for j&bit != 0 {
			j ^= bit
			bit >>= 1
		}
		j ^= bit
		if i < j {
			a[i], a[j] = a[j], a[i]
		}
	}
	for length := 2; length <= n; length <<= 1 {
		ang := 2 * math.Pi / float64(length)
		if !inverse {
			ang = -ang
		}
		wlenR, wlenI := math.Cos(ang), math.Sin(ang)
		half := length >> 1
		for i := 0; i < n; i += length {
			wr, wi := 1.0, 0.0
			for k := 0; k < half; k++ {
				ur, ui := a[i+k].r, a[i+k].i
				idx := i + k + half
				vr := a[idx].r*wr - a[idx].i*wi
				vi := a[idx].r*wi + a[idx].i*wr
				a[i+k].r = ur + vr
				a[i+k].i = ui + vi
				a[idx].r = ur - vr
				a[idx].i = ui - vi
				nwr := wr*wlenR - wi*wlenI
				wi = wr*wlenI + wi*wlenR
				wr = nwr
			}
		}
	}
}

// realFFT and realIFFT follow kiss_fftr / kiss_fftri. A full complex FFT of the
// same length is a different rounding, and that shows up in the int16 PCM.
func realFFT(x []float64) []cpx {
	return kissFFTR(x)
}

func realIFFT(spec []cpx) []float64 {
	return kissFFTRI(spec)
}

type c32 struct{ r, i float32 }

type kissCfg struct {
	inverse bool
	tw      []c32
	fac     []int
	super   []c32 // set on the real-FFT wrapper's complex half-size state
}

func newKiss(n int, inverse bool) *kissCfg {
	st := &kissCfg{inverse: inverse, tw: make([]c32, n)}
	if n == 256 {
		bits := &kissTwFwdBits
		if inverse {
			bits = &kissTwInvBits
		}
		for i := 0; i < n; i++ {
			st.tw[i] = c32{math.Float32frombits(bits[2*i]), math.Float32frombits(bits[2*i+1])}
		}
	} else {
		const pi = 3.141592653589793238462643383279502884197169399375105820974944
		for i := 0; i < n; i++ {
			phase := -2 * pi * float64(i) / float64(n)
			if inverse {
				phase = -phase
			}
			st.tw[i] = c32{c32cos(phase), c32sin(phase)}
		}
	}
	st.fac = kissFactor(n)
	return st
}

func kissFactor(n int) []int {
	p := 4
	floorSqrt := int(math.Floor(math.Sqrt(float64(n))))
	var fac []int
	for n > 1 {
		for n%p != 0 {
			switch p {
			case 4:
				p = 2
			case 2:
				p = 3
			default:
				p += 2
			}
			if p > floorSqrt {
				p = n
			}
		}
		n /= p
		fac = append(fac, p, n)
	}
	return fac
}

func c32cos(phase float64) float32 { return glibcCosf(float32(phase)) }

func c32sin(phase float64) float32 { return glibcSinf(float32(phase)) }

// glibc 2.36 sinf/cosf (scalar range reduction). math.Cos is a different
// rounding, and libcodec2 calls libm.
type glibcSincos struct {
	sign               [4]float64
	hpiInv, hpi        float64
	c0, c1, c2, c3, c4 float64
	s1, s2, s3         float64
}

var glibcSincosTab = [2]glibcSincos{
	{
		sign:   [4]float64{1, -1, -1, 1},
		hpiInv: 0x1.45F306DC9C883p+23,
		hpi:    0x1.921FB54442D18p0,
		c0:     0x1p0,
		c1:     -0x1.ffffffd0c621cp-2,
		c2:     0x1.55553e1068f19p-5,
		c3:     -0x1.6c087e89a359dp-10,
		c4:     0x1.99343027bf8c3p-16,
		s1:     -0x1.555545995a603p-3,
		s2:     0x1.1107605230bc4p-7,
		s3:     -0x1.994eb3774cf24p-13,
	},
	{
		sign:   [4]float64{1, -1, -1, 1},
		hpiInv: 0x1.45F306DC9C883p+23,
		hpi:    0x1.921FB54442D18p0,
		c0:     -0x1p0,
		c1:     0x1.ffffffd0c621cp-2,
		c2:     -0x1.55553e1068f19p-5,
		c3:     0x1.6c087e89a359dp-10,
		c4:     -0x1.99343027bf8c3p-16,
		s1:     -0x1.555545995a603p-3,
		s2:     0x1.1107605230bc4p-7,
		s3:     -0x1.994eb3774cf24p-13,
	},
}

var glibcInvPio4 = [24]uint32{
	0xa2, 0xa2f9, 0xa2f983, 0xa2f9836e,
	0xf9836e4e, 0x836e4e44, 0x6e4e4415, 0x4e441529,
	0x441529fc, 0x1529fc27, 0x29fc2757, 0xfc2757d1,
	0x2757d1f5, 0x57d1f534, 0xd1f534dd, 0xf534ddc0,
	0x34ddc0db, 0xddc0db62, 0xc0db6295, 0xdb629599,
	0x6295993c, 0x95993c43, 0x993c4390, 0x3c439041,
}

func glibcAbstop12(x float32) uint32 {
	return (math.Float32bits(x) >> 20) & 0x7ff
}

func glibcSinfPoly(x, x2 float64, p *glibcSincos, n int) float32 {
	if n&1 == 0 {
		x3 := x * x2
		s1 := p.s2 + x2*p.s3
		x7 := x3 * x2
		s := x + x3*p.s1
		return float32(s + x7*s1)
	}
	x4 := x2 * x2
	c2 := p.c3 + x2*p.c4
	c1 := p.c0 + x2*p.c1
	x6 := x4 * x2
	c := c1 + x4*p.c2
	return float32(c + x6*c2)
}

func glibcReduceFast(x float64, p *glibcSincos) (float64, int) {
	r := x * p.hpiInv
	n := (int32(r) + 0x800000) >> 24
	return x - float64(n)*p.hpi, int(n)
}

func glibcReduceLarge(xi uint32) (float64, int) {
	arr := (xi >> 26) & 15
	shift := (xi >> 23) & 7
	xi = (xi & 0xffffff) | 0x800000
	xi <<= shift
	res0 := uint64(xi * glibcInvPio4[arr])
	res1 := uint64(xi) * uint64(glibcInvPio4[arr+4])
	res2 := uint64(xi) * uint64(glibcInvPio4[arr+8])
	res0 = (res2 >> 32) | (res0 << 32)
	res0 += res1
	n := (res0 + (1 << 61)) >> 62
	res0 -= n << 62
	const pi63 = 0x1.921FB54442D18p-62
	return float64(int64(res0)) * pi63, int(n)
}

func glibcCosf(y float32) float32 {
	const pio4 float32 = 0x1.921FB6p-1
	const tiny float32 = 0x1p-12
	const lim float32 = 120
	p := &glibcSincosTab[0]
	if glibcAbstop12(y) < glibcAbstop12(pio4) {
		if glibcAbstop12(y) < glibcAbstop12(tiny) {
			return 1
		}
		x := float64(y)
		return glibcSinfPoly(x, x*x, p, 1)
	}
	if glibcAbstop12(y) < glibcAbstop12(lim) {
		x, n := glibcReduceFast(float64(y), p)
		s := p.sign[n&3]
		if n&2 != 0 {
			p = &glibcSincosTab[1]
		}
		return glibcSinfPoly(x*s, x*x, p, n^1)
	}
	if glibcAbstop12(y) < glibcAbstop12(float32(math.Inf(1))) {
		xi := math.Float32bits(y)
		sign := int(xi >> 31)
		x, n := glibcReduceLarge(xi)
		s := glibcSincosTab[0].sign[(n+sign)&3]
		p = &glibcSincosTab[0]
		if (n+sign)&2 != 0 {
			p = &glibcSincosTab[1]
		}
		return glibcSinfPoly(x*s, x*x, p, n^1)
	}
	return float32(math.NaN())
}

func glibcSinf(y float32) float32 {
	const pio4 float32 = 0x1.921FB6p-1
	const tiny float32 = 0x1p-12
	const lim float32 = 120
	p := &glibcSincosTab[0]
	if glibcAbstop12(y) < glibcAbstop12(pio4) {
		x := float64(y)
		x2 := x * x
		if glibcAbstop12(y) < glibcAbstop12(tiny) {
			return y
		}
		return glibcSinfPoly(x, x2, p, 0)
	}
	if glibcAbstop12(y) < glibcAbstop12(lim) {
		x, n := glibcReduceFast(float64(y), p)
		s := p.sign[n&3]
		if n&2 != 0 {
			p = &glibcSincosTab[1]
		}
		return glibcSinfPoly(x*s, x*x, p, n)
	}
	if glibcAbstop12(y) < glibcAbstop12(float32(math.Inf(1))) {
		xi := math.Float32bits(y)
		sign := int(xi >> 31)
		x, n := glibcReduceLarge(xi)
		s := glibcSincosTab[0].sign[(n+sign)&3]
		p = &glibcSincosTab[0]
		if (n+sign)&2 != 0 {
			p = &glibcSincosTab[1]
		}
		return glibcSinfPoly(x*s, x*x, p, n)
	}
	return float32(math.NaN())
}

func c32mul(a, b c32) c32 {
	return c32{a.r*b.r - a.i*b.i, a.r*b.i + a.i*b.r}
}

func kissWork(fout []c32, fin []c32, fstride, inStride int, fac []int, st *kissCfg) {
	p := fac[0]
	m := fac[1]
	if m == 1 {
		for i := 0; i < p; i++ {
			fout[i] = fin[i*fstride*inStride]
		}
	} else {
		for k := 0; k < p; k++ {
			kissWork(fout[k*m:], fin[k*fstride*inStride:], fstride*p, inStride, fac[2:], st)
		}
	}
	switch p {
	case 2:
		kissBfly2(fout, fstride, m, st)
	case 4:
		kissBfly4(fout, fstride, m, st)
	default:
		panic("kiss fft radix")
	}
}

func kissBfly2(fout []c32, fstride, m int, st *kissCfg) {
	f2 := m
	tw := 0
	for k := 0; k < m; k++ {
		t := c32mul(fout[f2], st.tw[tw])
		tw += fstride
		fr, fi := fout[k].r, fout[k].i
		fout[f2] = c32{fr - t.r, fi - t.i}
		fout[k] = c32{fr + t.r, fi + t.i}
		f2++
	}
}

func kissBfly4(fout []c32, fstride, m int, st *kissCfg) {
	m2 := 2 * m
	m3 := 3 * m
	tw1, tw2, tw3 := 0, 0, 0
	for k := 0; k < m; k++ {
		s0 := c32mul(fout[k+m], st.tw[tw1])
		s1 := c32mul(fout[k+m2], st.tw[tw2])
		s2 := c32mul(fout[k+m3], st.tw[tw3])
		fr, fi := fout[k].r, fout[k].i
		s5r, s5i := fr-s1.r, fi-s1.i
		fr += s1.r
		fi += s1.i
		s3r, s3i := s0.r+s2.r, s0.i+s2.i
		s4r, s4i := s0.r-s2.r, s0.i-s2.i
		fout[k+m2] = c32{fr - s3r, fi - s3i}
		tw1 += fstride
		tw2 += fstride * 2
		tw3 += fstride * 3
		fout[k] = c32{fr + s3r, fi + s3i}
		if st.inverse {
			fout[k+m] = c32{s5r - s4i, s5i + s4r}
			fout[k+m3] = c32{s5r + s4i, s5i - s4r}
		} else {
			fout[k+m] = c32{s5r + s4i, s5i - s4r}
			fout[k+m3] = c32{s5r - s4i, s5i + s4r}
		}
	}
}

func kissFFT(st *kissCfg, fin []c32) []c32 {
	fout := make([]c32, len(st.tw))
	kissWork(fout, fin, 1, 1, st.fac, st)
	return fout
}

var (
	kissFwd512 = newKissFFTR(512, false)
	kissInv512 = newKissFFTR(512, true)
)

func newKissFFTR(n int, inverse bool) *kissCfg {
	if n%2 != 0 {
		panic("kiss fftr length")
	}
	nc := n / 2
	st := newKiss(nc, inverse)
	st.super = make([]c32, nc/2)
	bits := kissSuperFwdBits
	if inverse {
		bits = kissSuperInvBits
	}
	if len(bits) != nc {
		// 512-point real FFT only; other lengths rebuild from libm-style cos.
		for i := 0; i < nc/2; i++ {
			t := float64(float32(i+1) / float32(nc))
			phase := -3.14159265358979323846264338327 * (t + 0.5)
			if inverse {
				phase = -phase
			}
			st.super[i] = c32{c32cos(phase), c32sin(phase)}
		}
		return st
	}
	for i := 0; i < nc/2; i++ {
		st.super[i] = c32{math.Float32frombits(bits[2*i]), math.Float32frombits(bits[2*i+1])}
	}
	return st
}

func kissFFTR(x []float64) []cpx {
	st := kissFwd512
	nc := len(st.tw)
	fin := make([]c32, nc)
	for i := 0; i < nc; i++ {
		fin[i] = c32{float32(x[2*i]), float32(x[2*i+1])}
	}
	tmp := kissFFT(st, fin)
	out := make([]cpx, nc+1)
	out[0] = cpx{float64(tmp[0].r + tmp[0].i), 0}
	out[nc] = cpx{float64(tmp[0].r - tmp[0].i), 0}
	for k := 1; k <= nc/2; k++ {
		fpk := tmp[k]
		fpnk := c32{tmp[nc-k].r, -tmp[nc-k].i}
		f1 := c32{fpk.r + fpnk.r, fpk.i + fpnk.i}
		f2 := c32{fpk.r - fpnk.r, fpk.i - fpnk.i}
		tw := c32mul(f2, st.super[k-1])
		out[k] = cpx{float64(0.5 * (f1.r + tw.r)), float64(0.5 * (f1.i + tw.i))}
		out[nc-k] = cpx{float64(0.5 * (f1.r - tw.r)), float64(0.5 * (tw.i - f1.i))}
	}
	return out
}

func kissFFTRI(spec []cpx) []float64 {
	st := kissInv512
	nc := len(st.tw)
	tmpin := make([]c32, nc)
	dc := float32(spec[0].r)
	ny := float32(spec[nc].r)
	tmpin[0] = c32{dc + ny, dc - ny}
	for k := 1; k <= nc/2; k++ {
		fk := c32{float32(spec[k].r), float32(spec[k].i)}
		fnkc := c32{float32(spec[nc-k].r), -float32(spec[nc-k].i)}
		fek := c32{fk.r + fnkc.r, fk.i + fnkc.i}
		diff := c32{fk.r - fnkc.r, fk.i - fnkc.i}
		fok := c32mul(diff, st.super[k-1])
		tmpin[k] = c32{fek.r + fok.r, fek.i + fok.i}
		nk := c32{fek.r - fok.r, fek.i - fok.i}
		nk.i = -nk.i
		tmpin[nc-k] = nk
	}
	timed := kissFFT(st, tmpin)
	out := make([]float64, nc*2)
	for i, v := range timed {
		out[2*i] = float64(v.r)
		out[2*i+1] = float64(v.i)
	}
	return out
}
