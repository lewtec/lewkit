package avcodec

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/bits"
	"sync"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// FFmpeg 5.1.9 native Vorbis encoder (vorbisenc.c): long blocks only, quality 8.
// The C encoder rejects anything but stereo, so VorbisEncodeMono duplicates the
// mono signal onto both channels (dual-mono) and emits the same audio packets.

const (
	vorbisLog2Block = 11
	vorbisBlock     = 1 << vorbisLog2Block
	vorbisFrame     = vorbisBlock / 2
	vorbisInFrame   = 64
	vorbisQuality   = float32(64) // default global_quality path: 8*8
	vorbisFloorMult = 2
)

type fc32 struct {
	re, im float32
}

type bitWriter struct {
	buf []byte
	acc uint64
	n   int
}

func (b *bitWriter) put(n int, v uint32) {
	if n <= 0 {
		return
	}
	b.acc |= uint64(v) << uint(b.n)
	b.n += n
	for b.n >= 8 {
		b.buf = append(b.buf, byte(b.acc))
		b.acc >>= 8
		b.n -= 8
	}
}

func (b *bitWriter) flush() {
	if b.n > 0 {
		b.buf = append(b.buf, byte(b.acc))
		b.acc = 0
		b.n = 0
	}
}

func ilog(i int) int {
	return 31 - bits.LeadingZeros32((uint32(i)<<1)|1)
}

type codebook struct {
	ndims    int
	nentries int
	min      float32
	delta    float32
	lookup   int
	lens     []byte
	code     []uint32
	quant    []int
	dimv     []float32
	pow2     []float32
}

type floorClass struct {
	dim      int
	subclass int
	master   int
	books    []int
}

type floor1 struct {
	x, sort, low, high []int
	part               []int
	classes            []floorClass
	mult               int
	values             int
}

type residueInfo struct {
	books     [10][8]int
	maxes     [10][2]float32
	classbook int
	partSize  int
	begin     int
	end       int
	nclass    int
}

type vorbisMDCT struct {
	revtab []int
	tcos   []float32
	tsin   []float32
}

type vorbisModel struct {
	books []codebook
	floor floor1
	res   residueInfo
	mdct  vorbisMDCT
	win   []float32
	invDB []float32
	cos   [][]float32
	sqrtH float32
}

var getVorbis = sync.OnceValues(initVorbis)

func loadF32(bits []uint32) []float32 {
	o := make([]float32, len(bits))
	for i, u := range bits {
		o[i] = math.Float32frombits(u)
	}
	return o
}

func nthRoot(x, n uint32) uint32 {
	var ret uint32
	for {
		ret++
		j := ret
		for i := uint32(0); i < n-1; i++ {
			j *= ret
		}
		if j > x {
			return ret - 1
		}
	}
}

func lookupVals(lookup, dim, entries int) int {
	if lookup == 1 {
		return int(nthRoot(uint32(entries), uint32(dim)))
	}
	if lookup == 2 {
		return dim * entries
	}
	return 0
}

func len2vlc(lens []byte, codes []uint32) error {
	var exitAt [33]uint32
	exitAt[0] = 404
	p := 0
	for p < len(lens) && lens[p] == 0 {
		p++
	}
	if p == len(lens) {
		return nil
	}
	codes[p] = 0
	if lens[p] > 32 {
		return fmt.Errorf("%w: vorbis codebook", avutil.ErrInvalidData)
	}
	for i := 0; i < int(lens[p]); i++ {
		exitAt[i+1] = 1 << uint(i)
	}
	p++
	i := p
	for i < len(lens) && lens[i] == 0 {
		i++
	}
	if i == len(lens) {
		return nil
	}
	for ; p < len(lens); p++ {
		if lens[p] > 32 {
			return fmt.Errorf("%w: vorbis codebook", avutil.ErrInvalidData)
		}
		if lens[p] == 0 {
			continue
		}
		bi := int(lens[p])
		for bi > 0 && exitAt[bi] == 0 {
			bi--
		}
		if bi == 0 {
			return fmt.Errorf("%w: vorbis codebook", avutil.ErrInvalidData)
		}
		code := exitAt[bi]
		exitAt[bi] = 0
		for j := bi + 1; j <= int(lens[p]); j++ {
			exitAt[j] = code + (1 << uint(j-1))
		}
		codes[p] = code
	}
	for p = 1; p < 33; p++ {
		if exitAt[p] != 0 {
			return fmt.Errorf("%w: vorbis codebook", avutil.ErrInvalidData)
		}
	}
	return nil
}

