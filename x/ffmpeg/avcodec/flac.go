package avcodec

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

type flacCodec struct {
	par      Parameters
	pcm      []int32
	rate     int
	channels int
}

var (
	flacRate = [...]int{0, 88200, 176400, 192000, 8000, 16000, 22050, 24000, 32000, 44100, 48000, 96000, 0, 0, 0, 0}
	flacBits = [...]int{0, 8, 12, 0, 16, 20, 24, 0}
	flacBlock = [...]int{0, 192, 576, 1152, 2304, 4608, 0, 0, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768}
)

type bitr struct {
	b []byte
	i int
}

func (r *bitr) bitsLeft() int { return len(r.b)*8 - r.i }

func (r *bitr) u(n int) (uint32, error) {
	if n < 0 || n > 32 || r.bitsLeft() < n {
		return 0, fmt.Errorf("%w: flac bits", avutil.ErrInvalidData)
	}
	var v uint32
	for k := 0; k < n; k++ {
		bytei := r.i / 8
		bit := 7 - r.i%8
		v = v<<1 | uint32((r.b[bytei]>>bit)&1)
		r.i++
	}
	return v, nil
}

func (r *bitr) s(n int) (int32, error) {
	v, err := r.u(n)
	if err != nil {
		return 0, err
	}
	shift := 32 - n
	return int32(v<<shift) >> shift, nil
}

func (r *bitr) utf8() (uint64, error) {
	b, err := r.u(8)
	if err != nil {
		return 0, err
	}
	if b < 128 {
		return uint64(b), nil
	}
	ones := 0
	for mask := uint32(0x80); mask > 0 && b&mask != 0; mask >>= 1 {
		ones++
	}
	if ones < 2 || ones > 8 {
		return 0, fmt.Errorf("%w: flac utf8", avutil.ErrInvalidData)
	}
	v := uint64(b & (0xff >> ones))
	for i := 1; i < ones; i++ {
		c, err := r.u(8)
		if err != nil || c&0xc0 != 0x80 {
			return 0, fmt.Errorf("%w: flac utf8", avutil.ErrInvalidData)
		}
		v = v<<6 | uint64(c&0x3f)
	}
	return v, nil
}

type bitw struct {
	b   []byte
	acc uint32
	n   int
}

func (w *bitw) u(v uint32, n int) {
	for k := n - 1; k >= 0; k-- {
		bit := (v >> k) & 1
		w.acc = w.acc<<1 | bit
		w.n++
		if w.n == 8 {
			w.b = append(w.b, byte(w.acc))
			w.acc = 0
			w.n = 0
		}
	}
}

func (w *bitw) s(v int32, n int) { w.u(uint32(v), n) }

func (w *bitw) bytes() []byte {
	if w.n > 0 {
		w.b = append(w.b, byte(w.acc<<(8-w.n)))
		w.acc = 0
		w.n = 0
	}
	return w.b
}

