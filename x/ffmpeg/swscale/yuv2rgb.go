package swscale

import "github.com/lewtec/lewkit/x/ffmpeg/avutil"

const (
	yuvHeadroom  = 512
	yuvTableSize = 1024 + 2*yuvHeadroom
	yuvPad       = 8192
)

// yuvTables is libswscale's C yuv2rgb lookup for 8-bit RGB.
type yuvTables struct {
	y    []byte
	base int
	rV   [256 + 2*yuvHeadroom]int
	gU   [256 + 2*yuvHeadroom]int
	bU   [256 + 2*yuvHeadroom]int
	gV   [256 + 2*yuvHeadroom]int
}

func yuvToRGBTables(full bool) *yuvTables {
	crv := int64(104597)
	cbu := int64(132201)
	cgu := int64(-25675)
	cgv := int64(-53279)
	cy := int64(1 << 16)
	var oy int64
	if !full {
		cy = (cy * 255) / 219
		oy = 16 << 16
	} else {
		crv = (crv * 224) / 255
		cbu = (cbu * 224) / 255
		cgu = (cgu * 224) / 255
		cgv = (cgv * 224) / 255
	}
	crv = ((crv << 16) + 0x8000) / cy
	cbu = ((cbu << 16) + 0x8000) / cy
	cgu = ((cgu << 16) + 0x8000) / cy
	cgv = ((cgv << 16) + 0x8000) / cy

	y := make([]byte, yuvPad+yuvTableSize+yuvPad)
	base := yuvPad
	yb := -(384 << 16) - int64(yuvHeadroom)*cy - oy
	for i := 0; i < yuvTableSize; i++ {
		y[base+i] = avutil.ClipUint8(int((yb + 0x8000) >> 16))
		yb += cy
	}
	yoffs := 326 + yuvHeadroom
	if full {
		yoffs = 384 + yuvHeadroom
	}
	tables := &yuvTables{y: y, base: base}
	fill := func(dst *[256 + 2*yuvHeadroom]int, inc int64) {
		shift := inc >> 9
		for i := 0; i < 256+2*yuvHeadroom; i++ {
			sample := i - yuvHeadroom
			if sample < 0 {
				sample = 0
			} else if sample > 255 {
				sample = 255
			}
			cb := int64(sample) * inc
			dst[i] = yoffs - int(shift) + int(cb>>16)
		}
	}
	fill(&tables.rV, crv)
	fill(&tables.gU, cgu)
	fill(&tables.bU, cbu)
	off := -(cgv >> 9)
	for i := 0; i < 256+2*yuvHeadroom; i++ {
		sample := i - yuvHeadroom
		if sample < 0 {
			sample = 0
		} else if sample > 255 {
			sample = 255
		}
		cb := int64(sample) * cgv
		tables.gV[i] = int(off + (cb >> 16))
	}
	return tables
}

func (t *yuvTables) RGB(y, u, v byte) (byte, byte, byte) {
	ri := t.rV[int(v)+yuvHeadroom] + int(y)
	gi := t.gU[int(u)+yuvHeadroom] + t.gV[int(v)+yuvHeadroom] + int(y)
	bi := t.bU[int(u)+yuvHeadroom] + int(y)
	return t.at(ri), t.at(gi), t.at(bi)
}

func (t *yuvTables) at(index int) byte {
	pos := t.base + index
	if pos < 0 || pos >= len(t.y) {
		return 0
	}
	return t.y[pos]
}

func (c *Context) tablesFor(format avutil.PixelFormat) *yuvTables {
	if avutil.IsFullRange(format) {
		return c.fullTables
	}
	return c.tables
}