func readyFloor(x []int) (sort, low, high []int, err error) {
	n := len(x)
	sort = make([]int, n)
	low = make([]int, n)
	high = make([]int, n)
	sort[0] = 0
	sort[1] = 1
	for i := 2; i < n; i++ {
		low[i] = 0
		high[i] = 1
		sort[i] = i
		for j := 2; j < i; j++ {
			tmp := x[j]
			if tmp < x[i] {
				if tmp > x[low[i]] {
					low[i] = j
				}
			} else if tmp < x[high[i]] {
				high[i] = j
			}
		}
	}
	for i := 0; i < n-1; i++ {
		for j := i + 1; j < n; j++ {
			if x[i] == x[j] {
				return nil, nil, nil, fmt.Errorf("%w: vorbis floor", avutil.ErrInvalidData)
			}
			if x[sort[i]] > x[sort[j]] {
				sort[i], sort[j] = sort[j], sort[i]
			}
		}
	}
	return sort, low, high, nil
}

func splitRadixPerm(i, n, inverse int) int {
	if n <= 2 {
		return i & 1
	}
	m := n >> 1
	if i&m == 0 {
		return splitRadixPerm(i, m, inverse) * 2
	}
	m >>= 1
	not := 0
	if i&m == 0 {
		not = 1
	}
	if inverse == not {
		return splitRadixPerm(i, m, inverse)*4 + 1
	}
	return splitRadixPerm(i, m, inverse)*4 - 1
}

func cmul(are, aim, bre, bim float32) (dre, dim float32) {
	dre = are*bre - aim*bim
	dim = are*bim + aim*bre
	return dre, dim
}

func butterflies(a0, a1, a2, a3 *fc32, t1, t2, t5, t6 float32) {
	t3 := t5 - t1
	t5 = t5 + t1
	a2.re, a0.re = a0.re-t5, a0.re+t5
	a3.im, a1.im = a1.im-t3, a1.im+t3
	t4 := t2 - t6
	t6 = t2 + t6
	a3.re, a1.re = a1.re-t4, a1.re+t4
	a2.im, a0.im = a0.im-t6, a0.im+t6
}

func transform(z []fc32, i0, i1, i2, i3 int, wre, wim float32) {
	a2 := z[i2]
	a3 := z[i3]
	t1, t2 := cmul(a2.re, a2.im, wre, -wim)
	t5, t6 := cmul(a3.re, a3.im, wre, wim)
	butterflies(&z[i0], &z[i1], &z[i2], &z[i3], t1, t2, t5, t6)
}

func transformZero(z []fc32, i0, i1, i2, i3 int) {
	t1 := z[i2].re
	t2 := z[i2].im
	t5 := z[i3].re
	t6 := z[i3].im
	butterflies(&z[i0], &z[i1], &z[i2], &z[i3], t1, t2, t5, t6)
}

func (m *vorbisModel) fft4(z []fc32) {
	t3 := z[0].re - z[1].re
	t1 := z[0].re + z[1].re
	t8 := z[3].re - z[2].re
	t6 := z[3].re + z[2].re
	z[2].re = t1 - t6
	z[0].re = t1 + t6
	t4 := z[0].im - z[1].im
	t2 := z[0].im + z[1].im
	t7 := z[2].im - z[3].im
	t5 := z[2].im + z[3].im
	z[3].im = t4 - t8
	z[1].im = t4 + t8
	z[3].re = t3 - t7
	z[1].re = t3 + t7
	z[2].im = t2 - t5
	z[0].im = t2 + t5
}

func (m *vorbisModel) fft8(z []fc32) {
	m.fft4(z)
	t1 := z[4].re - (-z[5].re)
	z[5].re = z[4].re + (-z[5].re)
	t2 := z[4].im - (-z[5].im)
	z[5].im = z[4].im + (-z[5].im)
	t5 := z[6].re - (-z[7].re)
	z[7].re = z[6].re + (-z[7].re)
	t6 := z[6].im - (-z[7].im)
	z[7].im = z[6].im + (-z[7].im)
	butterflies(&z[0], &z[2], &z[4], &z[6], t1, t2, t5, t6)
	transform(z, 1, 3, 5, 7, m.sqrtH, m.sqrtH)
}