func crc8ATM(data []byte) byte {
	var crc byte
	for _, b := range data {
		crc ^= b
		for i := 0; i < 8; i++ {
			if crc&0x80 != 0 {
				crc = (crc << 1) ^ 0x07
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

func crc16ANSI(data []byte) uint16 {
	var crc uint16
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x8005
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

func (c *flacCodec) Decode(pkt *avutil.Packet) ([]*avutil.Frame, error) {
	if pkt == nil || len(pkt.Data) == 0 {
		return nil, nil
	}
	data := pkt.Data
	rate := c.par.SampleRate
	channels := c.par.Layout.Channels()
	bps := 16
	if len(data) >= 4 && string(data[:4]) == "fLaC" {
		info, rest, err := parseStreamInfo(data)
		if err != nil {
			return nil, err
		}
		rate, channels, bps = info.rate, info.channels, info.bps
		data = rest
	}
	if rate == 0 {
		rate = 44100
	}
	if channels == 0 {
		channels = 1
	}
	var all []int32
	for len(data) > 0 {
		if data[0] != 0xff {
			break
		}
		frame, n, fr, fc, fb, err := decodeFLACFrame(data)
		if err != nil {
			return nil, err
		}
		if fr != 0 {
			rate = fr
		}
		if fc != 0 {
			channels = fc
		}
		if fb != 0 {
			bps = fb
		}
		all = append(all, frame...)
		data = data[n:]
	}
	if len(all) == 0 || len(all)%channels != 0 {
		return nil, fmt.Errorf("%w: flac", avutil.ErrInvalidData)
	}
	samples := len(all) / channels
	format := avutil.SampleS16
	if bps > 16 {
		format = avutil.SampleS32
	}
	out, err := avutil.NewAudio(format, rate, layoutOf(c.par.Layout, channels), samples)
	if err != nil {
		return nil, err
	}
	for i, sample := range all {
		if format == avutil.SampleS16 {
			v := int16(sample)
			out.Data[0][i*2] = byte(v)
			out.Data[0][i*2+1] = byte(uint16(v) >> 8)
		} else {
			binary.LittleEndian.PutUint32(out.Data[0][i*4:], uint32(sample))
		}
	}
	out.PTS = pkt.PTS
	out.Duration = int64(samples)
	return []*avutil.Frame{out}, nil
}

type streamInfo struct {
	rate, channels, bps, block int
}

func parseStreamInfo(data []byte) (streamInfo, []byte, error) {
	i := 4
	var info streamInfo
	for {
		if i+4 > len(data) {
			return info, nil, fmt.Errorf("%w: flac metadata", avutil.ErrInvalidData)
		}
		last := data[i]&0x80 != 0
		kind := data[i] & 0x7f
		size := int(data[i+1])<<16 | int(data[i+2])<<8 | int(data[i+3])
		i += 4
		if i+size > len(data) {
			return info, nil, fmt.Errorf("%w: flac metadata", avutil.ErrInvalidData)
		}
		block := data[i : i+size]
		i += size
		if kind == 0 && len(block) >= 18 {
			v := uint64(block[10])<<32 | uint64(block[11])<<24 | uint64(block[12])<<16 | uint64(block[13])<<8 | uint64(block[14])
			info.rate = int(v >> 12)
			// bytes 10-17 hold rate(20) channels(3) bps(5) samples(36)
			bits := uint64(block[10])<<56 | uint64(block[11])<<48 | uint64(block[12])<<40 | uint64(block[13])<<32 |
				uint64(block[14])<<24 | uint64(block[15])<<16 | uint64(block[16])<<8 | uint64(block[17])
			info.rate = int((bits >> 44) & 0xFFFFF)
			info.channels = int((bits>>41)&7) + 1
			info.bps = int((bits>>36)&31) + 1
		}
		if last {
			break
		}
	}
	return info, data[i:], nil
}

func decodeFLACFrame(data []byte) ([]int32, int, int, int, int, error) {
	r := &bitr{b: data}
	sync, err := r.u(15)
	if err != nil || sync != 0x7FFC {
		return nil, 0, 0, 0, 0, fmt.Errorf("%w: flac sync", avutil.ErrInvalidData)
	}
	if _, err = r.u(1); err != nil {
		return nil, 0, 0, 0, 0, err
	}
	bsCode, err := r.u(4)
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}
	srCode, err := r.u(4)
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}
	chCode, err := r.u(4)
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}
	bpsCode, err := r.u(3)
	if err != nil || bpsCode == 3 || bpsCode == 7 {
		return nil, 0, 0, 0, 0, fmt.Errorf("%w: flac bps", avutil.ErrInvalidData)
	}
	if bit, err := r.u(1); err != nil || bit != 0 {
		return nil, 0, 0, 0, 0, fmt.Errorf("%w: flac pad", avutil.ErrInvalidData)
	}
	if _, err = r.utf8(); err != nil {
		return nil, 0, 0, 0, 0, err
	}
	block := flacBlock[bsCode]
	switch bsCode {
	case 0:
		return nil, 0, 0, 0, 0, fmt.Errorf("%w: flac block", avutil.ErrInvalidData)
	case 6:
		v, err := r.u(8)
		if err != nil {
			return nil, 0, 0, 0, 0, err
		}
		block = int(v) + 1
	case 7:
		v, err := r.u(16)
		if err != nil {
			return nil, 0, 0, 0, 0, err
		}
		block = int(v) + 1
	}
	rate := flacRate[srCode]
	switch srCode {
	case 12:
		v, err := r.u(8)
		if err != nil {
			return nil, 0, 0, 0, 0, err
		}
		rate = int(v) * 1000
	case 13:
		v, err := r.u(16)
		if err != nil {
			return nil, 0, 0, 0, 0, err
		}
		rate = int(v)
	case 14:
		v, err := r.u(16)
		if err != nil {
			return nil, 0, 0, 0, 0, err
		}
		rate = int(v) * 10
	}
	bps := flacBits[bpsCode]
	channels := int(chCode) + 1
	mode := 0
	if chCode >= 8 {
		if chCode > 10 {
			return nil, 0, 0, 0, 0, fmt.Errorf("%w: flac channels", avutil.ErrInvalidData)
		}
		channels = 2
		mode = int(chCode) - 7
	}
	if _, err = r.u(8); err != nil {
		return nil, 0, 0, 0, 0, err
	}
	headerBytes := r.i / 8
	if crc8ATM(data[:headerBytes]) != 0 {
		return nil, 0, 0, 0, 0, fmt.Errorf("%w: flac crc8", avutil.ErrInvalidData)
	}
	planes := make([][]int32, channels)
	for ch := 0; ch < channels; ch++ {
		cbps := bps
		if mode == 2 && ch == 0 || (mode == 1 || mode == 3) && ch == 1 {
			cbps++
		}
		samples, err := decodeSubframe(r, block, cbps)
		if err != nil {
			return nil, 0, 0, 0, 0, err
		}
		planes[ch] = samples
	}
	if r.i%8 != 0 {
		if _, err = r.u(8 - r.i%8); err != nil {
			return nil, 0, 0, 0, 0, err
		}
	}
	end := r.i / 8
	if end+2 > len(data) {
		return nil, 0, 0, 0, 0, fmt.Errorf("%w: flac crc", avutil.ErrInvalidData)
	}
	want := binary.BigEndian.Uint16(data[end : end+2])
	if crc16ANSI(data[:end]) != want {
		return nil, 0, 0, 0, 0, fmt.Errorf("%w: flac crc16", avutil.ErrInvalidData)
	}
	switch mode {
	case 1:
		for i := 0; i < block; i++ {
			a := planes[0][i]
			b := planes[1][i]
			planes[1][i] = a - b
		}
	case 2:
		for i := 0; i < block; i++ {
			a := planes[0][i]
			b := planes[1][i]
			planes[0][i] = a + b
		}
	case 3:
		for i := 0; i < block; i++ {
			mid := uint32(planes[0][i])
			side := planes[1][i]
			mid -= uint32(side >> 1)
			planes[0][i] = int32(mid + uint32(side))
			planes[1][i] = int32(mid)
		}
	}
	out := make([]int32, block*channels)
	for i := 0; i < block; i++ {
		for ch := 0; ch < channels; ch++ {
			out[i*channels+ch] = planes[ch][i]
		}
	}
	return out, end + 2, rate, channels, bps, nil
}

func decodeSubframe(r *bitr, block, bps int) ([]int32, error) {
	pad, err := r.u(1)
	if err != nil || pad != 0 {
		return nil, fmt.Errorf("%w: flac subframe", avutil.ErrInvalidData)
	}
	kind, err := r.u(6)
	if err != nil {
		return nil, err
	}
	wasted := 0
	flag, err := r.u(1)
	if err != nil {
		return nil, err
	}
	if flag == 1 {
		for {
			bit, err := r.u(1)
			if err != nil {
				return nil, err
			}
			wasted++
			if bit == 1 {
				break
			}
		}
		bps -= wasted
	}
	out := make([]int32, block)
	switch {
	case kind == 0:
		v, err := r.s(bps)
		if err != nil {
			return nil, err
		}
		for i := range out {
			out[i] = v
		}
	case kind == 1:
		for i := range out {
			v, err := r.s(bps)
			if err != nil {
				return nil, err
			}
			out[i] = v
		}
	case kind >= 8 && kind <= 12:
		if err := decodeFixed(r, out, int(kind&^8), bps); err != nil {
			return nil, err
		}
	case kind >= 32:
		if err := decodeLPC(r, out, int(kind&^32)+1, bps); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%w: flac subframe %d", avutil.ErrInvalidData, kind)
	}
	if wasted > 0 {
		for i := range out {
			out[i] <<= wasted
		}
	}
	return out, nil
}

func rice(r *bitr, k int) (int32, error) {
	q := 0
	for {
		bit, err := r.u(1)
		if err != nil {
			return 0, err
		}
		if bit == 1 {
			break
		}
		q++
		if q > 1<<20 {
			return 0, fmt.Errorf("%w: flac rice", avutil.ErrInvalidData)
		}
	}
	rest := uint32(0)
	if k > 0 {
		v, err := r.u(k)
		if err != nil {
			return 0, err
		}
		rest = v
	}
	u := (uint32(q) << k) | rest
	return int32(u>>1) ^ -int32(u&1), nil
}

func decodeResidual(r *bitr, out []int32, order int) error {
	method, err := r.u(2)
	if err != nil || method > 1 {
		return fmt.Errorf("%w: flac residual", avutil.ErrInvalidData)
	}
	riceOrder, err := r.u(4)
	if err != nil {
		return err
	}
	partitions := 1 << riceOrder
	samples := len(out) >> riceOrder
	if samples<<riceOrder != len(out) || order > samples {
		return fmt.Errorf("%w: flac residual", avutil.ErrInvalidData)
	}
	paramBits := 4 + int(method)
	esc := (1 << paramBits) - 1
	i := order
	for part := 0; part < partitions; part++ {
		param, err := r.u(paramBits)
		if err != nil {
			return err
		}
		if int(param) == esc {
			bits, err := r.u(5)
			if err != nil {
				return err
			}
			for ; i < samples; i++ {
				v, err := r.s(int(bits))
				if err != nil {
					return err
				}
				out[i] = v
			}
		} else {
			for ; i < samples; i++ {
				v, err := rice(r, int(param))
				if err != nil {
					return err
				}
				out[i] = v
			}
		}
		i = 0
		out = out[samples:]
	}
	return nil
}

func decodeFixed(r *bitr, out []int32, order, bps int) error {
	for i := 0; i < order; i++ {
		v, err := r.s(bps)
		if err != nil {
			return err
		}
		out[i] = v
	}
	if err := decodeResidual(r, out, order); err != nil {
		return err
	}
	var a, b, c, d int32
	if order > 0 {
		a = out[order-1]
	}
	if order > 1 {
		b = a - out[order-2]
	}
	if order > 2 {
		c = b - out[order-2] + out[order-3]
	}
	if order > 3 {
		d = c - out[order-2] + 2*out[order-3] - out[order-4]
	}
	switch order {
	case 1:
		for i := order; i < len(out); i++ {
			a += out[i]
			out[i] = a
		}
	case 2:
		for i := order; i < len(out); i++ {
			b += out[i]
			a += b
			out[i] = a
		}
	case 3:
		for i := order; i < len(out); i++ {
			c += out[i]
			b += c
			a += b
			out[i] = a
		}
	case 4:
		for i := order; i < len(out); i++ {
			d += out[i]
			c += d
			b += c
			a += b
			out[i] = a
		}
	}
	return nil
}

func decodeLPC(r *bitr, out []int32, order, bps int) error {
	for i := 0; i < order; i++ {
		v, err := r.s(bps)
		if err != nil {
			return err
		}
		out[i] = v
	}
	prec, err := r.u(4)
	if err != nil || prec+1 == 16 {
		return fmt.Errorf("%w: flac lpc", avutil.ErrInvalidData)
	}
	prec++
	qlevel, err := r.s(5)
	if err != nil || qlevel < 0 {
		return fmt.Errorf("%w: flac lpc", avutil.ErrInvalidData)
	}
	coeff := make([]int32, order)
	for i := 0; i < order; i++ {
		v, err := r.s(int(prec))
		if err != nil {
			return err
		}
		coeff[i] = v
	}
	if err := decodeResidual(r, out, order); err != nil {
		return err
	}
	for i := order; i < len(out); i++ {
		var sum int64
		for j := 0; j < order; j++ {
			sum += int64(coeff[j]) * int64(out[i-1-j])
		}
		out[i] += int32(sum >> qlevel)
	}
	return nil
}

func (c *flacCodec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil || frame.NbSamples == 0 {
		return nil, nil
	}
	if frame.SampleFmt != avutil.SampleS16 {
		return nil, fmt.Errorf("%w: flac %s", avutil.ErrInvalidData, frame.SampleFmt)
	}
	channels := frame.Layout.Channels()
	if channels <= 0 || channels > 8 {
		return nil, fmt.Errorf("%w: flac channels", avutil.ErrRange)
	}
	if c.rate == 0 {
		c.rate = frame.SampleRate
		c.channels = channels
	}
	if frame.SampleRate != c.rate || channels != c.channels {
		return nil, fmt.Errorf("%w: flac parameter change", avutil.ErrInvalidData)
	}
	for i := 0; i < frame.NbSamples*channels; i++ {
		c.pcm = append(c.pcm, int32(int16(binary.LittleEndian.Uint16(frame.Data[0][i*2:]))))
	}
	return nil, nil
}

// Flush writes one FLAC file for the samples Encode stored.
func (c *flacCodec) Flush() ([]*avutil.Packet, error) {
	if len(c.pcm) == 0 || c.channels == 0 {
		return nil, nil
	}
	samples := c.pcm
	channels := c.channels
	rate := c.rate
	c.pcm = nil
	n := len(samples) / channels
	var body []byte
	const block = 4096
	frameNo := 0
	for off := 0; off < n; {
		count := block
		if off+count > n {
			count = n - off
		}
		chunk := samples[off*channels : (off+count)*channels]
		raw, err := encodeFLACFrame(chunk, count, channels, rate, 16, frameNo)
		if err != nil {
			return nil, err
		}
		body = append(body, raw...)
		off += count
		frameNo++
	}
	pcm := make([]byte, n*channels*2)
	for i, v := range samples[:n*channels] {
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(int16(v)))
	}
	info := streamInfoBytes(rate, channels, 16, n, md5PCM(pcm))
	out := append([]byte("fLaC"), 0x80, 0, 0, 34)
	out = append(out, info...)
	out = append(out, body...)
	tb := avutil.Q(1, int32(rate))
	return []*avutil.Packet{{
		Data: out, Duration: int64(n), TimeBase: tb, Key: true,
	}}, nil
}

func md5PCM(pcm []byte) []byte {
	sum := md5.Sum(pcm)
	return sum[:]
}

func streamInfoBytes(rate, channels, bps, total int, sum []byte) []byte {
	var w bitw
	w.u(16, 16)
	w.u(4096, 16)
	w.u(0, 24)
	w.u(0, 24)
	w.u(uint32(rate), 20)
	w.u(uint32(channels-1), 3)
	w.u(uint32(bps-1), 5)
	w.u(0, 4)
	w.u(uint32(total), 32)
	raw := w.bytes()
	if len(raw) < 18 {
		raw = append(raw, make([]byte, 18-len(raw))...)
	}
	raw = raw[:18]
	raw = append(raw, sum...)
	return raw
}

func encodeFLACFrame(interleaved []int32, block, channels, rate, bps, index int) ([]byte, error) {
	var w bitw
	w.u(0x7FFC, 15)
	w.u(0, 1)
	bs := uint32(7)
	switch block {
	case 192:
		bs = 1
	case 576:
		bs = 2
	case 1152:
		bs = 3
	case 2304:
		bs = 4
	case 4608:
		bs = 5
	case 256:
		bs = 8
	case 512:
		bs = 9
	case 1024:
		bs = 10
	case 2048:
		bs = 11
	case 4096:
		bs = 12
	case 8192:
		bs = 13
	case 16384:
		bs = 14
	case 32768:
		bs = 15
	}
	w.u(bs, 4)
	sr := uint32(0)
	for i, v := range flacRate {
		if v == rate {
			sr = uint32(i)
			break
		}
	}
	if sr == 0 {
		sr = 13
	}
	w.u(sr, 4)
	w.u(uint32(channels-1), 4)
	sampleCode := uint32(4)
	if bps == 8 {
		sampleCode = 1
	}
	w.u(sampleCode, 3)
	w.u(0, 1)
	w.utf8(uint64(index))
	if bs == 6 {
		w.u(uint32(block-1), 8)
	}
	if bs == 7 {
		w.u(uint32(block-1), 16)
	}
	if sr == 13 {
		w.u(uint32(rate), 16)
	}
	hdr := w.bytes()
	hdr = append(hdr, crc8ATM(hdr))
	var sub bitw
	for ch := 0; ch < channels; ch++ {
		plane := make([]int32, block)
		constant := true
		for i := 0; i < block; i++ {
			plane[i] = interleaved[i*channels+ch]
			if plane[i] != plane[0] {
				constant = false
			}
		}
		sub.u(0, 1)
		if constant {
			sub.u(0, 6)
			sub.u(0, 1)
			sub.s(plane[0], bps)
			continue
		}
		sub.u(1, 6)
		sub.u(0, 1)
		for i := 0; i < block; i++ {
			sub.s(plane[i], bps)
		}
	}
	payload := sub.bytes()
	frame := append(hdr, payload...)
	sum := crc16ANSI(frame)
	frame = append(frame, byte(sum>>8), byte(sum))
	return frame, nil
}

func (w *bitw) utf8(v uint64) {
	switch {
	case v < 0x80:
		w.u(uint32(v), 8)
	case v < 0x800:
		w.u(uint32(0xc0|(v>>6)), 8)
		w.u(uint32(0x80|(v&0x3f)), 8)
	default:
		w.u(uint32(0xe0|(v>>12)), 8)
		w.u(uint32(0x80|((v>>6)&0x3f)), 8)
		w.u(uint32(0x80|(v&0x3f)), 8)
	}
}
