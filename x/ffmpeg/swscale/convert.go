package swscale

import (
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

func (c *Context) yuvToPacked(dst, src *avutil.Frame) error {
	tables := c.tablesFor(src.Format)
	for y := 0; y < src.Height; y++ {
		for x := 0; x < src.Width; x++ {
			yy, u, v := sampleYUV(src, x, y)
			r, g, b := tables.RGB(yy, u, v)
			putRGB(dst, x, y, r, g, b, 255)
		}
	}
	return nil
}

func sampleYUV(src *avutil.Frame, x, y int) (byte, byte, byte) {
	switch src.Format {
	case avutil.PixYUYV422:
		row := src.PlaneRow(0, y, src.Width*2)
		base := (x &^ 1) * 2
		if base+3 >= len(row) {
			return 0, 128, 128
		}
		return row[base], row[base+1], row[base+3]
	case avutil.PixUYVY422:
		row := src.PlaneRow(0, y, src.Width*2)
		base := (x &^ 1) * 2
		if base+3 >= len(row) {
			return 0, 128, 128
		}
		return row[base+1], row[base], row[base+2]
	case avutil.PixNV12, avutil.PixNV21:
		yrow := src.PlaneRow(0, y, src.Width)
		crow := src.PlaneRow(1, y>>avutil.DescMust(src.Format).ChromaH, avutil.ChromaWidth(src.Format, src.Width)*2)
		if x >= len(yrow) || len(crow) == 0 {
			return 0, 128, 128
		}
		cx := (x >> 1) * 2
		if cx+1 >= len(crow) {
			return yrow[x], 128, 128
		}
		u, v := crow[cx], crow[cx+1]
		if src.Format == avutil.PixNV21 {
			u, v = v, u
		}
		return yrow[x], u, v
	default:
		yrow := src.PlaneRow(0, y, src.Width)
		cw := avutil.ChromaWidth(src.Format, src.Width)
		chShift := 0
		if item, ok := avutil.Desc(src.Format); ok {
			chShift = item.ChromaH
		}
		crowU := src.PlaneRow(1, y>>chShift, cw)
		crowV := src.PlaneRow(2, y>>chShift, cw)
		if x >= len(yrow) {
			return 0, 128, 128
		}
		cx := x
		if item, ok := avutil.Desc(src.Format); ok {
			cx = x >> item.ChromaW
		}
		u, v := byte(128), byte(128)
		if cx < len(crowU) {
			u = crowU[cx]
		}
		if cx < len(crowV) {
			v = crowV[cx]
		}
		return yrow[x], u, v
	}
}

func shufflePacked(dst, src *avutil.Frame) error {
	for y := 0; y < src.Height; y++ {
		for x := 0; x < src.Width; x++ {
			r, g, b, a := getRGB(src, x, y)
			putRGB(dst, x, y, r, g, b, a)
		}
	}
	return nil
}

func unpack(dst, src *avutil.Frame, c *Context) error {
	if dst.Format == avutil.PixGBRP {
		for y := 0; y < src.Height; y++ {
			g := dst.PlaneRow(0, y, dst.Width)
			b := dst.PlaneRow(1, y, dst.Width)
			r := dst.PlaneRow(2, y, dst.Width)
			for x := 0; x < src.Width; x++ {
				rr, gg, bb, _ := getRGB(src, x, y)
				if x < len(g) {
					g[x], b[x], r[x] = gg, bb, rr
				}
			}
		}
		return nil
	}
	tables := c.tablesFor(dst.Format)
	_ = tables
	for y := 0; y < src.Height; y++ {
		for x := 0; x < src.Width; x++ {
			var yy, u, v byte
			if isYUV8(src.Format) || src.Format == avutil.PixGray {
				yy, u, v = sampleYUV(src, x, y)
			} else {
				r, g, b, _ := getRGB(src, x, y)
				yy, u, v = rgbToYUV(r, g, b, avutil.IsFullRange(dst.Format))
			}
			putYUV444(dst, x, y, yy, u, v)
		}
	}
	return nil
}

func pack(dst, src *avutil.Frame, c *Context) error {
	switch dst.Format {
	case avutil.PixGBRP:
		copyPlanes(dst, src)
		return nil
	case avutil.PixYUV444P, avutil.PixYUVJ444P:
		copyPlanes(dst, src)
		return nil
	case avutil.PixGray:
		for y := 0; y < dst.Height; y++ {
			row := dst.PlaneRow(0, y, dst.Width)
			for x := 0; x < dst.Width && x < len(row); x++ {
				yy, _, _ := sampleYUV(src, x, y)
				row[x] = yy
			}
		}
		return nil
	}
	if isYUV8(dst.Format) {
		for y := 0; y < dst.Height; y++ {
			for x := 0; x < dst.Width; x++ {
				yy, u, v := sampleYUV(src, x, y)
				putYUV(dst, x, y, yy, u, v)
			}
		}
		return nil
	}
	if isPackedRGB(dst.Format) {
		tables := c.tablesFor(src.Format)
		for y := 0; y < dst.Height; y++ {
			for x := 0; x < dst.Width; x++ {
				yy, u, v := sampleYUV(src, x, y)
				r, g, b := tables.RGB(yy, u, v)
				if isRGBFamily(src.Format) && src.Format == avutil.PixGBRP {
					gplane := src.PlaneRow(0, y, src.Width)
					bplane := src.PlaneRow(1, y, src.Width)
					rplane := src.PlaneRow(2, y, src.Width)
					if x < len(gplane) {
						r, g, b = rplane[x], gplane[x], bplane[x]
					}
				}
				putRGB(dst, x, y, r, g, b, 255)
			}
		}
		return nil
	}
	return fmt.Errorf("%w: pack %s", avutil.ErrUnsupported, dst.Format)
}

func putYUV444(dst *avutil.Frame, x, y int, yy, u, v byte) {
	rowY := dst.PlaneRow(0, y, dst.Width)
	rowU := dst.PlaneRow(1, y, dst.Width)
	rowV := dst.PlaneRow(2, y, dst.Width)
	if x < len(rowY) {
		rowY[x] = yy
	}
	if x < len(rowU) {
		rowU[x] = u
	}
	if x < len(rowV) {
		rowV[x] = v
	}
}

func putYUV(dst *avutil.Frame, x, y int, yy, u, v byte) {
	switch dst.Format {
	case avutil.PixYUYV422:
		row := dst.PlaneRow(0, y, dst.Width*2)
		base := (x &^ 1) * 2
		if base+3 < len(row) {
			row[base] = yy
			if x&1 == 0 {
				row[base+1] = u
				row[base+3] = v
			}
		}
	case avutil.PixUYVY422:
		row := dst.PlaneRow(0, y, dst.Width*2)
		base := (x &^ 1) * 2
		if base+3 < len(row) {
			row[base+1] = yy
			if x&1 == 0 {
				row[base] = u
				row[base+2] = v
			}
		}
	case avutil.PixNV12, avutil.PixNV21:
		row := dst.PlaneRow(0, y, dst.Width)
		if x < len(row) {
			row[x] = yy
		}
		if x&1 == 0 && y&1 == 0 {
			crow := dst.PlaneRow(1, y>>1, avutil.ChromaWidth(dst.Format, dst.Width)*2)
			cx := x
			if cx+1 < len(crow) {
				if dst.Format == avutil.PixNV21 {
					crow[cx], crow[cx+1] = v, u
				} else {
					crow[cx], crow[cx+1] = u, v
				}
			}
		}
	default:
		row := dst.PlaneRow(0, y, dst.Width)
		if x < len(row) {
			row[x] = yy
		}
		item, ok := avutil.Desc(dst.Format)
		if !ok {
			return
		}
		if x&(1<<item.ChromaW-1) == 0 && y&(1<<item.ChromaH-1) == 0 || (item.ChromaW == 0 && item.ChromaH == 0) {
			cx := x >> item.ChromaW
			cy := y >> item.ChromaH
			cw := avutil.ChromaWidth(dst.Format, dst.Width)
			urow := dst.PlaneRow(1, cy, cw)
			vrow := dst.PlaneRow(2, cy, cw)
			if cx < len(urow) {
				urow[cx] = u
			}
			if cx < len(vrow) {
				vrow[cx] = v
			}
		}
	}
}

func rgbToYUV(r, g, b byte, full bool) (byte, byte, byte) {
	rr, gg, bb := int(r), int(g), int(b)
	if full {
		yy := (77*rr + 150*gg + 29*bb + 128) >> 8
		u := ((-43*rr - 84*gg + 127*bb + 128) >> 8) + 128
		v := ((127*rr - 106*gg - 21*bb + 128) >> 8) + 128
		return avutil.ClipUint8(yy), avutil.ClipUint8(u), avutil.ClipUint8(v)
	}
	yy := ((66*rr + 129*gg + 25*bb + 128) >> 8) + 16
	u := ((-38*rr - 74*gg + 112*bb + 128) >> 8) + 128
	v := ((112*rr - 94*gg - 18*bb + 128) >> 8) + 128
	return avutil.ClipUint8(yy), avutil.ClipUint8(u), avutil.ClipUint8(v)
}

func getRGB(src *avutil.Frame, x, y int) (byte, byte, byte, byte) {
	switch src.Format {
	case avutil.PixGBRP:
		g := src.PlaneRow(0, y, src.Width)
		b := src.PlaneRow(1, y, src.Width)
		r := src.PlaneRow(2, y, src.Width)
		if x < len(g) && x < len(b) && x < len(r) {
			return r[x], g[x], b[x], 255
		}
	case avutil.PixGray, avutil.PixPAL8:
		row := src.PlaneRow(0, y, src.Width)
		if x < len(row) {
			return row[x], row[x], row[x], 255
		}
	case avutil.PixRGB565LE:
		return unpack565(src, x, y)
	case avutil.PixRGB555LE:
		return unpack555(src, x, y)
	default:
		row := src.PlaneRow(0, y, src.Width*4)
		if isYUV8(src.Format) {
			yy, u, v := sampleYUV(src, x, y)
			r, g, b := yuvToRGBTables(avutil.IsFullRange(src.Format)).RGB(yy, u, v)
			return r, g, b, 255
		}
		packed := 3
		if item, ok := avutil.Desc(src.Format); ok && item.Packed > 0 {
			packed = item.Packed
		}
		base := x * packed
		if base+2 >= len(row) {
			return 0, 0, 0, 255
		}
		switch src.Format {
		case avutil.PixRGB24:
			return row[base], row[base+1], row[base+2], 255
		case avutil.PixBGR24:
			return row[base+2], row[base+1], row[base], 255
		case avutil.PixRGBA:
			return row[base], row[base+1], row[base+2], row[base+3]
		case avutil.PixBGRA:
			return row[base+2], row[base+1], row[base], row[base+3]
		case avutil.PixARGB:
			return row[base+1], row[base+2], row[base+3], row[base]
		case avutil.PixABGR:
			return row[base+3], row[base+2], row[base+1], row[base]
		case avutil.PixRGB0:
			return row[base], row[base+1], row[base+2], 255
		case avutil.PixBGR0:
			return row[base+2], row[base+1], row[base], 255
		case avutil.Pix0RGB:
			return row[base+1], row[base+2], row[base+3], 255
		case avutil.Pix0BGR:
			return row[base+3], row[base+2], row[base+1], 255
		}
	}
	return 0, 0, 0, 255
}

func putRGB(dst *avutil.Frame, x, y int, r, g, b, a byte) {
	switch dst.Format {
	case avutil.PixRGB565LE:
		pack565(dst, x, y, r, g, b)
		return
	case avutil.PixRGB555LE:
		pack555(dst, x, y, r, g, b)
		return
	}
	packed := 3
	if item, ok := avutil.Desc(dst.Format); ok && item.Packed > 0 {
		packed = item.Packed
	}
	row := dst.PlaneRow(0, y, dst.Width*packed)
	base := x * packed
	if base+packed-1 >= len(row) {
		return
	}
	switch dst.Format {
	case avutil.PixRGB24:
		row[base], row[base+1], row[base+2] = r, g, b
	case avutil.PixBGR24:
		row[base], row[base+1], row[base+2] = b, g, r
	case avutil.PixRGBA:
		row[base], row[base+1], row[base+2], row[base+3] = r, g, b, a
	case avutil.PixBGRA:
		row[base], row[base+1], row[base+2], row[base+3] = b, g, r, a
	case avutil.PixARGB:
		row[base], row[base+1], row[base+2], row[base+3] = a, r, g, b
	case avutil.PixABGR:
		row[base], row[base+1], row[base+2], row[base+3] = a, b, g, r
	case avutil.PixRGB0:
		row[base], row[base+1], row[base+2], row[base+3] = r, g, b, 0
	case avutil.PixBGR0:
		row[base], row[base+1], row[base+2], row[base+3] = b, g, r, 0
	case avutil.Pix0RGB:
		row[base], row[base+1], row[base+2], row[base+3] = 0, r, g, b
	case avutil.Pix0BGR:
		row[base], row[base+1], row[base+2], row[base+3] = 0, b, g, r
	}
}

func unpack565(src *avutil.Frame, x, y int) (byte, byte, byte, byte) {
	row := src.PlaneRow(0, y, src.Width*2)
	base := x * 2
	if base+1 >= len(row) {
		return 0, 0, 0, 255
	}
	v := uint16(row[base]) | uint16(row[base+1])<<8
	r := byte((v >> 11) << 3)
	g := byte(((v >> 5) & 0x3f) << 2)
	b := byte((v & 0x1f) << 3)
	return r, g, b, 255
}

func unpack555(src *avutil.Frame, x, y int) (byte, byte, byte, byte) {
	row := src.PlaneRow(0, y, src.Width*2)
	base := x * 2
	if base+1 >= len(row) {
		return 0, 0, 0, 255
	}
	v := uint16(row[base]) | uint16(row[base+1])<<8
	r := byte(((v >> 10) & 0x1f) << 3)
	g := byte(((v >> 5) & 0x1f) << 3)
	b := byte((v & 0x1f) << 3)
	return r, g, b, 255
}

func pack565(dst *avutil.Frame, x, y int, r, g, b byte) {
	row := dst.PlaneRow(0, y, dst.Width*2)
	base := x * 2
	if base+1 >= len(row) {
		return
	}
	v := uint16(r>>3)<<11 | uint16(g>>2)<<5 | uint16(b>>3)
	row[base] = byte(v)
	row[base+1] = byte(v >> 8)
}

func pack555(dst *avutil.Frame, x, y int, r, g, b byte) {
	row := dst.PlaneRow(0, y, dst.Width*2)
	base := x * 2
	if base+1 >= len(row) {
		return
	}
	v := uint16(r>>3)<<10 | uint16(g>>3)<<5 | uint16(b>>3)
	row[base] = byte(v)
	row[base+1] = byte(v >> 8)
}