func (m *vorbisModel) fft16(z []fc32) {
	c := m.cos[4]
	m.fft8(z)
	m.fft4(z[8:])
	m.fft4(z[12:])
	transformZero(z, 0, 4, 8, 12)
	transform(z, 2, 6, 10, 14, m.sqrtH, m.sqrtH)
	transform(z, 1, 5, 9, 13, c[1], c[3])
	transform(z, 3, 7, 11, 15, c[3], c[1])
}

func (m *vorbisModel) pass(z []fc32, wre []float32, n int) {
	o1 := 2 * n
	o2 := 4 * n
	o3 := 6 * n
	n--
	transformZero(z, 0, o1, o2, o3)
	transform(z, 1, o1+1, o2+1, o3+1, wre[1], wre[o1-1])
	zi := 0
	wi := 0
	for {
		zi += 2
		wi += 2
		transform(z, zi, zi+o1, zi+o2, zi+o3, wre[wi], wre[o1-wi])
		transform(z, zi+1, zi+o1+1, zi+o2+1, zi+o3+1, wre[wi+1], wre[o1-wi-1])
		n--
		if n == 0 {
			break
		}
	}
}

func (m *vorbisModel) fftN(z []fc32, n int) {
	switch n {
	case 4:
		m.fft4(z)
	case 8:
		m.fft8(z)
	case 16:
		m.fft16(z)
	default:
		n2 := n >> 1
		n4 := n >> 2
		m.fftN(z[:n2], n2)
		m.fftN(z[n2:n2+n4], n4)
		m.fftN(z[n2+n4:], n4)
		nb := bits.TrailingZeros(uint(n))
		m.pass(z, m.cos[nb], n4/2)
	}
}

func (m *vorbisModel) mdctCalc(out, in []float32) {
	const n = vorbisBlock
	const n2 = n >> 1
	const n4 = n >> 2
	const n8 = n >> 3
	const n3 = 3 * n4
	x := make([]fc32, n4)
	rev := m.mdct.revtab
	tcos := m.mdct.tcos
	tsin := m.mdct.tsin
	for i := 0; i < n8; i++ {
		re := -in[2*i+n3] - in[n3-1-2*i]
		im := -in[n4+2*i] + in[n4-1-2*i]
		j := rev[i]
		x[j].re, x[j].im = cmul(re, im, -tcos[i], tsin[i])

		re = in[2*i] - in[n2-1-2*i]
		im = -in[n2+2*i] - in[n-1-2*i]
		j = rev[n8+i]
		x[j].re, x[j].im = cmul(re, im, -tcos[n8+i], tsin[n8+i])
	}
	m.fftN(x, n4)
	for i := 0; i < n8; i++ {
		i1, r0 := cmul(x[n8-i-1].re, x[n8-i-1].im, -tsin[n8-i-1], -tcos[n8-i-1])
		i0, r1 := cmul(x[n8+i].re, x[n8+i].im, -tsin[n8+i], -tcos[n8+i])
		x[n8-i-1].re = r0
		x[n8-i-1].im = i0
		x[n8+i].re = r1
		x[n8+i].im = i1
	}
	for i := range x {
		out[2*i] = x[i].re
		out[2*i+1] = x[i].im
	}
}

func applyWindow(buf, win []float32) {
	const scale float32 = 1.0 / 512.0
	n := vorbisFrame
	for i := 0; i < n; i++ {
		buf[i] *= win[i]
		buf[i] *= scale
	}
	for i := 0; i < n; i++ {
		buf[n+i] *= win[n-1-i]
		buf[n+i] *= scale
	}
}

func fabs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func fmax32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func clipU8(a int) int {
	if a < 0 {
		return 0
	}
	if a > 255 {
		return 255
	}
	return a
}

func iabs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func renderPoint(x0, y0, x1, y1, x int) int {
	return y0 + (x-x0)*(y1-y0)/(x1-x0)
}

