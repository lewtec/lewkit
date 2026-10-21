package avcodec

import (
	"fmt"
	"math/bits"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// gsmEnc is the libgsm encoder state (FAST off, standard 13 kbit/s).
type gsmEnc struct {
	z1    int16
	lz2   int32
	mp    int16
	u     [8]int16
	larpp [2][8]int16
	j     int
	nrp   int16 // initial 40
	dp0   [280]int16
	e     [50]int16
}

type gsmCodes struct {
	larc              [8]int16
	nc, bc, mc, xmaxc [4]int16
	xmc               [52]int16
}

func (g *gsmCodec) encodeFrame(samples []int16) ([33]byte, error) {
	c, err := g.gsmAnalyze(samples)
	if err != nil {
		return [33]byte{}, err
	}
	return gsmEncPack(c.larc, c.nc, c.bc, c.mc, c.xmaxc, c.xmc), nil
}

func (g *gsmCodec) gsmAnalyze(samples []int16) (gsmCodes, error) {
	var c gsmCodes
	if len(samples) != 160 {
		return c, fmt.Errorf("%w: gsm frame", avutil.ErrInvalidData)
	}
	e := &g.enc
	var so [160]int16
	gsmEncPreprocess(e, samples, so[:])
	gsmEncLPC(so[:], c.larc[:])
	gsmEncShortTerm(e, c.larc[:], so[:])
	base := 120
	for k := 0; k < 4; k++ {
		var dpp [40]int16
		gsmEncLTP(so[k*40:k*40+40], e.dp0[:], base, e.e[5:45], dpp[:], &c.nc[k], &c.bc[k])
		gsmEncRPE(e.e[:], &c.xmaxc[k], &c.mc[k], c.xmc[k*13:k*13+13])
		for i := 0; i < 40; i++ {
			e.dp0[base+i] = gsmEncAdd(e.e[5+i], dpp[i])
		}
		base += 40
	}
	copy(e.dp0[:120], e.dp0[160:280])
	return c, nil
}

func gsmEncAdd(a, b int16) int16 { return int16(clip16(int32(a) + int32(b))) }

func gsmEncSub(a, b int16) int16 { return int16(clip16(int32(a) - int32(b))) }

func gsmEncAbs(a int16) int16 {
	if a < 0 {
		if a == -32768 {
			return 32767
		}
		return -a
	}
	return a
}

func gsmEncMult(a, b int16) int16 {
	if a == -32768 && b == -32768 {
		return 32767
	}
	return int16((int32(a) * int32(b)) >> 15)
}

func gsmEncMultR(a, b int16) int16 {
	if a == -32768 && b == -32768 {
		return 32767
	}
	return int16((int32(a)*int32(b) + 16384) >> 15)
}

func gsmEncLAdd(a, b int32) int32 {
	s := int64(a) + int64(b)
	if s > 2147483647 {
		return 2147483647
	}
	if s < -2147483648 {
		return -2147483648
	}
	return int32(s)
}

func gsmEncNorm(a int32) int16 {
	if a < 0 {
		if a <= -1073741824 {
			return 0
		}
		a = ^a
	}
	if a == 0 {
		return 31
	}
	return int16(bits.LeadingZeros32(uint32(a)) - 1)
}

func gsmEncDiv(num, denum int16) int16 {
	if num == 0 {
		return 0
	}
	lNum := int32(num)
	lDenum := int32(denum)
	var div int16
	for k := 0; k < 15; k++ {
		div <<= 1
		lNum <<= 1
		if lNum >= lDenum {
			lNum -= lDenum
			div++
		}
	}
	return div
}

func gsmEncAsr(a int16, n int) int16 {
	if n >= 16 {
		if a < 0 {
			return -1
		}
		return 0
	}
	if n <= -16 {
		return 0
	}
	if n < 0 {
		return gsmEncAsl(a, -n)
	}
	return a >> n
}

func gsmEncAsl(a int16, n int) int16 {
	if n >= 16 {
		return 0
	}
	if n <= -16 {
		if a < 0 {
			return -1
		}
		return 0
	}
	if n < 0 {
		return gsmEncAsr(a, -n)
	}
	return int16(int32(a) << n)
}

func gsmEncPreprocess(e *gsmEnc, s, so []int16) {
	z1 := e.z1
	lz2 := e.lz2
	mp := e.mp
	for k := 0; k < 160; k++ {
		down := int16(int32(s[k]) >> 3 << 2)
		s1 := int16(int32(down) - int32(z1))
		z1 = down

		ls2 := int32(s1) << 15
		msp := int16(lz2 >> 15)
		lsp := int16(lz2 - (int32(msp) << 15))
		ls2 += int32(gsmEncMultR(lsp, 32735))
		ltemp := int32(msp) * 32735
		lz2 = gsmEncLAdd(ltemp, ls2)
		ltemp = gsmEncLAdd(lz2, 16384)

		pre := gsmEncMultR(mp, -28180)
		mp = int16(ltemp >> 15)
		so[k] = gsmEncAdd(mp, pre)
	}
	e.z1 = z1
	e.lz2 = lz2
	e.mp = mp
}

func gsmEncLPC(s, larc []int16) {
	var lacf [9]int32
	gsmEncAutocorr(s, lacf[:])
	gsmEncReflection(lacf[:], larc)
	gsmEncToLAR(larc)
	gsmEncQuantLAR(larc)
}

func gsmEncAutocorr(s []int16, lacf []int32) {
	var smax int16
	for k := 0; k < 160; k++ {
		t := gsmEncAbs(s[k])
		if t > smax {
			smax = t
		}
	}
	var scalauto int
	if smax != 0 {
		scalauto = 4 - int(gsmEncNorm(int32(smax)<<16))
	}
	if scalauto > 0 {
		fac := int16(16384 >> (scalauto - 1))
		for k := 0; k < 160; k++ {
			s[k] = gsmEncMultR(s[k], fac)
		}
	}
	for i := 0; i < 160; i++ {
		sl := int32(s[i])
		nk := i
		if nk > 8 {
			nk = 8
		}
		for k := 0; k <= nk; k++ {
			lacf[k] += sl * int32(s[i-k])
		}
	}
	for k := 0; k < 9; k++ {
		lacf[k] <<= 1
	}
	if scalauto > 0 {
		for k := 0; k < 160; k++ {
			s[k] = int16(int32(s[k]) << scalauto)
		}
	}
}

func gsmEncReflection(lacf []int32, r []int16) {
	if lacf[0] == 0 {
		for i := 0; i < 8; i++ {
			r[i] = 0
		}
		return
	}
	tempn := gsmEncNorm(lacf[0])
	var acf, p, kk [9]int16
	for i := 0; i <= 8; i++ {
		acf[i] = int16((lacf[i] << tempn) >> 16)
	}
	for i := 1; i <= 7; i++ {
		kk[i] = acf[i]
	}
	for i := 0; i <= 8; i++ {
		p[i] = acf[i]
	}
	for n := 1; n <= 8; n++ {
		temp := gsmEncAbs(p[1])
		if p[0] < temp {
			for i := n; i <= 8; i++ {
				r[i-1] = 0
			}
			return
		}
		r[n-1] = gsmEncDiv(temp, p[0])
		if p[1] > 0 {
			r[n-1] = -r[n-1]
		}
		if n == 8 {
			return
		}
		t := gsmEncMultR(p[1], r[n-1])
		p[0] = gsmEncAdd(p[0], t)
		for m := 1; m <= 8-n; m++ {
			t = gsmEncMultR(kk[m], r[n-1])
			pNext := p[m+1]
			p[m] = gsmEncAdd(pNext, t)
			t = gsmEncMultR(pNext, r[n-1])
			kk[m] = gsmEncAdd(kk[m], t)
		}
	}
}

func gsmEncToLAR(r []int16) {
	for i := 0; i < 8; i++ {
		temp := gsmEncAbs(r[i])
		switch {
		case temp < 22118:
			temp >>= 1
		case temp < 31130:
			temp -= 11059
		default:
			temp -= 26112
			temp <<= 2
		}
		if r[i] < 0 {
			r[i] = -temp
		} else {
			r[i] = temp
		}
	}
}

func gsmEncQuantLAR(lar []int16) {
	type step struct{ a, b, mac, mic int16 }
	steps := [...]step{
		{20480, 0, 31, -32},
		{20480, 0, 31, -32},
		{20480, 2048, 15, -16},
		{20480, -2560, 15, -16},
		{13964, 94, 7, -8},
		{15360, -1792, 7, -8},
		{8534, -341, 3, -4},
		{9036, -1144, 3, -4},
	}
	for i, st := range steps {
		temp := gsmEncMult(st.a, lar[i])
		temp = gsmEncAdd(temp, st.b)
		temp = gsmEncAdd(temp, 256)
		temp >>= 9
		switch {
		case temp > st.mac:
			lar[i] = st.mac - st.mic
		case temp < st.mic:
			lar[i] = 0
		default:
			lar[i] = temp - st.mic
		}
	}
}

func gsmEncShortTerm(e *gsmEnc, larc, s []int16) {
	cur := e.j
	e.j ^= 1
	prev := e.j
	gsmEncDecodeLAR(larc, e.larpp[cur][:])

	var larp [8]int16
	gsmEncCoeff012(e.larpp[prev][:], e.larpp[cur][:], larp[:])
	gsmEncLARpToRP(larp[:])
	gsmEncAnalysis(e, larp[:], s[0:13])

	gsmEncCoeff1326(e.larpp[prev][:], e.larpp[cur][:], larp[:])
	gsmEncLARpToRP(larp[:])
	gsmEncAnalysis(e, larp[:], s[13:27])

	gsmEncCoeff2739(e.larpp[prev][:], e.larpp[cur][:], larp[:])
	gsmEncLARpToRP(larp[:])
	gsmEncAnalysis(e, larp[:], s[27:40])

	gsmEncCoeff40159(e.larpp[cur][:], larp[:])
	gsmEncLARpToRP(larp[:])
	gsmEncAnalysis(e, larp[:], s[40:160])
}

func gsmEncDecodeLAR(larc, larpp []int16) {
	type step struct{ b2, mic, inva int16 }
	steps := [...]step{
		{0, -32, 13107},
		{0, -32, 13107},
		{4096, -16, 13107},
		{-5120, -16, 13107},
		{188, -8, 19223},
		{-3584, -8, 17476},
		{-682, -4, 31454},
		{-2288, -4, 29708},
	}
	for i, st := range steps {
		temp := int16(int32(gsmEncAdd(larc[i], st.mic)) << 10)
		temp = gsmEncSub(temp, st.b2)
		temp = gsmEncMultR(st.inva, temp)
		larpp[i] = gsmEncAdd(temp, temp)
	}
}

func gsmEncCoeff012(prev, cur, larp []int16) {
	for i := 0; i < 8; i++ {
		larp[i] = gsmEncAdd(prev[i]>>2, cur[i]>>2)
		larp[i] = gsmEncAdd(larp[i], prev[i]>>1)
	}
}

func gsmEncCoeff1326(prev, cur, larp []int16) {
	for i := 0; i < 8; i++ {
		larp[i] = gsmEncAdd(prev[i]>>1, cur[i]>>1)
	}
}

func gsmEncCoeff2739(prev, cur, larp []int16) {
	for i := 0; i < 8; i++ {
		larp[i] = gsmEncAdd(prev[i]>>2, cur[i]>>2)
		larp[i] = gsmEncAdd(larp[i], cur[i]>>1)
	}
}

func gsmEncCoeff40159(cur, larp []int16) {
	copy(larp, cur[:8])
}

func gsmEncRPFromAbs(temp int16) int16 {
	if temp < 11059 {
		return temp << 1
	}
	if temp < 20070 {
		return temp + 11059
	}
	return gsmEncAdd(temp>>2, 26112)
}

func gsmEncLARpToRP(larp []int16) {
	for i := 0; i < 8; i++ {
		v := larp[i]
		if v < 0 {
			temp := v
			if temp == -32768 {
				temp = 32767
			} else {
				temp = -temp
			}
			larp[i] = -gsmEncRPFromAbs(temp)
		} else {
			larp[i] = gsmEncRPFromAbs(v)
		}
	}
}

func gsmEncAnalysis(e *gsmEnc, rp, s []int16) {
	for n := range s {
		di := s[n]
		sav := di
		for i := 0; i < 8; i++ {
			ui := e.u[i]
			rpi := rp[i]
			e.u[i] = sav
			zzz := gsmEncMultR(rpi, di)
			sav = gsmEncAdd(ui, zzz)
			zzz = gsmEncMultR(rpi, ui)
			di = gsmEncAdd(di, zzz)
		}
		s[n] = di
	}
}

func gsmEncLTP(d, dp0 []int16, base int, e, dpp []int16, nc, bc *int16) {
	var dmax int16
	for k := 0; k < 40; k++ {
		t := gsmEncAbs(d[k])
		if t > dmax {
			dmax = t
		}
	}
	var temp int16
	if dmax != 0 {
		temp = gsmEncNorm(int32(dmax) << 16)
	}
	var scal int16
	if temp > 6 {
		scal = 0
	} else {
		scal = 6 - temp
	}
	var wt [40]int16
	for k := 0; k < 40; k++ {
		wt[k] = d[k] >> scal
	}
	var lMax int32
	n := int16(40)
	for lambda := 40; lambda <= 120; lambda++ {
		var lResult int32
		for k := 0; k < 40; k++ {
			lResult += int32(wt[k]) * int32(dp0[base+k-lambda])
		}
		if lResult > lMax {
			n = int16(lambda)
			lMax = lResult
		}
	}
	*nc = n
	lMax <<= 1
	lMax >>= 6 - scal

	var lPower int32
	for k := 0; k < 40; k++ {
		lt := int32(dp0[base+k-int(n)] >> 3)
		lPower += lt * lt
	}
	lPower <<= 1

	var b int16
	switch {
	case lMax <= 0:
		b = 0
	case lMax >= lPower:
		b = 3
	default:
		norm := gsmEncNorm(lPower)
		r := int16((lMax << norm) >> 16)
		sv := int16((lPower << norm) >> 16)
		dlb := [...]int16{6554, 16384, 26214}
		b = 3
		for i := int16(0); i <= 2; i++ {
			if r <= gsmEncMult(sv, dlb[i]) {
				b = i
				break
			}
		}
	}
	*bc = b
	gsmEncLTFilter(b, n, dp0, base, d, dpp, e)
}

func gsmEncLTFilter(bc, nc int16, dp0 []int16, base int, d, dpp, e []int16) {
	bp := [...]int16{3277, 11469, 21299, 32767}[bc]
	lag := int(nc)
	for k := 0; k < 40; k++ {
		dpp[k] = gsmEncMultR(bp, dp0[base+k-lag])
		e[k] = gsmEncSub(d[k], dpp[k])
	}
}

func gsmEncRPE(e []int16, xmaxc, mc *int16, xmc []int16) {
	var x [40]int16
	gsmEncWeight(e, x[:])
	var xm [13]int16
	gsmEncGridSel(x[:], xm[:], mc)
	mant, exp := gsmEncAPCMQuant(xm[:], xmc, xmaxc)
	var xmp [13]int16
	gsmEncAPCMInv(xmc, mant, exp, xmp[:])
	gsmEncGridPos(*mc, xmp[:], e[5:45])
}

func gsmEncWeight(e, x []int16) {
	for k := 0; k < 40; k++ {
		l := int32(4096)
		l += int32(e[k+0]) * -134
		l += int32(e[k+1]) * -374
		l += int32(e[k+3]) * 2054
		l += int32(e[k+4]) * 5741
		l += int32(e[k+5]) * 8192
		l += int32(e[k+6]) * 5741
		l += int32(e[k+7]) * 2054
		l += int32(e[k+9]) * -374
		l += int32(e[k+10]) * -134
		x[k] = int16(clip16(l >> 13))
	}
}

func gsmEncGridSel(x, xm []int16, mc *int16) {
	step := func(m, i int) int32 {
		lt := int32(x[m+3*i] >> 2)
		return lt * lt
	}
	var common int32
	for i := 1; i <= 12; i++ {
		common += step(0, i)
	}
	lResult := common + step(0, 0)
	lResult <<= 1
	em := lResult
	sel := int16(0)

	lResult = 0
	for i := 0; i <= 12; i++ {
		lResult += step(1, i)
	}
	lResult <<= 1
	if lResult > em {
		sel = 1
		em = lResult
	}

	lResult = 0
	for i := 0; i <= 12; i++ {
		lResult += step(2, i)
	}
	lResult <<= 1
	if lResult > em {
		sel = 2
		em = lResult
	}

	lResult = common + step(3, 12)
	lResult <<= 1
	if lResult > em {
		sel = 3
	}
	for i := 0; i <= 12; i++ {
		xm[i] = x[int(sel)+3*i]
	}
	*mc = sel
}

func gsmEncXmaxExpMant(xmaxc int16) (exp, mant int16) {
	if xmaxc > 15 {
		exp = (xmaxc >> 3) - 1
	}
	mant = xmaxc - (exp << 3)
	if mant == 0 {
		return -4, 7
	}
	for mant <= 7 {
		mant = (mant << 1) | 1
		exp--
	}
	mant -= 8
	return exp, mant
}

func gsmEncAPCMQuant(xm, xmc []int16, xmaxc *int16) (mant, exp int16) {
	var xmax int16
	for i := 0; i <= 12; i++ {
		t := gsmEncAbs(xm[i])
		if t > xmax {
			xmax = t
		}
	}
	exp = 0
	temp := xmax >> 9
	itest := 0
	for i := 0; i <= 5; i++ {
		if temp <= 0 {
			itest = 1
		}
		temp >>= 1
		if itest == 0 {
			exp++
		}
	}
	*xmaxc = gsmEncAdd(xmax>>int(exp+5), exp<<3)
	exp, mant = gsmEncXmaxExpMant(*xmaxc)

	nrfac := [...]int16{29128, 26215, 23832, 21846, 20165, 18725, 17476, 16384}
	temp1 := 6 - exp
	temp2 := nrfac[mant]
	for i := 0; i <= 12; i++ {
		t := int16(int32(xm[i]) << temp1)
		t = gsmEncMult(t, temp2)
		t >>= 12
		xmc[i] = t + 4
	}
	return mant, exp
}

func gsmEncAPCMInv(xmc []int16, mant, exp int16, xmp []int16) {
	fac := [...]int16{18431, 20479, 22527, 24575, 26623, 28671, 30719, 32767}
	temp1 := fac[mant]
	temp2 := gsmEncSub(6, exp)
	temp3 := gsmEncAsl(1, int(gsmEncSub(temp2, 1)))
	sh := int(temp2)
	for i := 0; i < 13; i++ {
		temp := int16((int32(xmc[i]) << 1) - 7)
		temp = int16(int32(temp) << 12)
		temp = gsmEncMultR(temp1, temp)
		temp = gsmEncAdd(temp, temp3)
		xmp[i] = gsmEncAsr(temp, sh)
	}
}

func gsmEncGridPos(mc int16, xmp, ep []int16) {
	for k := 0; k < 40; k++ {
		ep[k] = 0
	}
	m := int(mc)
	for i := 0; i <= 12; i++ {
		ep[m+3*i] = xmp[i]
	}
}

func gsmEncPack(larc [8]int16, nc, bc, mc, xmaxc [4]int16, xmc [52]int16) [33]byte {
	u := func(v int16) uint { return uint(uint16(v)) }
	var c [33]byte
	c[0] = byte(((0xD & 0xF) << 4) | ((u(larc[0]) >> 2) & 0xF))
	c[1] = byte(((u(larc[0]) & 0x3) << 6) | (u(larc[1]) & 0x3F))
	c[2] = byte(((u(larc[2]) & 0x1F) << 3) | ((u(larc[3]) >> 2) & 0x7))
	c[3] = byte(((u(larc[3]) & 0x3) << 6) | ((u(larc[4]) & 0xF) << 2) | ((u(larc[5]) >> 2) & 0x3))
	c[4] = byte(((u(larc[5]) & 0x3) << 6) | ((u(larc[6]) & 0x7) << 3) | (u(larc[7]) & 0x7))

	putSub := func(off int, sub int, x0 int) {
		c[off] = byte(((u(nc[sub]) & 0x7F) << 1) | ((u(bc[sub]) >> 1) & 0x1))
		c[off+1] = byte(((u(bc[sub]) & 0x1) << 7) | ((u(mc[sub]) & 0x3) << 5) | ((u(xmaxc[sub]) >> 1) & 0x1F))
		c[off+2] = byte(((u(xmaxc[sub]) & 0x1) << 7) | ((u(xmc[x0]) & 0x7) << 4) | ((u(xmc[x0+1]) & 0x7) << 1) | ((u(xmc[x0+2]) >> 2) & 0x1))
		c[off+3] = byte(((u(xmc[x0+2]) & 0x3) << 6) | ((u(xmc[x0+3]) & 0x7) << 3) | (u(xmc[x0+4]) & 0x7))
		c[off+4] = byte(((u(xmc[x0+5]) & 0x7) << 5) | ((u(xmc[x0+6]) & 0x7) << 2) | ((u(xmc[x0+7]) >> 1) & 0x3))
		c[off+5] = byte(((u(xmc[x0+7]) & 0x1) << 7) | ((u(xmc[x0+8]) & 0x7) << 4) | ((u(xmc[x0+9]) & 0x7) << 1) | ((u(xmc[x0+10]) >> 2) & 0x1))
		c[off+6] = byte(((u(xmc[x0+10]) & 0x3) << 6) | ((u(xmc[x0+11]) & 0x7) << 3) | (u(xmc[x0+12]) & 0x7))
	}
	putSub(5, 0, 0)
	putSub(12, 1, 13)
	putSub(19, 2, 26)
	putSub(26, 3, 39)
	return c
}
