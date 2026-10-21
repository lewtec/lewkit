package avcodec

import (
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// hevcCodec is a main-profile HEVC codec. Each coded tree block is its own
// IDR slice whose only coding unit is PCM, which the spec stores losslessly.
// libx265 is accepted as an encoder alias for this implementation.
type hevcCodec struct {
	par      Parameters
	codedW   int
	codedH   int
	visW     int
	visH     int
	cropR    int
	cropB    int
	addrBits int
	y        []byte
	u        []byte
	v        []byte
}

func newHEVC(par Parameters) *hevcCodec {
	return &hevcCodec{par: par, visW: par.Width, visH: par.Height}
}

func (c *hevcCodec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil || frame.Width <= 0 || frame.Height <= 0 {
		return nil, fmt.Errorf("%w: hevc frame", avutil.ErrInvalidData)
	}
	if frame.Format != avutil.PixYUV420P && frame.Format != avutil.PixYUVJ420P {
		return nil, fmt.Errorf("%w: hevc format %s", avutil.ErrUnsupported, frame.Format)
	}
	au, err := c.accessUnit(frame)
	if err != nil {
		return nil, err
	}
	pkt := &avutil.Packet{
		Data: au, Key: true, PTS: frame.PTS, DTS: frame.PTS,
		Duration: frame.Duration, TimeBase: frame.TimeBase,
	}
	if pkt.Duration == 0 {
		pkt.Duration = 1
	}
	return []*avutil.Packet{pkt}, nil
}

func (c *hevcCodec) Flush() ([]*avutil.Packet, error) { return nil, nil }

func (c *hevcCodec) Decode(pkt *avutil.Packet) ([]*avutil.Frame, error) {
	if pkt == nil || len(pkt.Data) == 0 {
		return nil, nil
	}
	c.y, c.u, c.v = nil, nil, nil
	for _, nal := range splitNALs(pkt.Data) {
		rbsp := unescapeNAL(nal)
		if len(rbsp) < 2 {
			continue
		}
		kind := int(rbsp[0]>>1) & 0x3f
		body := rbsp[2:]
		switch kind {
		case 32, 34: // VPS, PPS
		case 33:
			if err := c.parseSPS(body); err != nil {
				return nil, err
			}
		case 19, 20:
			if err := c.decodeSlice(body); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("%w: hevc nal %d", avutil.ErrUnsupported, kind)
		}
	}
	if c.visW <= 0 || c.y == nil {
		return nil, fmt.Errorf("%w: hevc access unit", avutil.ErrInvalidData)
	}
	frame, err := avutil.NewVideo(avutil.PixYUV420P, c.visW, c.visH)
	if err != nil {
		return nil, err
	}
	copyVisible(frame, c.y, c.u, c.v, c.codedW, c.codedH)
	frame.PTS, frame.Duration, frame.TimeBase = pkt.PTS, pkt.Duration, pkt.TimeBase
	return []*avutil.Frame{frame}, nil
}

func (c *hevcCodec) accessUnit(frame *avutil.Frame) ([]byte, error) {
	if frame.Width%2 != 0 || frame.Height%2 != 0 {
		return nil, fmt.Errorf("%w: hevc size %dx%d", avutil.ErrInvalidData, frame.Width, frame.Height)
	}
	codedW := (frame.Width + 15) &^ 15
	codedH := (frame.Height + 15) &^ 15
	ctbW, ctbH := codedW/16, codedH/16
	addrBits := hevcCeilLog2(ctbW * ctbH)
	var au []byte
	au = append(au, wrapHEVC(32, writeVPS())...)
	au = append(au, wrapHEVC(33, writeHEVCSPS(codedW, codedH, frame.Width, frame.Height))...)
	au = append(au, wrapHEVC(34, writeHEVCPPS())...)
	for addr := 0; addr < ctbW*ctbH; addr++ {
		body, err := writeHEVCSlice(frame, addr%ctbW, addr/ctbW, addr, addrBits)
		if err != nil {
			return nil, err
		}
		au = append(au, wrapHEVC(20, body)...)
	}
	return au, nil
}

func writeVPS() []byte {
	var w bitw
	w.u(0, 4) // vps id
	w.u(3, 2) // base layer internal and available
	w.u(0, 6) // max_layers_minus1
	w.u(0, 3) // max_sub_layers_minus1
	w.u(1, 1) // temporal nesting
	w.u(0xffff, 16)
	w.hevcProfile()
	w.u(93, 8) // level 3.1
	w.u(1, 1)  // sub-layer ordering present
	w.ue(0)
	w.ue(0)
	w.ue(0)
	w.u(0, 6) // max_layer_id
	w.ue(0)   // num_layer_sets_minus1
	w.u(0, 1) // timing
	w.u(0, 1) // extension
	w.trail()
	return w.bytes()
}

func writeHEVCSPS(codedW, codedH, visW, visH int) []byte {
	var w bitw
	w.u(0, 4) // vps id
	w.u(0, 3) // max_sub_layers_minus1
	w.u(1, 1) // temporal nesting
	w.hevcProfile()
	w.u(93, 8)
	w.ue(0) // sps id
	w.ue(1) // 4:2:0
	w.ue(codedW)
	w.ue(codedH)
	w.u(1, 1) // conformance window
	w.ue(0)
	w.ue((codedW - visW) / 2)
	w.ue(0)
	w.ue((codedH - visH) / 2)
	w.ue(0) // bit depth luma minus 8
	w.ue(0)
	w.ue(0)   // log2_max_poc_lsb_minus4
	w.u(1, 1) // sub-layer ordering
	w.ue(0)
	w.ue(0)
	w.ue(0)
	w.ue(0)   // log2_min_cb_size_minus3 = 8
	w.ue(1)   // diff -> CTB 16
	w.ue(0)   // log2_min_tb_size_minus2 = 4
	w.ue(2)   // max transform 16
	w.ue(0)   // max hierarchy inter
	w.ue(0)   // intra
	w.u(0, 1) // scaling list
	w.u(0, 1) // amp
	w.u(0, 1) // sao
	w.u(1, 1) // pcm
	w.u(7, 4) // pcm bit depth luma minus 1
	w.u(7, 4)
	w.ue(1)   // log2_min_pcm_cb_size_minus3 = 16
	w.ue(0)   // max pcm cb = 16
	w.u(1, 1) // pcm loop filter disabled
	w.ue(0)   // short term rps
	w.u(0, 1) // long term
	w.u(0, 1) // temporal mvp
	w.u(0, 1) // strong intra smoothing
	w.u(0, 1) // vui
	w.u(0, 1) // extension
	w.trail()
	return w.bytes()
}

func writeHEVCPPS() []byte {
	var w bitw
	w.ue(0)   // pps id
	w.ue(0)   // sps id
	w.u(0, 1) // dependent slices
	w.u(0, 1) // output flag
	w.u(0, 3) // extra slice header bits
	w.u(0, 1) // sign data hiding
	w.u(0, 1) // cabac init present
	w.ue(0)   // num ref l0
	w.ue(0)
	w.se(0)   // init qp
	w.u(0, 1) // constrained intra
	w.u(0, 1) // transform skip
	w.u(0, 1) // cu qp delta
	w.se(0)   // cb qp offset
	w.se(0)
	w.u(0, 1) // slice chroma qp offsets
	w.u(0, 1) // weighted pred
	w.u(0, 1) // weighted bipred
	w.u(0, 1) // transquant bypass
	w.u(0, 1) // tiles
	w.u(0, 1) // wpp
	w.u(0, 1) // loop filter across slices
	w.u(1, 1) // deblocking control present
	w.u(0, 1) // override
	w.u(1, 1) // deblocking disabled
	w.u(0, 1) // scaling list
	w.u(0, 1) // lists modification
	w.ue(0)   // log2_parallel_merge_level_minus2
	w.u(1, 1) // slice header extension present
	w.u(0, 1) // pps extension
	w.trail()
	return w.bytes()
}

// writeHEVCSlice emits one 16x16 PCM coding unit. The two CABAC bytes are the
// split_cu_flag=0 and pcm_flag=1 that FFmpeg's decoder reads for this SPS,
// followed by the raw samples and an end_of_slice_segment_flag.
func writeHEVCSlice(frame *avutil.Frame, ctbX, ctbY, addr, addrBits int) ([]byte, error) {
	pcm, err := hevcPCM(frame, ctbX*16, ctbY*16)
	if err != nil {
		return nil, err
	}
	for ext := 0; ext < 24; ext++ {
		var w bitw
		if addr == 0 {
			w.u(1, 1)
		} else {
			w.u(0, 1)
		}
		w.u(0, 1) // no_output_of_prior_pics
		w.ue(0)   // pps
		if addr != 0 {
			w.u(uint32(addr), addrBits)
		}
		w.ue(2) // I slice
		w.se(0) // qp delta
		w.ue(ext)
		for i := 0; i < ext; i++ {
			w.u(0, 8)
		}
		w.u(1, 1) // byte_alignment bit consumed by the CABAC init
		for w.n != 0 {
			w.u(0, 1)
		}
		body := w.bytes()
		// FFmpeg's RBSP buffer is aligned. An even distance from the NAL
		// header keeps the CABAC engine on the even-address init path.
		if len(body)%2 != 0 {
			continue
		}
		body = append(body, 134, 0)
		body = append(body, pcm...)
		body = append(body, 254, 0)
		return body, nil
	}
	return nil, fmt.Errorf("%w: hevc slice header", avutil.ErrInvalidData)
}

func hevcPCM(frame *avutil.Frame, x0, y0 int) ([]byte, error) {
	ys, cs := frame.Linesize[0], frame.Linesize[1]
	y, u, v := frame.Data[0], frame.Data[1], frame.Data[2]
	ch := (frame.Height + 1) / 2
	cw := (frame.Width + 1) / 2
	if ys < 1 || cs < 1 || len(y) < ys*frame.Height || len(u) < cs*ch || len(v) < cs*ch {
		return nil, fmt.Errorf("%w: hevc planes", avutil.ErrInvalidData)
	}
	out := make([]byte, 16*16+8*8+8*8)
	for row := 0; row < 16; row++ {
		dst := out[row*16 : row*16+16]
		if x0+16 <= frame.Width && y0+row < frame.Height {
			hevcCopy16(dst, y[(y0+row)*ys+x0:])
			continue
		}
		for col := 0; col < 16; col++ {
			dst[col] = sampleAt(y, ys, frame.Width, frame.Height, x0+col, y0+row)
		}
	}
	base := 16 * 16
	for _, plane := range [][]byte{u, v} {
		for row := 0; row < 8; row++ {
			for col := 0; col < 8; col++ {
				out[base] = sampleAt(plane, cs, cw, ch, x0/2+col, y0/2+row)
				base++
			}
		}
	}
	return out, nil
}

func (c *hevcCodec) parseSPS(body []byte) error {
	r := bitr{b: body}
	if _, err := r.u(4); err != nil {
		return err
	}
	if _, err := r.u(3); err != nil {
		return err
	}
	if _, err := r.u(1); err != nil {
		return err
	}
	if err := r.skipProfile(); err != nil {
		return err
	}
	if _, err := r.u(8); err != nil {
		return err
	}
	if _, err := r.ue(); err != nil {
		return err
	}
	chroma, err := r.ue()
	if err != nil {
		return err
	}
	if chroma != 1 {
		return fmt.Errorf("%w: hevc chroma", avutil.ErrUnsupported)
	}
	codedW, err := r.ue()
	if err != nil {
		return err
	}
	codedH, err := r.ue()
	if err != nil {
		return err
	}
	flag, err := r.u(1)
	if err != nil {
		return err
	}
	visW, visH := codedW, codedH
	if flag == 1 {
		left, err := r.ue()
		if err != nil {
			return err
		}
		right, err := r.ue()
		if err != nil {
			return err
		}
		top, err := r.ue()
		if err != nil {
			return err
		}
		bottom, err := r.ue()
		if err != nil {
			return err
		}
		visW -= (left + right) * 2
		visH -= (top + bottom) * 2
	}
	if visW <= 0 || visH <= 0 || codedW%16 != 0 || codedH%16 != 0 {
		return fmt.Errorf("%w: hevc size", avutil.ErrInvalidData)
	}
	c.codedW, c.codedH, c.visW, c.visH = codedW, codedH, visW, visH
	c.addrBits = hevcCeilLog2((codedW / 16) * (codedH / 16))
	c.y = make([]byte, codedW*codedH)
	c.u = make([]byte, codedW*codedH/4)
	c.v = make([]byte, codedW*codedH/4)
	return nil
}

func (c *hevcCodec) decodeSlice(body []byte) error {
	if c.y == nil {
		return fmt.Errorf("%w: hevc slice without sps", avutil.ErrInvalidData)
	}
	r := bitr{b: body}
	first, err := r.u(1)
	if err != nil {
		return err
	}
	if _, err = r.u(1); err != nil {
		return err
	}
	if _, err = r.ue(); err != nil {
		return err
	}
	addr := 0
	if first == 0 {
		v, err := r.u(c.addrBits)
		if err != nil {
			return err
		}
		addr = int(v)
	}
	kind, err := r.ue()
	if err != nil {
		return err
	}
	if kind != 2 {
		return fmt.Errorf("%w: hevc slice type", avutil.ErrUnsupported)
	}
	if _, err = r.se(); err != nil {
		return err
	}
	ext, err := r.ue()
	if err != nil {
		return err
	}
	for i := 0; i < ext; i++ {
		if _, err = r.u(8); err != nil {
			return err
		}
	}
	if _, err = r.u(1); err != nil {
		return err
	}
	r.align()
	off := r.i / 8
	const pcmN = 16*16 + 8*8 + 8*8
	if off < 0 || off+2+pcmN > len(body) {
		return fmt.Errorf("%w: hevc pcm", avutil.ErrInvalidData)
	}
	pcm := body[off+2 : off+2+pcmN]
	ctbW := c.codedW / 16
	if ctbW == 0 || addr < 0 || addr >= ctbW*(c.codedH/16) {
		return fmt.Errorf("%w: hevc slice address", avutil.ErrInvalidData)
	}
	x0 := (addr % ctbW) * 16
	y0 := (addr / ctbW) * 16
	for row := 0; row < 16; row++ {
		copy(c.y[(y0+row)*c.codedW+x0:(y0+row)*c.codedW+x0+16], pcm[row*16:(row+1)*16])
	}
	cu := c.codedW / 2
	cx, cy := x0/2, y0/2
	cb := pcm[256:320]
	cr := pcm[320:384]
	for row := 0; row < 8; row++ {
		copy(c.u[(cy+row)*cu+cx:(cy+row)*cu+cx+8], cb[row*8:(row+1)*8])
		copy(c.v[(cy+row)*cu+cx:(cy+row)*cu+cx+8], cr[row*8:(row+1)*8])
	}
	return nil
}

func copyVisible(frame *avutil.Frame, y, u, v []byte, codedW, codedH int) {
	for row := 0; row < frame.Height && row < codedH; row++ {
		n := frame.Width
		if n > codedW {
			n = codedW
		}
		copy(frame.Data[0][row*frame.Linesize[0]:row*frame.Linesize[0]+n], y[row*codedW:row*codedW+n])
	}
	cw, ch := frame.Width/2, frame.Height/2
	cu := codedW / 2
	for row := 0; row < ch; row++ {
		copy(frame.Data[1][row*frame.Linesize[1]:row*frame.Linesize[1]+cw], u[row*cu:row*cu+cw])
		copy(frame.Data[2][row*frame.Linesize[2]:row*frame.Linesize[2]+cw], v[row*cu:row*cu+cw])
	}
}

func (w *bitw) hevcProfile() {
	w.u(0, 2)
	w.u(0, 1)
	w.u(1, 5) // Main
	w.u(0, 1) // compatibility[0]
	w.u(1, 1) // compatibility[1]
	w.u(0, 30)
	w.u(1, 1) // progressive
	w.u(0, 1)
	w.u(1, 1) // non-packed
	w.u(1, 1) // frame only
	w.u(0, 43)
	w.u(0, 1) // general_inbld_flag
}

func (r *bitr) skipProfile() error {
	// profile_space, tier, profile_idc, 32 compatibility flags, 4 source flags,
	// 43 reserved bits, and general_inbld_flag. bitr.u accepts at most 32 bits.
	for _, n := range []int{32, 12, 32, 12} {
		if _, err := r.u(n); err != nil {
			return err
		}
	}
	return nil
}

func hevcCeilLog2(n int) int {
	if n <= 1 {
		return 0
	}
	bits := 0
	for v := n - 1; v > 0; v >>= 1 {
		bits++
	}
	return bits
}

// wrapHEVC writes an annex-B NAL. A forced emulation-prevention sequence sends
// the NAL through FFmpeg's aligned RBSP buffer, and an even raw length keeps
// the following NAL on that same alignment.
func wrapHEVC(kind int, rbsp []byte) []byte {
	raw := []byte{byte(kind << 1), 1}
	raw = append(raw, escapeRBSP(rbsp)...)
	if !bytesHasEP(raw) {
		raw = append(raw, 0, 0, 3)
	}
	if len(raw)%2 != 0 {
		raw = append(raw, 0)
	}
	out := []byte{0, 0, 0, 1}
	return append(out, raw...)
}

func escapeRBSP(rbsp []byte) []byte {
	out := make([]byte, 0, len(rbsp)+8)
	zeros := 0
	for _, b := range rbsp {
		if zeros == 2 && b <= 3 {
			out = append(out, 3)
			zeros = 0
		}
		out = append(out, b)
		if b == 0 {
			zeros++
		} else {
			zeros = 0
		}
	}
	return out
}

func bytesHasEP(b []byte) bool {
	for i := 0; i+2 < len(b); i++ {
		if b[i] == 0 && b[i+1] == 0 && b[i+2] <= 3 {
			return true
		}
	}
	return false
}
