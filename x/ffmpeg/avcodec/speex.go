package avcodec

import (
	"fmt"
	"math"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

const (
	speexNBOrder     = 10
	speexNBFrame     = 160
	speexNBSub       = 40
	speexNBNSub      = 4
	speexPitchStart  = 17
	speexPitchEnd    = 144
	speexWindow      = 200
	speexExcBuf      = speexNBFrame + speexPitchEnd + 2
	speexVerySmall   = 1e-15
	speexLSPMargin   = 0.002
	speexLSPDelta1   = 0.2
	speexComplexity  = 3
	speexSubmodeID   = 5
	speexGainBits    = 7
	speexPitchBits   = 7
	speexShapeBits   = 6
	speexSubvect     = 5
	speexNBSubvect   = 8
	speexShapeCBSize = 64
	speexPi          = math.Pi
)

var speexExcGainBound3 = [7]float32{0.112338, 0.236980, 0.369316, 0.492054, 0.637471, 0.828874, 1.132784}
var speexExcGainScal3 = [8]float32{0.061130, 0.163546, 0.310413, 0.428220, 0.555887, 0.719055, 0.938694, 1.326874}

var speexHPDen = [3]float32{1.00000, -1.91120, 0.91498}
var speexHPNum = [3]float32{0.95654, -1.91309, 0.95654}

type speexBits struct {
	chars   []byte
	nbBits  int
	charPtr int
	bitPtr  int
}

func (b *speexBits) reset() {
	if len(b.chars) < 256 {
		b.chars = make([]byte, 256)
	}
	b.chars[0] = 0
	b.nbBits = 0
	b.charPtr = 0
	b.bitPtr = 0
}

func (b *speexBits) pack(data, nbits int) {
	d := uint(data)
	for nbits > 0 {
		if b.charPtr >= len(b.chars) {
			b.chars = append(b.chars, 0)
		}
		bit := byte((d >> (nbits - 1)) & 1)
		b.chars[b.charPtr] |= bit << (7 - b.bitPtr)
		b.bitPtr++
		if b.bitPtr == 8 {
			b.bitPtr = 0
			b.charPtr++
			if b.charPtr >= len(b.chars) {
				b.chars = append(b.chars, 0)
			} else {
				b.chars[b.charPtr] = 0
			}
		}
		b.nbBits++
		nbits--
	}
}

func (b *speexBits) insertTerminator() {
	if b.bitPtr != 0 {
		b.pack(0, 1)
	}
	for b.bitPtr != 0 {
		b.pack(1, 1)
	}
}

func (b *speexBits) write() []byte {
	b.insertTerminator()
	n := (b.nbBits + 7) >> 3
	out := make([]byte, n)
	copy(out, b.chars[:n])
	return out
}

type speexEnc struct {
	first        bool
	boundedPitch bool
	cumulGain    float32
	gamma1       float32
	gamma2       float32
	lpcFloor     float32
	complexity   int
	submodeID    int
	highpass     bool
	winBuf       [speexWindow - speexNBFrame]float32
	excBuf       [speexExcBuf]float32
	swBuf        [speexExcBuf]float32
	oldLSP       [speexNBOrder]float32
	oldQLSP      [speexNBOrder]float32
	memSP        [speexNBOrder]float32
	memSW        [speexNBOrder]float32
	memSWWhole   [speexNBOrder]float32
	memExc       [speexNBOrder]float32
	memExc2      [speexNBOrder]float32
	memHP        [2]float32
	piGain       [speexNBNSub]float32
}

func newSpeexEnc() *speexEnc {
	st := &speexEnc{
		first:        true,
		boundedPitch: true,
		cumulGain:    1024,
		gamma1:       0.92,
		gamma2:       0.6,
		lpcFloor:     0.0002,
		complexity:   speexComplexity,
		submodeID:    speexSubmodeID,
		highpass:     true,
	}
	for i := 0; i < speexNBOrder; i++ {
		st.oldLSP[i] = float32(3.1415927) * float32(i+1) / float32(speexNBOrder+1)
	}
	return st
}

// SpeexEncodeNB encodes 8 kHz mono PCM as Speex narrowband quality 8
// (ffmpeg -c:a libspeex defaults: CBR quality 8, complexity 3, one frame
// per packet). A trailing partial frame of fewer than 160 samples is dropped.
func SpeexEncodeNB(pcm []int16) ([]byte, error) {
	if pcm == nil {
		return nil, fmt.Errorf("%w: speex pcm", avutil.ErrInvalidData)
	}
	nframes := len(pcm) / speexNBFrame
	if nframes == 0 {
		return []byte{}, nil
	}
	st := newSpeexEnc()
	var bits speexBits
	out := make([]byte, 0, nframes*38)
	in := make([]float32, speexNBFrame)
	for f := 0; f < nframes; f++ {
		frame := pcm[f*speexNBFrame : (f+1)*speexNBFrame]
		for i, s := range frame {
			in[i] = float32(s)
		}
		bits.reset()
		st.encode(in, &bits)
		out = append(out, bits.write()...)
	}
	return out, nil
}

func (st *speexEnc) encode(in []float32, bits *speexBits) {
	excOff := speexPitchEnd + 2
	swOff := speexPitchEnd + 2
	copy(st.excBuf[:speexPitchEnd+2], st.excBuf[speexNBFrame:speexNBFrame+speexPitchEnd+2])
	copy(st.swBuf[:speexPitchEnd+2], st.swBuf[speexNBFrame:speexNBFrame+speexPitchEnd+2])

	if st.highpass {
		speexHighpass(in, in, st.memHP[:])
	}

	var lpc, bw1, bw2 [speexNBOrder]float32
	var lsp, qlsp, interpLSP, interpQLSP [speexNBOrder]float32
	var interpLPC, interpQLPC [speexNBOrder]float32

	var wSig [speexWindow]float32
	var autocorr [speexNBOrder + 1]float32
	for i := 0; i < speexWindow-speexNBFrame; i++ {
		wSig[i] = st.winBuf[i] * speexLPCWindow[i]
	}
	for i := speexWindow - speexNBFrame; i < speexWindow; i++ {
		wSig[i] = in[i-speexWindow+speexNBFrame] * speexLPCWindow[i]
	}
	speexAutocorr(wSig[:], autocorr[:], speexNBOrder+1, speexWindow)
	autocorr[0] += autocorr[0] * st.lpcFloor
	for i := 0; i < speexNBOrder+1; i++ {
		autocorr[i] *= speexLagWindow[i]
	}
	autocorr[0] += 1
	speexLPC(lpc[:], autocorr[:], speexNBOrder)
	roots := speexLPCToLSP(lpc[:], lsp[:], 10, speexLSPDelta1)
	if roots != speexNBOrder {
		copy(lsp[:], st.oldLSP[:])
	}

	diff := speexWindow - speexNBFrame
	if st.first {
		copy(interpLSP[:], lsp[:])
	} else {
		speexLSPInterpolate(st.oldLSP[:], lsp[:], interpLSP[:], speexNBNSub, speexNBNSub<<1)
	}
	speexLSPToLPC(interpLSP[:], interpLPC[:])

	olPitch := 0
	olPitchCoef := float32(0)

	speexFIRMem(st.winBuf[:], interpLPC[:], st.excBuf[excOff:], diff, st.memExc[:])
	speexFIRMem(in, interpLPC[:], st.excBuf[excOff+diff:], speexNBFrame-diff, st.memExc[:])
	g := speexRMS(st.excBuf[excOff : excOff+speexNBFrame])
	olGain := g
	if st.submodeID != 1 && olPitch > 0 {
		c := olPitchCoef * olPitchCoef
		olGain = g * (1.1 * float32(math.Sqrt(float64(1-0.8*c))))
	}

	bits.pack(0, 1)
	bits.pack(st.submodeID, 4)

	if st.first {
		copy(st.oldLSP[:], lsp[:])
	}
	speexLSPQuantNB(lsp[:], qlsp[:], bits)

	qe := int(math.Floor(0.5 + 3.5*math.Log(float64(olGain))))
	if qe < 0 {
		qe = 0
	}
	if qe > 31 {
		qe = 31
	}
	olGain = float32(math.Exp(float64(qe) / 3.5))
	bits.pack(qe, 5)

	if st.first {
		copy(st.oldQLSP[:], qlsp[:])
	}

	var target [speexNBSub]float32
	var innov [speexNBSub]float32
	var exc32 [speexNBSub]float32
	var synResp [speexNBSub]float32
	var mem [speexNBOrder]float32

	for sub := 0; sub < speexNBNSub; sub++ {
		offset := speexNBSub * sub
		exc := st.excBuf[excOff+offset:]
		sw := st.swBuf[swOff+offset:]

		speexLSPInterpolate(st.oldLSP[:], lsp[:], interpLSP[:], sub, speexNBNSub)
		speexLSPInterpolate(st.oldQLSP[:], qlsp[:], interpQLSP[:], sub, speexNBNSub)
		speexLSPToLPC(interpLSP[:], interpLPC[:])
		speexLSPToLPC(interpQLSP[:], interpQLPC[:])

		piG := float32(1)
		for i := 0; i < speexNBOrder; i += 2 {
			piG += interpQLPC[i+1] - interpQLPC[i]
		}
		st.piGain[sub] = piG

		speexBWLPC(st.gamma1, interpLPC[:], bw1[:])
		speexBWLPC(st.gamma2, interpLPC[:], bw2[:])

		var inBuf []float32
		if sub == 0 {
			inBuf = st.winBuf[:]
		} else {
			inBuf = in[(sub-1)*speexNBSub:]
		}
		copy(sw[:speexNBSub], inBuf[:speexNBSub])

		speexImpulse(interpQLPC[:], bw1[:], bw2[:], synResp[:], speexNBSub, speexNBOrder)

		for i := 0; i < speexNBOrder; i++ {
			mem[i] = st.memSP[i]
		}
		for i := 0; i < speexNBSub; i++ {
			exc[i] = speexVerySmall
		}
		speexIIRMem(exc, interpQLPC[:], exc, speexNBSub, mem[:])
		for i := 0; i < speexNBOrder; i++ {
			mem[i] = st.memSW[i]
		}
		speexFilterMem(exc, bw1[:], bw2[:], exc, mem[:], speexNBSub, speexNBOrder)

		copy(mem[:], st.memSW[:])
		speexFilterMem(sw, bw1[:], bw2[:], sw, mem[:], speexNBSub, speexNBOrder)

		for i := 0; i < speexNBSub; i++ {
			target[i] = sw[i] - exc[i]
		}
		copy(exc[:speexNBSub], inBuf[:speexNBSub])
		speexFIRMem(exc, interpQLPC[:], exc, speexNBSub, st.memExc2[:])

		pitMin := speexPitchStart
		pitMax := speexPitchEnd
		if st.boundedPitch && pitMax > offset {
			pitMax = offset
		}
		speexPitchSearch3Tap(target[:], st.swBuf[:], swOff+offset, interpQLPC[:], bw1[:], bw2[:], exc32[:], pitMin, pitMax, olPitchCoef, bits, st.excBuf[:], excOff+offset, synResp[:], st.complexity, &st.cumulGain)

		for i := 0; i < speexNBSub; i++ {
			innov[i] = 0
			exc[i] = exc[i] - exc32[i]
		}
		ener := speexRMS(exc[:speexNBSub])
		fineGain := ener / olGain
		qsf := speexScalQuant(fineGain, speexExcGainBound3[:], 8)
		bits.pack(qsf, 3)
		ener = speexExcGainScal3[qsf] * olGain
		inv := float32(1) / ener
		for i := 0; i < speexNBSub; i++ {
			target[i] *= inv
		}
		speexSplitCB(target[:], interpQLPC[:], bw1[:], bw2[:], speexNBOrder, speexNBSub, innov[:], synResp[:], bits, st.complexity, 0)
		for i := 0; i < speexNBSub; i++ {
			innov[i] *= ener
			exc[i] = exc32[i] + innov[i]
		}
		speexIIRMem(exc, interpQLPC[:], sw, speexNBSub, st.memSP[:])
		speexFilterMem(sw, bw1[:], bw2[:], sw, st.memSW[:], speexNBSub, speexNBOrder)
	}

	copy(st.oldLSP[:], lsp[:])
	copy(st.oldQLSP[:], qlsp[:])
	st.first = false
	copy(st.winBuf[:], in[2*speexNBFrame-speexWindow:])
	st.boundedPitch = false
}

func speexHighpass(x, y []float32, mem []float32) {
	den := speexHPDen
	num := speexHPNum
	for i := range x {
		vout := num[0]*x[i] + mem[0]
		yi := vout
		mem[0] = mem[1] + num[1]*x[i] + (-den[1])*vout
		mem[1] = num[2]*x[i] + (-den[2])*vout
		y[i] = yi
	}
}

func speexAutocorr(x, ac []float32, lag, n int) {
	for l := lag - 1; l >= 0; l-- {
		d := float32(0)
		for i := l; i < n; i++ {
			d += x[i] * x[i-l]
		}
		ac[l] = d
	}
	ac[0] += 10
}

func speexLPC(lpc, ac []float32, p int) {
	error := ac[0]
	for i := 0; i < p; i++ {
		rr := -ac[i+1]
		for j := 0; j < i; j++ {
			rr -= lpc[j] * ac[i-j]
		}
		r := speexF32(float64(rr) / (float64(error) + 0.003*float64(ac[0])))
		lpc[i] = r
		for j := 0; j < (i+1)>>1; j++ {
			tmp1 := lpc[j]
			tmp2 := lpc[i-1-j]
			lpc[j] = tmp1 + r*tmp2
			lpc[i-1-j] = tmp2 + r*tmp1
		}
		error -= r * (error * r)
	}
}

func speexF32(v float64) float32 { return float32(v) }

func speexCheb(coef []float32, x float32, m int) float32 {
	b0 := float32(0)
	b1 := float32(0)
	x *= 2
	for k := m; k > 0; k-- {
		tmp := b0
		b0 = x*b0 - b1 + coef[m-k]
		b1 = tmp
	}
	return speexF32(float64(-b1) + 0.5*float64(x)*float64(b0) + float64(coef[m]))
}

func speexCos(x float32) float32 {
	const (
		c1     = 0.9999932946
		c2     = -0.4999124376
		c3     = 0.0414877472
		c4     = -0.0012712095
		spxPi2 = 1.5707963268
	)
	if x < spxPi2 {
		x *= x
		return c1 + x*(c2+x*(c3+c4*x))
	}
	x = speexF32(speexPi - float64(x))
	x *= x
	return -(c1 + x*(c2+x*(c3+c4*x)))
}

func speexLPCToLSP(a, freq []float32, nb int, delta float32) int {
	m := speexNBOrder / 2
	p := make([]float32, m+1)
	q := make([]float32, m+1)
	p[0] = 1
	q[0] = 1
	for i := 0; i < m; i++ {
		p[i+1] = (a[i] + a[speexNBOrder-1-i]) - p[i]
		q[i+1] = (a[i] - a[speexNBOrder-1-i]) + q[i]
	}
	for i := 0; i < m; i++ {
		p[i] *= 2
		q[i] *= 2
	}
	xr := float32(0)
	xl := float32(1)
	roots := 0
	for j := 0; j < speexNBOrder; j++ {
		pt := p
		if j&1 != 0 {
			pt = q
		}
		psuml := speexCheb(pt, xl, m)
		for xr >= -1 {
			dd := speexF32(float64(delta) * (1 - 0.9*float64(xl)*float64(xl)))
			if math.Abs(float64(psuml)) < 0.2 {
				dd = speexF32(float64(dd) * 0.5)
			}
			xr = xl - dd
			psumr := speexCheb(pt, xr, m)
			tempPsumr := psumr
			tempXr := xr
			if psumr*psuml < 0 {
				roots++
				xm := float32(0)
				for k := 0; k <= nb; k++ {
					xm = speexF32(0.5 * float64(xl+xr))
					psumm := speexCheb(pt, xm, m)
					if psumm*psuml >= 0 {
						psuml = psumm
						xl = xm
					} else {
						psumr = psumm
						xr = xm
					}
				}
				freq[j] = float32(math.Acos(float64(xm)))
				xl = xm
				break
			}
			psuml = tempPsumr
			xl = tempXr
		}
	}
	return roots
}

func speexLSPToLPC(freq, ak []float32) {
	m := speexNBOrder >> 1
	wp := make([]float32, 4*m+2)
	xin1 := float32(1)
	xin2 := float32(1)
	xFreq := make([]float32, speexNBOrder)
	for i := 0; i < speexNBOrder; i++ {
		xFreq[i] = speexCos(freq[i])
	}
	var n4 int
	for j := 0; j <= speexNBOrder; j++ {
		i2 := 0
		var xout1, xout2 float32
		for i := 0; i < m; i++ {
			n1 := i * 4
			n2 := n1 + 1
			n3 := n2 + 1
			n4 = n3 + 1
			xout1 = xin1 - 2*xFreq[i2]*wp[n1] + wp[n2]
			xout2 = xin2 - 2*xFreq[i2+1]*wp[n3] + wp[n4]
			wp[n2] = wp[n1]
			wp[n4] = wp[n3]
			wp[n1] = xin1
			wp[n3] = xin2
			xin1 = xout1
			xin2 = xout2
			i2 += 2
		}
		xout1 = xin1 + wp[n4+1]
		xout2 = xin2 - wp[n4+2]
		if j > 0 {
			ak[j-1] = (xout1 + xout2) * 0.5
		}
		wp[n4+1] = xin1
		wp[n4+2] = xin2
		xin1 = 0
		xin2 = 0
	}
}

func speexLSPInterpolate(oldLSP, newLSP, lsp []float32, subframe, nsub int) {
	tmp := (1.0 + float32(subframe)) / float32(nsub)
	for i := 0; i < speexNBOrder; i++ {
		lsp[i] = (1-tmp)*oldLSP[i] + tmp*newLSP[i]
	}
	if lsp[0] < speexLSPMargin {
		lsp[0] = speexLSPMargin
	}
	if lsp[speexNBOrder-1] > float32(speexPi)-speexLSPMargin {
		lsp[speexNBOrder-1] = float32(speexPi) - speexLSPMargin
	}
	for i := 1; i < speexNBOrder-1; i++ {
		if lsp[i] < lsp[i-1]+speexLSPMargin {
			lsp[i] = lsp[i-1] + speexLSPMargin
		}
		if lsp[i] > lsp[i+1]-speexLSPMargin {
			lsp[i] = 0.5 * (lsp[i] + lsp[i+1] - speexLSPMargin)
		}
	}
}

func speexLSPQuantNB(lsp, qlsp []float32, bits *speexBits) {
	var weight [10]float32
	copy(qlsp, lsp)
	for i := 0; i < speexNBOrder; i++ {
		tmp1 := qlsp[i]
		if i != 0 {
			tmp1 = qlsp[i] - qlsp[i-1]
		}
		tmp2 := float32(speexPi) - qlsp[i]
		if i != speexNBOrder-1 {
			tmp2 = qlsp[i+1] - qlsp[i]
		}
		if tmp2 < tmp1 {
			tmp1 = tmp2
		}
		weight[i] = speexF32(10 / (0.04 + float64(tmp1)))
	}
	for i := 0; i < speexNBOrder; i++ {
		qlsp[i] = qlsp[i] - speexF32(0.25*float64(i)+0.25)
		qlsp[i] = speexF32(256 * float64(qlsp[i]))
	}
	id := speexLSPQuant(qlsp[:speexNBOrder], speexCdbkNB[:], 64, 10)
	bits.pack(id, 6)
	for i := 0; i < speexNBOrder; i++ {
		qlsp[i] *= 2
	}
	id = speexLSPWeightQuant(qlsp[:5], weight[:5], speexCdbkNBLow1[:], 64, 5)
	bits.pack(id, 6)
	for i := 0; i < 5; i++ {
		qlsp[i] *= 2
	}
	id = speexLSPWeightQuant(qlsp[:5], weight[:5], speexCdbkNBLow2[:], 64, 5)
	bits.pack(id, 6)
	id = speexLSPWeightQuant(qlsp[5:10], weight[5:10], speexCdbkNBHigh1[:], 64, 5)
	bits.pack(id, 6)
	for i := 5; i < 10; i++ {
		qlsp[i] *= 2
	}
	id = speexLSPWeightQuant(qlsp[5:10], weight[5:10], speexCdbkNBHigh2[:], 64, 5)
	bits.pack(id, 6)
	for i := 0; i < speexNBOrder; i++ {
		qlsp[i] = speexF32(float64(qlsp[i]) * 0.00097656)
		qlsp[i] = lsp[i] - qlsp[i]
	}
}

func speexLSPQuant(x []float32, cdbk []int8, nbVec, nbDim int) int {
	bestDist := float32(1e15)
	bestID := 0
	ptr := 0
	for i := 0; i < nbVec; i++ {
		dist := float32(0)
		for j := 0; j < nbDim; j++ {
			tmp := x[j] - float32(cdbk[ptr])
			ptr++
			dist += tmp * tmp
		}
		if dist < bestDist {
			bestDist = dist
			bestID = i
		}
	}
	for j := 0; j < nbDim; j++ {
		x[j] -= float32(cdbk[bestID*nbDim+j])
	}
	return bestID
}

func speexLSPWeightQuant(x, weight []float32, cdbk []int8, nbVec, nbDim int) int {
	bestDist := float32(1e15)
	bestID := 0
	ptr := 0
	for i := 0; i < nbVec; i++ {
		dist := float32(0)
		for j := 0; j < nbDim; j++ {
			tmp := x[j] - float32(cdbk[ptr])
			ptr++
			dist += weight[j] * (tmp * tmp)
		}
		if dist < bestDist {
			bestDist = dist
			bestID = i
		}
	}
	for j := 0; j < nbDim; j++ {
		x[j] -= float32(cdbk[bestID*nbDim+j])
	}
	return bestID
}

func speexBWLPC(gamma float32, in, out []float32) {
	tmp := gamma
	for i := range in {
		out[i] = tmp * in[i]
		tmp *= gamma
	}
}

func speexFilterMem(x, num, den, y, mem []float32, n, ord int) {
	for i := 0; i < n; i++ {
		xi := x[i]
		yi := xi + mem[0]
		nyi := -yi
		for j := 0; j < ord-1; j++ {
			mem[j] = mem[j+1] + num[j]*xi + den[j]*nyi
		}
		mem[ord-1] = num[ord-1]*xi + den[ord-1]*nyi
		y[i] = yi
	}
}

func speexIIRMem(x, den, y []float32, n int, mem []float32) {
	ord := len(den)
	for i := 0; i < n; i++ {
		yi := x[i] + mem[0]
		nyi := -yi
		for j := 0; j < ord-1; j++ {
			mem[j] = mem[j+1] + den[j]*nyi
		}
		mem[ord-1] = den[ord-1] * nyi
		y[i] = yi
	}
}

func speexFIRMem(x, num, y []float32, n int, mem []float32) {
	ord := len(num)
	for i := 0; i < n; i++ {
		xi := x[i]
		yi := xi + mem[0]
		for j := 0; j < ord-1; j++ {
			mem[j] = mem[j+1] + num[j]*xi
		}
		mem[ord-1] = num[ord-1] * xi
		y[i] = yi
	}
}

func speexImpulse(ak, awk1, awk2, y []float32, n, ord int) {
	y[0] = 1
	for i := 0; i < ord; i++ {
		y[i+1] = awk1[i]
	}
	for i := ord + 1; i < n; i++ {
		y[i] = speexVerySmall
	}
	mem1 := make([]float32, ord)
	mem2 := make([]float32, ord)
	for i := 0; i < n; i++ {
		y1 := y[i] + mem1[0]
		ny1 := -y1
		y[i] = y1 + mem2[0]
		ny2 := -y[i]
		for j := 0; j < ord-1; j++ {
			mem1[j] = mem1[j+1] + awk2[j]*ny1
			mem2[j] = mem2[j+1] + ak[j]*ny2
		}
		mem1[ord-1] = awk2[ord-1] * ny1
		mem2[ord-1] = ak[ord-1] * ny2
	}
}

func speexRMS(x []float32) float32 {
	sum := float32(0)
	for _, v := range x {
		sum += v * v
	}
	return speexF32(math.Sqrt(0.1 + float64(sum)/float64(len(x))))
}

func speexInnerProd(x []float32, xOff int, y []float32, yOff int, length int) float32 {
	var s0, s1, s2, s3 float32
	n4 := length >> 2
	xi := xOff
	yi := yOff
	i := 0
	for i+2 <= n4 {
		s0 += x[xi+0] * y[yi+0]
		s1 += x[xi+1] * y[yi+1]
		s2 += x[xi+2] * y[yi+2]
		s3 += x[xi+3] * y[yi+3]
		s0 += x[xi+4] * y[yi+4]
		s1 += x[xi+5] * y[yi+5]
		s2 += x[xi+6] * y[yi+6]
		s3 += x[xi+7] * y[yi+7]
		xi += 8
		yi += 8
		i += 2
	}
	return (s0 + s2) + (s1 + s3)
}

func speexInnerProdS(a, b []float32) float32 {
	return speexInnerProd(a, 0, b, 0, len(a))
}

func speexScalQuant(in float32, bound []float32, entries int) int {
	i := 0
	for i < entries-1 && in > bound[i] {
		i++
	}
	return i
}

func speexPitchXCorr(x []float32, xOff int, y []float32, yOff int, corr []float32, length, nbPitch int) {
	for i := 0; i < nbPitch; i++ {
		corr[nbPitch-1-i] = speexInnerProd(x, xOff, y, yOff+i, length)
	}
}

func speexOpenLoopPitch(sw []float32, swOff, start, end, length int, pitch []int, gain []float32, n int) {
	bestScore := make([]float32, n)
	bestEner := make([]float32, n)
	for i := 0; i < n; i++ {
		bestScore[i] = -1
		bestEner[i] = 0
		pitch[i] = start
	}
	energy := make([]float32, end-start+2)
	corr := make([]float32, end-start+1)
	energy[0] = speexInnerProd(sw, swOff-start, sw, swOff-start, length)
	e0 := speexInnerProd(sw, swOff, sw, swOff, length)
	for i := start; i < end; i++ {
		energy[i-start+1] = energy[i-start] + sw[swOff-i-1]*sw[swOff-i-1] - sw[swOff-i+length-1]*sw[swOff-i+length-1]
		if energy[i-start+1] < 0 {
			energy[i-start+1] = 0
		}
	}
	speexPitchXCorr(sw, swOff, sw, swOff-end, corr, length, end-start+1)
	for i := start; i <= end; i++ {
		tmp := corr[i-start] * corr[i-start]
		if tmp*bestEner[n-1] > bestScore[n-1]*(1+energy[i-start]) {
			bestScore[n-1] = tmp
			bestEner[n-1] = energy[i-start] + 1
			pitch[n-1] = i
			for j := 0; j < n-1; j++ {
				if tmp*bestEner[j] > bestScore[j]*(1+energy[i-start]) {
					for k := n - 1; k > j; k-- {
						bestScore[k] = bestScore[k-1]
						bestEner[k] = bestEner[k-1]
						pitch[k] = pitch[k-1]
					}
					bestScore[j] = tmp
					bestEner[j] = energy[i-start] + 1
					pitch[j] = i
					break
				}
			}
		}
	}
	if gain != nil {
		for j := 0; j < n; j++ {
			i := pitch[j]
			g := corr[i-start] / (10 + float32(math.Sqrt(float64(e0)))*float32(math.Sqrt(float64(energy[i-start]))))
			if g < 0 {
				g = 0
			}
			gain[j] = g
		}
	}
}

func speexPitchError(c, g []float32, pitchControl float32) float32 {
	sum := float32(0)
	sum += g[0] * pitchControl * c[0]
	sum += g[1] * pitchControl * c[1]
	sum += g[2] * pitchControl * c[2]
	sum -= g[0] * g[1] * c[3]
	sum -= g[2] * g[1] * c[4]
	sum -= g[2] * g[0] * c[5]
	sum -= g[0] * g[0] * c[6]
	sum -= g[1] * g[1] * c[7]
	sum -= g[2] * g[2] * c[8]
	return sum
}

func speexPitchGainVQ(c []float32, maxGain float32) int {
	best := 0
	bestSum := float32(-1e15)
	var g [3]float32
	for i := 0; i < 128; i++ {
		ptr := i * 4
		g[0] = float32(speexGainCdbkNB[ptr]) + 32
		g[1] = float32(speexGainCdbkNB[ptr+1]) + 32
		g[2] = float32(speexGainCdbkNB[ptr+2]) + 32
		gainSum := float32(speexGainCdbkNB[ptr+3])
		sum := speexPitchError(c, g[:], 64)
		if sum > bestSum && gainSum <= maxGain {
			bestSum = sum
			best = i
		}
	}
	return best
}

func speexPitchGainSearch(target, ak, awk1, awk2 []float32, exc []float32, pitch, p, nsf int, exc2 []float32, exc2Off int, r []float32, newTarget []float32, plcTuning int, cumulGain float32) (err float32, cdbkIndex int) {
	x0 := make([]float32, nsf)
	x1 := make([]float32, nsf)
	x2 := make([]float32, nsf)
	e := make([]float32, nsf)
	copy(newTarget, target[:nsf])
	maxGain := float32(128)
	if cumulGain > 262144 {
		maxGain = 31
	}
	pp := pitch - 1
	bound := nsf
	if nsf-pp > 0 {
		bound = pp
	}
	j := 0
	for ; j < bound; j++ {
		e[j] = exc2[exc2Off+j-pp]
	}
	bound = nsf
	if nsf-pp-pitch > 0 {
		bound = pp + pitch
	}
	for ; j < bound; j++ {
		e[j] = exc2[exc2Off+j-pp-pitch]
	}
	for ; j < nsf; j++ {
		e[j] = 0
	}
	mm := make([]float32, p)
	speexIIRMem(e, ak, e, nsf, mm)
	for i := 0; i < p; i++ {
		mm[i] = 0
	}
	speexFilterMem(e, awk1, awk2, e, mm, nsf, p)
	copy(x2, e[:nsf])
	xs := [3][]float32{x0, x1, x2}
	for i := 1; i >= 0; i-- {
		e0 := exc2[exc2Off-pitch-1+i]
		xs[i][0] = r[0] * e0
		for j := 0; j < nsf-1; j++ {
			xs[i][j+1] = xs[i+1][j] + r[j+1]*e0
		}
	}
	var corr [3]float32
	var a [3][3]float32
	for i := 0; i < 3; i++ {
		corr[i] = speexInnerProdS(xs[i][:nsf], newTarget[:nsf])
	}
	for i := 0; i < 3; i++ {
		for j := 0; j <= i; j++ {
			v := speexInnerProdS(xs[i][:nsf], xs[j][:nsf])
			a[i][j] = v
			a[j][i] = v
		}
	}
	c := [9]float32{
		corr[2], corr[1], corr[0],
		a[1][2], a[0][1], a[0][2],
		a[2][2], a[1][1], a[0][0],
	}
	if plcTuning < 2 {
		plcTuning = 2
	}
	if plcTuning > 30 {
		plcTuning = 30
	}
	scale := speexF32(0.5 * (1 + 0.02*float64(plcTuning)))
	c[6] *= scale
	c[7] *= scale
	c[8] *= scale
	best := speexPitchGainVQ(c[:], maxGain)
	var gain [3]float32
	gain[0] = speexF32(0.015625*float64(speexGainCdbkNB[best*4]) + 0.5)
	gain[1] = speexF32(0.015625*float64(speexGainCdbkNB[best*4+1]) + 0.5)
	gain[2] = speexF32(0.015625*float64(speexGainCdbkNB[best*4+2]) + 0.5)
	for i := 0; i < nsf; i++ {
		exc[i] = 0
	}
	for i := 0; i < 3; i++ {
		pp := pitch + 1 - i
		tmp1 := nsf
		if tmp1 > pp {
			tmp1 = pp
		}
		for j := 0; j < tmp1; j++ {
			exc[j] += gain[2-i] * exc2[exc2Off+j-pp]
		}
		tmp3 := nsf
		if tmp3 > pp+pitch {
			tmp3 = pp + pitch
		}
		for j := tmp1; j < tmp3; j++ {
			exc[j] += gain[2-i] * exc2[exc2Off+j-pp-pitch]
		}
	}
	for i := 0; i < nsf; i++ {
		tmp := gain[0]*x2[i] + gain[1]*x1[i] + gain[2]*x0[i]
		newTarget[i] -= tmp
	}
	return speexInnerProdS(newTarget[:nsf], newTarget[:nsf]), best
}

func speexPitchSearch3Tap(target, swBuf []float32, swOff int, ak, awk1, awk2 []float32, exc []float32, start, end int, pitchCoef float32, bits *speexBits, exc2 []float32, exc2Off int, r []float32, complexity int, cumulGain *float32) int {
	n := complexity
	if n > 10 {
		n = 10
	}
	if n < 1 {
		n = 1
	}
	nsf := speexNBSub
	if end < start {
		bits.pack(0, speexPitchBits)
		bits.pack(0, speexGainBits)
		for i := 0; i < nsf; i++ {
			exc[i] = 0
		}
		return start
	}
	if n > end-start+1 {
		n = end - start + 1
	}
	nbest := make([]int, n)
	if end != start {
		speexOpenLoopPitch(swBuf, swOff, start, end, nsf, nbest, nil, n)
	} else {
		nbest[0] = start
	}
	bestExc := make([]float32, nsf)
	newTarget := make([]float32, nsf)
	bestTarget := make([]float32, nsf)
	bestErr := float32(-1)
	bestPitch := 0
	bestGain := 0
	pitch := 0
	for i := 0; i < n; i++ {
		pitch = nbest[i]
		for j := 0; j < nsf; j++ {
			exc[j] = 0
		}
		err, idx := speexPitchGainSearch(target, ak, awk1, awk2, exc, pitch, speexNBOrder, nsf, exc2, exc2Off, r, newTarget, 2, *cumulGain)
		if err < bestErr || bestErr < 0 {
			copy(bestExc, exc[:nsf])
			copy(bestTarget, newTarget)
			bestErr = err
			bestPitch = pitch
			bestGain = idx
		}
	}
	bits.pack(bestPitch-start, speexPitchBits)
	bits.pack(bestGain, speexGainBits)
	cg := *cumulGain
	if cg < 1024 {
		cg = 1024
	}
	*cumulGain = speexF32(0.03125 * float64(cg) * float64(speexGainCdbkNB[4*bestGain+3]))
	copy(exc[:nsf], bestExc)
	copy(target[:nsf], bestTarget)
	return pitch
}

func speexSynPercepZero(xx, ak, awk1, awk2, y []float32, n, ord int) {
	mem := make([]float32, ord)
	speexIIRMem(xx, ak, y, n, mem)
	for i := 0; i < ord; i++ {
		mem[i] = 0
	}
	speexFilterMem(y, awk1, awk2, y, mem, n, ord)
}

func speexWeightedCB(shapeCB []int8, r, resp, energy []float32, cbSize, subSize int) {
	shape := make([]float32, subSize)
	for i := 0; i < cbSize; i++ {
		res := resp[i*subSize:]
		for k := 0; k < subSize; k++ {
			shape[k] = speexF32(0.03125 * float64(shapeCB[i*subSize+k]))
		}
		energy[i] = 0
		for j := 0; j < subSize; j++ {
			resj := float32(0)
			for k := 0; k <= j; k++ {
				resj += shape[k] * r[j-k]
			}
			energy[i] += resj * resj
			res[j] = resj
		}
	}
}

func speexVQNBest(in, codebook, energy []float32, length, entries, n int, nbest []int, bestDist []float32) {
	used := 0
	ptr := 0
	for i := 0; i < entries; i++ {
		dist := float32(0.5) * energy[i]
		for j := 0; j < length; j++ {
			dist -= in[j] * codebook[ptr]
			ptr++
		}
		if i < n || dist < bestDist[n-1] {
			k := n - 1
			for k >= 1 && (k > used || dist < bestDist[k-1]) {
				bestDist[k] = bestDist[k-1]
				nbest[k] = nbest[k-1]
				k--
			}
			bestDist[k] = dist
			nbest[k] = i
			used++
		}
	}
}

func speexTargetUpdate(t []float32, g float32, r []float32, length int) {
	for n := 0; n < length; n++ {
		t[n] -= g * r[n]
	}
}

func speexSplitCB(target, ak, awk1, awk2 []float32, p, nsf int, exc, r []float32, bits *speexBits, complexity, updateTarget int) {
	n := complexity
	if n > 10 {
		n = 10
	}
	n = (2 * n) / 3
	if n < 1 {
		n = 1
	}
	subSize := speexSubvect
	nbSub := speexNBSubvect
	cbSize := speexShapeCBSize
	shapeCB := speexExc564[:]
	resp := make([]float32, cbSize*subSize)
	energy := make([]float32, cbSize)
	speexWeightedCB(shapeCB, r, resp, energy, cbSize, subSize)
	if n == 1 {
		speexSplitCBN1(target, ak, awk1, awk2, exc, r, bits, updateTarget, resp, energy, shapeCB, subSize, nbSub, cbSize, nsf, p)
		return
	}
	t := make([]float32, nsf)
	copy(t, target[:nsf])
	ot := make([][]float32, n)
	nt := make([][]float32, n)
	tmp := make([]float32, 2*n*nsf)
	for i := 0; i < n; i++ {
		ot[i] = tmp[2*i*nsf : (2*i+1)*nsf]
		nt[i] = tmp[(2*i+1)*nsf : (2*i+2)*nsf]
		copy(ot[i], t)
	}
	oind := make([][]int, n)
	nind := make([][]int, n)
	itmp := make([]int, 2*n*nbSub)
	for i := 0; i < n; i++ {
		nind[i] = itmp[2*i*nbSub : (2*i+1)*nbSub]
		oind[i] = itmp[(2*i+1)*nbSub : (2*i+2)*nbSub]
	}
	bestIndex := make([]int, n)
	bestDist := make([]float32, n)
	bestNInd := make([]int, n)
	bestNTarget := make([]int, n)
	ndist := make([]float32, n)
	odist := make([]float32, n)
	for i := 0; i < nbSub; i++ {
		for j := 0; j < n; j++ {
			ndist[j] = 1e15
			bestNInd[j] = 0
			bestNTarget[j] = 0
		}
		for j := 0; j < n; j++ {
			x := ot[j][subSize*i:]
			tener := float32(0)
			for m := 0; m < subSize; m++ {
				tener += x[m] * x[m]
			}
			tener = speexF32(float64(tener) * 0.5)
			speexVQNBest(x, resp, energy, subSize, cbSize, n, bestIndex, bestDist)
			for k := 0; k < n; k++ {
				err := odist[j] + bestDist[k] + tener
				if err < ndist[n-1] {
					for m := 0; m < n; m++ {
						if err < ndist[m] {
							for nn := n - 1; nn > m; nn-- {
								ndist[nn] = ndist[nn-1]
								bestNInd[nn] = bestNInd[nn-1]
								bestNTarget[nn] = bestNTarget[nn-1]
							}
							ndist[m] = err
							bestNInd[m] = bestIndex[k]
							bestNTarget[m] = j
							break
						}
					}
				}
			}
			if i == 0 {
				break
			}
		}
		for j := 0; j < n; j++ {
			for m := (i + 1) * subSize; m < nsf; m++ {
				nt[j][m] = ot[bestNTarget[j]][m]
			}
			for m := 0; m < subSize; m++ {
				rind := bestNInd[j]
				sign := float32(1)
				if rind >= cbSize {
					sign = -1
					rind -= cbSize
				}
				q := subSize - m
				g := sign * speexF32(0.03125*float64(shapeCB[rind*subSize+m]))
				remain := nsf - subSize*(i+1)
				if remain > 0 {
					speexTargetUpdate(nt[j][subSize*(i+1):], g, r[q:], remain)
				}
			}
			copy(nind[j], oind[bestNTarget[j]])
			nind[j][i] = bestNInd[j]
		}
		ot, nt = nt, ot
		for j := 0; j < n; j++ {
			copy(oind[j], nind[j])
			odist[j] = ndist[j]
		}
	}
	ind := make([]int, nbSub)
	for i := 0; i < nbSub; i++ {
		ind[i] = nind[0][i]
		bits.pack(ind[i], speexShapeBits)
	}
	e := make([]float32, nsf)
	for i := 0; i < nbSub; i++ {
		rind := ind[i]
		sign := float32(1)
		if rind >= cbSize {
			sign = -1
			rind -= cbSize
		}
		for j := 0; j < subSize; j++ {
			e[subSize*i+j] = sign * speexF32(0.03125*float64(shapeCB[rind*subSize+j]))
		}
	}
	for j := 0; j < nsf; j++ {
		exc[j] += e[j]
	}
	if updateTarget != 0 {
		r2 := make([]float32, nsf)
		copy(r2, e)
		speexSynPercepZero(r2, ak, awk1, awk2, r2, nsf, p)
		for j := 0; j < nsf; j++ {
			target[j] -= r2[j]
		}
	}
}

func speexSplitCBN1(target, ak, awk1, awk2, exc, r []float32, bits *speexBits, updateTarget int, resp, energy []float32, shapeCB []int8, subSize, nbSub, cbSize, nsf, p int) {
	t := make([]float32, nsf)
	copy(t, target[:nsf])
	e := make([]float32, nsf)
	for i := 0; i < nbSub; i++ {
		x := t[subSize*i:]
		bestIndex := make([]int, 1)
		bestDist := make([]float32, 1)
		speexVQNBest(x, resp, energy, subSize, cbSize, 1, bestIndex, bestDist)
		bits.pack(bestIndex[0], speexShapeBits)
		rind := bestIndex[0]
		sign := float32(1)
		if rind >= cbSize {
			sign = -1
			rind -= cbSize
		}
		res := resp[rind*subSize:]
		if sign > 0 {
			for m := 0; m < subSize; m++ {
				t[subSize*i+m] -= res[m]
			}
		} else {
			for m := 0; m < subSize; m++ {
				t[subSize*i+m] += res[m]
			}
		}
		for j := 0; j < subSize; j++ {
			e[subSize*i+j] = sign * speexF32(0.03125*float64(shapeCB[rind*subSize+j]))
		}
		for m := 0; m < subSize; m++ {
			g := sign * speexF32(0.03125*float64(shapeCB[rind*subSize+m]))
			q := subSize - m
			remain := nsf - subSize*(i+1)
			if remain > 0 {
				speexTargetUpdate(t[subSize*(i+1):], g, r[q:], remain)
			}
		}
	}
	for j := 0; j < nsf; j++ {
		exc[j] += e[j]
	}
	if updateTarget != 0 {
		r2 := make([]float32, nsf)
		copy(r2, e)
		speexSynPercepZero(r2, ak, awk1, awk2, r2, nsf, p)
		for j := 0; j < nsf; j++ {
			target[j] -= r2[j]
		}
	}
}
