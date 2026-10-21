package avcodec

import (
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// h264Codec is a constrained-baseline H.264 codec. Each frame is an IDR
// access unit of I_PCM macroblocks, which the spec stores losslessly.
// libx264 is accepted as an encoder alias for this implementation.
type h264Codec struct {
	par    Parameters
	width  int
	height int
	cropR  int
	cropB  int
	idr    int
	sps    []byte
	pps    []byte
}

func newH264(par Parameters) *h264Codec {
	return &h264Codec{par: par, width: par.Width, height: par.Height}
}

func (c *h264Codec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil || frame.Width <= 0 || frame.Height <= 0 {
		return nil, fmt.Errorf("%w: h264 frame", avutil.ErrInvalidData)
	}
	if frame.Format != avutil.PixYUV420P && frame.Format != avutil.PixYUVJ420P {
		return nil, fmt.Errorf("%w: h264 format %s", avutil.ErrUnsupported, frame.Format)
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

func (c *h264Codec) Flush() ([]*avutil.Packet, error) { return nil, nil }

func (c *h264Codec) Decode(pkt *avutil.Packet) ([]*avutil.Frame, error) {
	if pkt == nil || len(pkt.Data) == 0 {
		return nil, nil
	}
	nals := splitNALs(pkt.Data)
	var frame *avutil.Frame
	for _, nal := range nals {
		if len(nal) == 0 {
			continue
		}
		rbsp := unescapeNAL(nal)
		if len(rbsp) == 0 {
			continue
		}
		kind := rbsp[0] & 0x1f
		body := rbsp[1:]
		switch kind {
		case 7:
			w, h, cr, cb, err := parseSPS(body)
			if err != nil {
				return nil, err
			}
			c.width, c.height, c.cropR, c.cropB = w, h, cr, cb
		case 8:
			c.pps = append([]byte(nil), body...)
		case 5, 1:
			pic, err := c.decodeSlice(body)
			if err != nil {
				return nil, err
			}
			pic.PTS, pic.Duration, pic.TimeBase = pkt.PTS, pkt.Duration, pkt.TimeBase
			frame = pic
		}
	}
	if frame == nil {
		return nil, fmt.Errorf("%w: h264 access unit", avutil.ErrInvalidData)
	}
	return []*avutil.Frame{frame}, nil
}

func (c *h264Codec) accessUnit(frame *avutil.Frame) ([]byte, error) {
	mbW := (frame.Width + 15) / 16
	mbH := (frame.Height + 15) / 16
	codedW, codedH := mbW*16, mbH*16
	cropR := (codedW - frame.Width) / 2
	cropB := (codedH - frame.Height) / 2
	if cropR*2 != codedW-frame.Width || cropB*2 != codedH-frame.Height {
		return nil, fmt.Errorf("%w: h264 size %dx%d", avutil.ErrInvalidData, frame.Width, frame.Height)
	}
	sps := writeSPS(mbW, mbH, cropR, cropB)
	pps := writePPS()
	slice, err := writeSlice(frame, mbW, mbH, c.idr)
	if err != nil {
		return nil, err
	}
	c.idr ^= 1
	var au []byte
	au = append(au, wrapNAL(3, 7, sps)...)
	au = append(au, wrapNAL(3, 8, pps)...)
	au = append(au, wrapNAL(3, 5, slice)...)
	return au, nil
}

func writeSPS(mbW, mbH, cropR, cropB int) []byte {
	var w bitw
	w.u(66, 8)
	w.u(0xC0, 8) // constraint_set0 and constraint_set1
	w.u(31, 8)   // level 3.1
	w.ue(0)      // sps id
	w.ue(0)      // log2_max_frame_num_minus4
	w.ue(0)      // pic_order_cnt_type
	w.ue(0)      // log2_max_pic_order_cnt_lsb_minus4
	w.ue(1)      // max_num_ref_frames
	w.u(0, 1)    // gaps
	w.ue(mbW - 1)
	w.ue(mbH - 1)
	w.u(1, 1) // frame_mbs_only
	w.u(1, 1) // direct_8x8_inference
	if cropR != 0 || cropB != 0 {
		w.u(1, 1)
		w.ue(0)
		w.ue(cropR)
		w.ue(0)
		w.ue(cropB)
	} else {
		w.u(0, 1)
	}
	w.u(0, 1) // vui
	w.trail()
	return w.bytes()
}

func writePPS() []byte {
	var w bitw
	w.ue(0)
	w.ue(0)
	w.u(0, 1) // CAVLC
	w.u(0, 1) // bottom field
	w.ue(0)   // slice groups
	w.ue(0)   // ref l0
	w.ue(0)   // ref l1
	w.u(0, 1) // weighted pred
	w.u(0, 2) // weighted bipred
	w.se(0)   // init qp
	w.se(0)
	w.se(0)   // chroma qp offset
	w.u(1, 1) // deblocking control
	w.u(0, 1) // constrained intra
	w.u(0, 1) // redundant
	w.trail()
	return w.bytes()
}

func writeSlice(frame *avutil.Frame, mbW, mbH, idr int) ([]byte, error) {
	var w bitw
	w.ue(0)   // first mb
	w.ue(7)   // I slice, all intra
	w.ue(0)   // pps
	w.u(0, 4) // frame_num
	w.ue(idr)
	w.u(0, 4) // poc
	w.u(0, 1) // no_output_of_prior
	w.u(0, 1) // long_term
	w.se(0)   // qp delta
	w.ue(1)   // deblocking off
	ys, cs := frame.Linesize[0], frame.Linesize[1]
	y, u, v := frame.Data[0], frame.Data[1], frame.Data[2]
	if ys < 16 || cs < 8 || len(y) < ys*frame.Height || len(u) < cs*((frame.Height+1)/2) || len(v) < cs*((frame.Height+1)/2) {
		return nil, fmt.Errorf("%w: h264 planes", avutil.ErrInvalidData)
	}
	for my := 0; my < mbH; my++ {
		for mx := 0; mx < mbW; mx++ {
			w.ue(25) // I_PCM
			w.align0()
			for row := 0; row < 16; row++ {
				for col := 0; col < 16; col++ {
					w.u(uint32(sampleAt(y, ys, frame.Width, frame.Height, mx*16+col, my*16+row)), 8)
				}
			}
			for _, plane := range [][]byte{u, v} {
				for row := 0; row < 8; row++ {
					for col := 0; col < 8; col++ {
						w.u(uint32(sampleAt(plane, cs, (frame.Width+1)/2, (frame.Height+1)/2, mx*8+col, my*8+row)), 8)
					}
				}
			}
		}
	}
	w.trail()
	return w.bytes(), nil
}

func sampleAt(plane []byte, stride, width, height, x, y int) byte {
	if x >= width {
		x = width - 1
	}
	if y >= height {
		y = height - 1
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	i := y*stride + x
	if i < 0 || i >= len(plane) {
		return 0
	}
	return plane[i]
}

func (c *h264Codec) decodeSlice(rbsp []byte) (*avutil.Frame, error) {
	r := bitr{b: rbsp}
	if _, err := r.ue(); err != nil {
		return nil, err
	}
	if _, err := r.ue(); err != nil { // slice type
		return nil, err
	}
	if _, err := r.ue(); err != nil {
		return nil, err
	}
	if _, err := r.u(4); err != nil {
		return nil, err
	}
	if _, err := r.ue(); err != nil { // idr pic id, also present on non-IDR if we only emit IDR
		return nil, err
	}
	if _, err := r.u(4); err != nil {
		return nil, err
	}
	if _, err := r.u(1); err != nil {
		return nil, err
	}
	if _, err := r.u(1); err != nil {
		return nil, err
	}
	if _, err := r.se(); err != nil {
		return nil, err
	}
	dbc, err := r.ue()
	if err != nil {
		return nil, err
	}
	if dbc != 1 {
		if _, err = r.se(); err != nil {
			return nil, err
		}
		if _, err = r.se(); err != nil {
			return nil, err
		}
	}
	if c.width <= 0 || c.height <= 0 {
		if c.par.Width > 0 && c.par.Height > 0 {
			c.width, c.height = c.par.Width, c.par.Height
		} else {
			return nil, fmt.Errorf("%w: h264 sps", avutil.ErrInvalidData)
		}
	}
	visW := c.width - c.cropR*2
	visH := c.height - c.cropB*2
	if visW <= 0 || visH <= 0 {
		visW, visH = c.width, c.height
	}
	frame, err := avutil.NewVideo(avutil.PixYUV420P, visW, visH)
	if err != nil {
		return nil, err
	}
	mbW := (c.width + 15) / 16
	mbH := (c.height + 15) / 16
	for my := 0; my < mbH; my++ {
		for mx := 0; mx < mbW; mx++ {
			mb, err := r.ue()
			if err != nil {
				return nil, err
			}
			if mb != 25 {
				return nil, fmt.Errorf("%w: h264 mb_type %d", avutil.ErrUnsupported, mb)
			}
			r.align()
			for row := 0; row < 16; row++ {
				for col := 0; col < 16; col++ {
					s, err := r.u(8)
					if err != nil {
						return nil, err
					}
					x, y := mx*16+col, my*16+row
					if x < visW && y < visH {
						frame.Data[0][y*frame.Linesize[0]+x] = byte(s)
					}
				}
			}
			for pi := 1; pi <= 2; pi++ {
				for row := 0; row < 8; row++ {
					for col := 0; col < 8; col++ {
						s, err := r.u(8)
						if err != nil {
							return nil, err
						}
						x, y := mx*8+col, my*8+row
						cw, ch := (visW+1)/2, (visH+1)/2
						if x < cw && y < ch {
							frame.Data[pi][y*frame.Linesize[pi]+x] = byte(s)
						}
					}
				}
			}
		}
	}
	frame.Key = true
	return frame, nil
}

func parseSPS(rbsp []byte) (width, height, cropR, cropB int, err error) {
	r := bitr{b: rbsp}
	profile, err := r.u(8)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	if _, err = r.u(8); err != nil {
		return 0, 0, 0, 0, err
	}
	if _, err = r.u(8); err != nil {
		return 0, 0, 0, 0, err
	}
	if _, err = r.ue(); err != nil {
		return 0, 0, 0, 0, err
	}
	if profile == 100 || profile == 110 || profile == 122 || profile == 244 || profile == 44 || profile == 83 || profile == 86 || profile == 118 || profile == 128 || profile == 138 || profile == 144 {
		chroma, err := r.ue()
		if err != nil {
			return 0, 0, 0, 0, err
		}
		if chroma == 3 {
			if _, err = r.u(1); err != nil {
				return 0, 0, 0, 0, err
			}
		}
		if _, err = r.ue(); err != nil {
			return 0, 0, 0, 0, err
		}
		if _, err = r.ue(); err != nil {
			return 0, 0, 0, 0, err
		}
		if _, err = r.u(1); err != nil {
			return 0, 0, 0, 0, err
		}
		scale, err := r.u(1)
		if err != nil {
			return 0, 0, 0, 0, err
		}
		if scale == 1 {
			return 0, 0, 0, 0, fmt.Errorf("%w: h264 scaling list", avutil.ErrUnsupported)
		}
	}
	if _, err = r.ue(); err != nil {
		return 0, 0, 0, 0, err
	}
	poc, err := r.ue()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	if poc == 0 {
		if _, err = r.ue(); err != nil {
			return 0, 0, 0, 0, err
		}
	} else if poc == 1 {
		return 0, 0, 0, 0, fmt.Errorf("%w: h264 poc type", avutil.ErrUnsupported)
	}
	if _, err = r.ue(); err != nil {
		return 0, 0, 0, 0, err
	}
	if _, err = r.u(1); err != nil {
		return 0, 0, 0, 0, err
	}
	mbW, err := r.ue()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	mbH, err := r.ue()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	ff, err := r.u(1)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	if ff == 0 {
		if _, err = r.u(1); err != nil {
			return 0, 0, 0, 0, err
		}
	}
	if _, err = r.u(1); err != nil {
		return 0, 0, 0, 0, err
	}
	crop, err := r.u(1)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	width = (mbW + 1) * 16
	height = (mbH + 1) * 16
	if ff == 0 {
		height *= 2
	}
	if crop == 1 {
		if _, err = r.ue(); err != nil {
			return 0, 0, 0, 0, err
		}
		cr, err := r.ue()
		if err != nil {
			return 0, 0, 0, 0, err
		}
		if _, err = r.ue(); err != nil {
			return 0, 0, 0, 0, err
		}
		cb, err := r.ue()
		if err != nil {
			return 0, 0, 0, 0, err
		}
		cropR, cropB = cr, cb
	}
	return width, height, cropR, cropB, nil
}

func (w *bitw) ue(v int) {
	x := uint32(v + 1)
	zeros := 0
	for t := x; t > 1; t >>= 1 {
		zeros++
	}
	w.u(0, zeros)
	w.u(x, zeros+1)
}

func (w *bitw) se(v int) {
	if v <= 0 {
		w.ue(-2 * v)
	} else {
		w.ue(2*v - 1)
	}
}

func (w *bitw) align0() {
	for w.n != 0 {
		w.u(0, 1)
	}
}

func (w *bitw) trail() {
	w.u(1, 1)
	for w.n != 0 {
		w.u(0, 1)
	}
}

func (r *bitr) ue() (int, error) {
	zeros := 0
	for {
		b, err := r.u(1)
		if err != nil {
			return 0, err
		}
		if b == 1 {
			break
		}
		zeros++
		if zeros > 31 {
			return 0, fmt.Errorf("%w: h264 expgolomb", avutil.ErrInvalidData)
		}
	}
	if zeros == 0 {
		return 0, nil
	}
	rest, err := r.u(zeros)
	if err != nil {
		return 0, err
	}
	return int(rest) + (1 << zeros) - 1, nil
}

func (r *bitr) se() (int, error) {
	v, err := r.ue()
	if err != nil {
		return 0, err
	}
	if v%2 == 0 {
		return -v / 2, nil
	}
	return (v + 1) / 2, nil
}

func (r *bitr) align() {
	if r.i%8 != 0 {
		r.i += 8 - r.i%8
	}
}

func wrapNAL(nri, kind int, rbsp []byte) []byte {
	out := []byte{0, 0, 0, 1, byte(nri<<5 | kind)}
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

func unescapeNAL(nal []byte) []byte {
	out := make([]byte, 0, len(nal))
	for i := 0; i < len(nal); i++ {
		if i+2 < len(nal) && nal[i] == 0 && nal[i+1] == 0 && nal[i+2] == 3 {
			out = append(out, 0, 0)
			i += 2
			continue
		}
		out = append(out, nal[i])
	}
	return out
}

func splitNALs(b []byte) [][]byte {
	var starts []int
	for i := 0; i+3 < len(b); i++ {
		if b[i] == 0 && b[i+1] == 0 && b[i+2] == 1 {
			starts = append(starts, i+3)
			i += 2
			continue
		}
		if i+4 < len(b) && b[i] == 0 && b[i+1] == 0 && b[i+2] == 0 && b[i+3] == 1 {
			starts = append(starts, i+4)
			i += 3
		}
	}
	var nals [][]byte
	for i, s := range starts {
		e := len(b)
		if i+1 < len(starts) {
			e = starts[i+1]
			for e > s && b[e-1] == 0 {
				e--
			}
		}
		if s < e {
			nals = append(nals, b[s:e])
		}
	}
	return nals
}