func (m *vorbisModel) renderLineUnrolled(x, y, x1, sy, ady, adx int, buf []float32) {
	err := -adx
	x -= x1 - 1
	base := x1 - 1
	for {
		x++
		if x >= 0 {
			break
		}
		err += ady
		if err >= 0 {
			err += ady - adx
			y += sy
			buf[base+x] = m.invDB[clipU8(y)]
			x++
		}
		buf[base+x] = m.invDB[clipU8(y)]
	}
	if x <= 0 {
		if err+ady >= 0 {
			y += sy
		}
		buf[base+x] = m.invDB[clipU8(y)]
	}
}

func (m *vorbisModel) renderLine(x0, y0, x1, y1 int, buf []float32) {
	dy := y1 - y0
	adx := x1 - x0
	ady := iabs(dy)
	sy := 1
	if dy < 0 {
		sy = -1
	}
	buf[x0] = m.invDB[clipU8(y0)]
	if ady*2 <= adx {
		m.renderLineUnrolled(x0, y0, x1, sy, ady, adx, buf)
		return
	}
	base := dy / adx
	x := x0
	y := y0
	err := -adx
	ady -= iabs(base) * adx
	for {
		x++
		if x >= x1 {
			break
		}
		y += base
		err += ady
		if err >= 0 {
			err -= adx
			y += sy
		}
		buf[x] = m.invDB[clipU8(y)]
	}
}

func (m *vorbisModel) renderFloor(posts, coded []int, out []float32, samples int) {
	fc := &m.floor
	lx := 0
	ly := posts[0] * fc.mult
	for i := 1; i < fc.values; i++ {
		pos := fc.sort[i]
		if coded[pos] != 0 {
			x1 := fc.x[pos]
			y1 := posts[pos] * fc.mult
			if lx < samples {
				xend := x1
				if xend > samples {
					xend = samples
				}
				m.renderLine(lx, ly, xend, y1, out)
			}
			lx = x1
			ly = y1
		}
		if lx >= samples {
			break
		}
	}
	if lx < samples {
		m.renderLine(lx, ly, samples, ly, out)
	}
}

func (m *vorbisModel) floorAverage(coeffs []float32, i int) float32 {
	fc := &m.floor
	i0 := i - 1
	if i0 < 0 {
		i0 = 0
	}
	i1 := i + 1
	if i1 > fc.values-1 {
		i1 = fc.values - 1
	}
	begin := fc.x[fc.sort[i0]]
	end := fc.x[fc.sort[i1]]
	var average float32
	for j := begin; j < end; j++ {
		average += fabs32(coeffs[j])
	}
	return average / float32(end-begin)
}

func (m *vorbisModel) floorFit(coeffs []float32) []int {
	fc := &m.floor
	rangeN := 255/fc.mult + 1
	averages := make([]float32, fc.values)
	var tot float32
	for i := 0; i < fc.values; i++ {
		averages[i] = m.floorAverage(coeffs, i)
		tot += averages[i]
	}
	tot /= float32(fc.values)
	tot /= vorbisQuality
	posts := make([]int, fc.values)
	for i := 0; i < fc.values; i++ {
		position := fc.x[fc.sort[i]]
		average := averages[i]
		exp := float64(float32(position) * float32(0.005))
		average = float32(math.Sqrt(float64(tot*average)) * math.Pow(1.25, exp))
		j := 0
		for ; j < rangeN-1; j++ {
			if m.invDB[j*fc.mult] > average {
				break
			}
		}
		posts[fc.sort[i]] = j
	}
	return posts
}

func putCodeword(b *bitWriter, cb *codebook, entry int) error {
	if entry < 0 || entry >= cb.nentries || cb.lens[entry] == 0 {
		return fmt.Errorf("%w: vorbis codeword", avutil.ErrInvalidData)
	}
	b.put(int(cb.lens[entry]), cb.code[entry])
	return nil
}

func (m *vorbisModel) floorEncode(b *bitWriter, posts []int, floorOut []float32, samples int) error {
	fc := &m.floor
	rangeN := 255/fc.mult + 1
	coded := make([]int, fc.values)
	if ilog(rangeN-1) < 1 {
		return fmt.Errorf("%w: vorbis floor", avutil.ErrInvalidData)
	}
	b.put(1, 1)
	nb := ilog(rangeN - 1)
	b.put(nb, uint32(posts[0]))
	b.put(nb, uint32(posts[1]))
	coded[0], coded[1] = 1, 1
	for i := 2; i < fc.values; i++ {
		predicted := renderPoint(fc.x[fc.low[i]], posts[fc.low[i]], fc.x[fc.high[i]], posts[fc.high[i]], fc.x[i])
		highroom := rangeN - predicted
		lowroom := predicted
		room := highroom
		if lowroom < room {
			room = lowroom
		}
		if predicted == posts[i] {
			coded[i] = 0
			continue
		}
		if coded[fc.low[i]] == 0 {
			coded[fc.low[i]] = -1
		}
		if coded[fc.high[i]] == 0 {
			coded[fc.high[i]] = -1
		}
		if posts[i] > predicted {
			if posts[i]-predicted > room {
				coded[i] = posts[i] - predicted + lowroom
			} else {
				coded[i] = (posts[i] - predicted) << 1
			}
		} else {
			if predicted-posts[i] > room {
				coded[i] = predicted - posts[i] + highroom - 1
			} else {
				coded[i] = ((predicted - posts[i]) << 1) - 1
			}
		}
	}
	counter := 2
	for pi := range fc.part {
		c := &fc.classes[fc.part[pi]]
		cval := 0
		csub := 1 << c.subclass
		if c.subclass != 0 {
			book := &m.books[c.master]
			cshift := 0
			for k := 0; k < c.dim; k++ {
				l := 0
				for ; l < csub; l++ {
					maxval := 1
					if c.books[l] != -1 {
						maxval = m.books[c.books[l]].nentries
					}
					if coded[counter+k] < maxval {
						break
					}
				}
				if l == csub {
					return fmt.Errorf("%w: vorbis floor class", avutil.ErrInvalidData)
				}
				cval |= l << cshift
				cshift += c.subclass
			}
			if err := putCodeword(b, book, cval); err != nil {
				return err
			}
		}
		for k := 0; k < c.dim; k++ {
			bookID := c.books[cval&(csub-1)]
			entry := coded[counter]
			counter++
			cval >>= c.subclass
			if bookID == -1 {
				continue
			}
			if entry == -1 {
				entry = 0
			}
			if err := putCodeword(b, &m.books[bookID], entry); err != nil {
				return err
			}
		}
	}
	m.renderFloor(posts, coded, floorOut, samples)
	return nil
}

func putVector(b *bitWriter, book *codebook, num []float32) ([]float32, error) {
	entry := -1
	distance := float32(math.MaxFloat32)
	nd := book.ndims
	for i := 0; i < book.nentries; i++ {
		if book.lens[i] == 0 {
			continue
		}
		vec := book.dimv[i*nd : (i+1)*nd]
		d := book.pow2[i]
		for j := 0; j < nd; j++ {
			d -= vec[j] * num[j]
		}
		if distance > d {
			entry = i
			distance = d
		}
	}
	if err := putCodeword(b, book, entry); err != nil {
		return nil, err
	}
	return book.dimv[entry*nd : (entry+1)*nd], nil
}

func (m *vorbisModel) residueEncode(b *bitWriter, coeffs []float32, samples, realCh int) error {
	rc := &m.res
	psize := rc.partSize
	partitions := (rc.end - rc.begin) / psize
	classwords := m.books[rc.classbook].ndims
	classes := make([]int, partitions)
	for p := 0; p < partitions; p++ {
		var max1, max2 float32
		s := rc.begin + p*psize
		for k := s; k < s+psize; k += 2 {
			idx := k / realCh
			max1 = fmax32(max1, fabs32(coeffs[idx]))
			max2 = fmax32(max2, fabs32(coeffs[samples+idx]))
		}
		i := 0
		for ; i < rc.nclass-1; i++ {
			if max1 < rc.maxes[i][0] && max2 < rc.maxes[i][1] {
				break
			}
		}
		classes[p] = i
	}
	for pass := 0; pass < 8; pass++ {
		p := 0
		for p < partitions {
			if pass == 0 {
				book := &m.books[rc.classbook]
				entry := 0
				for i := 0; i < classwords; i++ {
					entry *= rc.nclass
					entry += classes[p+i]
				}
				if err := putCodeword(b, book, entry); err != nil {
					return err
				}
			}
			for i := 0; i < classwords && p < partitions; i, p = i+1, p+1 {
				nbook := rc.books[classes[p]][pass]
				if nbook < 0 {
					continue
				}
				book := &m.books[nbook]
				s0 := rc.begin + p*psize
				a1 := (s0 % realCh) * samples
				b1 := s0 / realCh
				total := realCh * samples
				for k := 0; k < psize; k += book.ndims {
					a2, b2 := a1, b1
					vec := make([]float32, book.ndims)
					for dim := book.ndims; dim > 0; dim-- {
						vec[book.ndims-dim] = coeffs[a2+b2]
						a2 += samples
						if a2 == total {
							a2 = 0
							b2++
						}
					}
					pv, err := putVector(b, book, vec)
					if err != nil {
						return err
					}
					for dim := book.ndims; dim > 0; dim-- {
						coeffs[a1+b1] -= pv[book.ndims-dim]
						a1 += samples
						if a1 == total {
							a1 = 0
							b1++
						}
					}
				}
			}
		}
	}
	return nil
}

func (m *vorbisModel) encodeBlock(saved, cur []float32) ([]byte, error) {
	bw := &bitWriter{}
	bw.put(1, 0)
	bw.put(ilog(1), 1)
	bw.put(1, 1)
	bw.put(1, 1)
	coeffs := make([]float32, 2*vorbisFrame)
	floors := make([]float32, 2*vorbisFrame)
	win := make([]float32, vorbisBlock)
	for ch := 0; ch < 2; ch++ {
		copy(win[:vorbisFrame], saved)
		copy(win[vorbisFrame:], cur)
		applyWindow(win, m.win)
		off := ch * vorbisFrame
		m.mdctCalc(coeffs[off:off+vorbisFrame], win)
		posts := m.floorFit(coeffs[off : off+vorbisFrame])
		if err := m.floorEncode(bw, posts, floors[off:off+vorbisFrame], vorbisFrame); err != nil {
			return nil, err
		}
	}
	for i := range coeffs {
		coeffs[i] /= floors[i]
	}
	mag := coeffs[:vorbisFrame]
	ang := coeffs[vorbisFrame:]
	for j := 0; j < vorbisFrame; j++ {
		a := ang[j]
		ang[j] -= mag[j]
		if mag[j] > 0 {
			ang[j] = -ang[j]
		}
		if ang[j] < 0 {
			mag[j] = a
		}
	}
	if err := m.residueEncode(bw, coeffs, vorbisFrame, 2); err != nil {
		return nil, err
	}
	bw.flush()
	return bw.buf, nil
}

type bookSpec struct {
	dim, length, realLen, lookup int
	min, delta                   float32
}

func vorbisBookSpecs() []bookSpec {
	z := float32(0)
	return []bookSpec{
		{2, 16, 16, 0, z, z},
		{2, 8, 8, 0, z, z},
		{2, 256, 256, 0, z, z},
		{2, 64, 64, 0, z, z},
		{2, 128, 128, 0, z, z},
		{2, 32, 32, 0, z, z},
		{2, 96, 96, 0, z, z},
		{2, 32, 32, 0, z, z},
		{2, 96, 96, 0, z, z},
		{2, 17, 17, 0, z, z},
		{2, 32, 32, 0, z, z},
		{2, 78, 78, 0, z, z},
		{2, 17, 17, 0, z, z},
		{2, 32, 32, 0, z, z},
		{2, 78, 78, 0, z, z},
		{2, 100, 100, 0, z, z},
		{8, 1641, 6561, 1, -1, 1},
		{4, 443, 625, 1, -2, 1},
		{4, 105, 625, 1, -2, 1},
		{2, 68, 81, 1, -4, 1},
		{2, 81, 81, 1, -4, 1},
		{2, 289, 289, 1, -8, 1},
		{4, 81, 81, 1, -11, 11},
		{2, 121, 121, 1, -5, 1},
		{2, 169, 169, 1, -30, 5},
		{2, 25, 25, 1, -2, 1},
		{2, 169, 169, 1, -1530, 255},
		{2, 225, 225, 1, -119, 17},
		{2, 289, 289, 1, -8, 1},
	}
}

func initVorbis() (*vorbisModel, error) {
	m := &vorbisModel{}
	m.win = loadF32(vorbisWin2048Bits)
	m.invDB = loadF32(vorbisInvDBBits)
	m.sqrtH = math.Float32frombits(vorbisSqrtHalfBits)
	m.cos = make([][]float32, 10)
	for nb := 4; nb <= 9; nb++ {
		a := vorbisCosOff[nb-4]
		b := vorbisCosOff[nb-4+1]
		m.cos[nb] = loadF32(vorbisCosBits[a:b])
	}
	nfft := vorbisBlock >> 2
	m.mdct.revtab = make([]int, nfft)
	for i := 0; i < nfft; i++ {
		k := (-splitRadixPerm(i, nfft, 0)) & (nfft - 1)
		m.mdct.revtab[k] = i
	}
	m.mdct.tcos = loadF32(vorbisTCosBits)
	m.mdct.tsin = loadF32(vorbisTSinBits)

	specs := vorbisBookSpecs()
	m.books = make([]codebook, len(specs))
	lpos := 0
	qpos := 0
	for bi, sp := range specs {
		cb := &m.books[bi]
		cb.ndims = sp.dim
		cb.nentries = sp.realLen
		cb.min = sp.min
		cb.delta = sp.delta
		cb.lookup = sp.lookup
		cb.lens = make([]byte, sp.realLen)
		copy(cb.lens, vorbisCodebookLens[lpos:lpos+sp.length])
		lpos += sp.length
		cb.code = make([]uint32, sp.realLen)
		if err := len2vlc(cb.lens, cb.code); err != nil {
			return nil, err
		}
		if sp.lookup != 0 {
			vals := lookupVals(sp.lookup, sp.dim, sp.realLen)
			cb.quant = make([]int, vals)
			for i := 0; i < vals; i++ {
				cb.quant[i] = int(vorbisQuantTables[qpos])
				qpos++
			}
			cb.dimv = make([]float32, sp.realLen*sp.dim)
			cb.pow2 = make([]float32, sp.realLen)
			for i := 0; i < sp.realLen; i++ {
				var last float32
				div := 1
				for j := 0; j < sp.dim; j++ {
					off := (i / div) % vals
					v := last + cb.min + float32(cb.quant[off])*cb.delta
					cb.dimv[i*sp.dim+j] = v
					cb.pow2[i] += v * v
					div *= vals
				}
				cb.pow2[i] /= 2
			}
		}
	}
	if lpos != len(vorbisCodebookLens) || qpos != len(vorbisQuantTables) {
		return nil, fmt.Errorf("%w: vorbis tables", avutil.ErrInvalidData)
	}

	xs := []int{0, 1 << (vorbisLog2Block - 1)}
	xs = append(xs, 93, 23, 372, 6, 46, 186, 750, 14, 33, 65,
		130, 260, 556, 3, 10, 18, 28, 39, 55, 79,
		111, 158, 220, 312, 464, 650, 850)
	sort, low, high, err := readyFloor(xs)
	if err != nil {
		return nil, err
	}
	classes := []floorClass{
		{3, 0, 0, []int{4}},
		{4, 1, 0, []int{5, 6}},
		{3, 1, 1, []int{7, 8}},
		{4, 2, 2, []int{-1, 9, 10, 11}},
		{3, 2, 3, []int{-1, 12, 13, 14}},
	}
	m.floor = floor1{
		x: xs, sort: sort, low: low, high: high,
		part:    []int{0, 1, 2, 2, 3, 3, 4, 4},
		classes: classes,
		mult:    vorbisFloorMult,
		values:  len(xs),
	}
	m.res = residueInfo{
		classbook: 15,
		partSize:  32,
		begin:     0,
		end:       1600,
		nclass:    10,
		books: [10][8]int{
			{-1, -1, -1, -1, -1, -1, -1, -1},
			{-1, -1, 16, -1, -1, -1, -1, -1},
			{-1, -1, 17, -1, -1, -1, -1, -1},
			{-1, -1, 18, -1, -1, -1, -1, -1},
			{-1, -1, 19, -1, -1, -1, -1, -1},
			{-1, -1, 20, -1, -1, -1, -1, -1},
			{-1, -1, 21, -1, -1, -1, -1, -1},
			{22, 23, -1, -1, -1, -1, -1, -1},
			{24, 25, -1, -1, -1, -1, -1, -1},
			{26, 27, 28, -1, -1, -1, -1, -1},
		},
	}
	for i := 0; i < m.res.nclass; i++ {
		j := 0
		for ; j < 8; j++ {
			if m.res.books[i][j] != -1 {
				break
			}
		}
		if j == 8 {
			continue
		}
		cb := &m.books[m.res.books[i][j]]
		for e := 0; e < cb.nentries; e++ {
			if cb.lens[e] == 0 {
				continue
			}
			a := fabs32(cb.dimv[e*cb.ndims])
			if a > m.res.maxes[i][0] {
				m.res.maxes[i][0] = a
			}
			a = fabs32(cb.dimv[e*cb.ndims+1])
			if a > m.res.maxes[i][1] {
				m.res.maxes[i][1] = a
			}
		}
	}
	for i := 0; i < m.res.nclass; i++ {
		m.res.maxes[i][0] = float32(float64(m.res.maxes[i][0]) + 0.8)
		m.res.maxes[i][1] = float32(float64(m.res.maxes[i][1]) + 0.8)
	}
	return m, nil
}

// VorbisEncodeMono encodes mono s16 PCM with FFmpeg's native Vorbis encoder
// defaults (long block, quality 8, experimental). The C encoder only accepts
// stereo, so the mono samples are duplicated onto both channels. The returned
// bytes are the concatenated Vorbis audio packets, without Ogg pages or headers.
// Sample blocks follow the encoder: 64-sample input frames, 1024-sample long
// blocks, plus the initial-padding flush packet.
func VorbisEncodeMono(pcm []int16, rate int) ([]byte, error) {
	if rate <= 0 || rate > 0x7fffffff {
		return nil, fmt.Errorf("%w: vorbis sample rate", avutil.ErrInvalidData)
	}
	m, err := getVorbis()
	if err != nil {
		return nil, err
	}
	if len(pcm) == 0 {
		return nil, nil
	}
	npad := len(pcm)
	if r := npad % vorbisInFrame; r != 0 {
		npad += vorbisInFrame - r
	}
	samples := make([]float32, npad)
	for i, s := range pcm {
		samples[i] = float32(s) / 32768
	}
	npack := (vorbisFrame + npad + vorbisFrame - 1) / vorbisFrame
	saved := make([]float32, vorbisFrame)
	var out []byte
	pos := 0
	for pkt := 0; pkt < npack; pkt++ {
		cur := make([]float32, vorbisFrame)
		if pos < len(samples) {
			n := vorbisFrame
			if pos+n > len(samples) {
				n = len(samples) - pos
			}
			copy(cur, samples[pos:pos+n])
			pos += n
		}
		raw, err := m.encodeBlock(saved, cur)
		if err != nil {
			return nil, err
		}
		out = append(out, raw...)
		saved = cur
	}
	return out, nil
}

// vorbisEnc buffers mono s16 and emits one packet of concatenated audio
// frames on Flush. The native encoder is stereo internally: the mono signal
// is duplicated, which is the dual-mono path VorbisEncodeMono compares.
type vorbisEnc struct {
	pcm  []int16
	rate int
}

func newVorbisEnc(par Parameters) (*vorbisEnc, error) {
	rate := par.SampleRate
	if rate <= 0 {
		rate = 44100
	}
	if rate > 0x7fffffff {
		return nil, fmt.Errorf("%w: vorbis sample rate", avutil.ErrInvalidData)
	}
	if par.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: vorbis channels", avutil.ErrInvalidData)
	}
	return &vorbisEnc{rate: rate}, nil
}

func (c *vorbisEnc) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil || frame.NbSamples == 0 {
		return nil, nil
	}
	if frame.SampleFmt != avutil.SampleS16 || frame.Layout.Channels() > 1 {
		return nil, fmt.Errorf("%w: vorbis", avutil.ErrInvalidData)
	}
	if len(frame.Data) == 0 || len(frame.Data[0]) < frame.NbSamples*2 {
		return nil, fmt.Errorf("%w: vorbis samples", avutil.ErrInvalidData)
	}
	for i := 0; i < frame.NbSamples; i++ {
		c.pcm = append(c.pcm, int16(binary.LittleEndian.Uint16(frame.Data[0][i*2:])))
	}
	return nil, nil
}

func (c *vorbisEnc) Flush() ([]*avutil.Packet, error) {
	n := len(c.pcm)
	raw, err := VorbisEncodeMono(c.pcm, c.rate)
	c.pcm = nil
	if err != nil || len(raw) == 0 {
		return nil, err
	}
	return []*avutil.Packet{{Data: raw, Duration: int64(n)}}, nil
}
