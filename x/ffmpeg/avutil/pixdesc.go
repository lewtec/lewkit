package avutil

// PixDesc describes one pixel format: chroma shift and component count.
type PixDesc struct {
	Format   PixelFormat
	ChromaW  int
	ChromaH  int
	Comp     int
	Depth    int
	Planar   bool
	RGB      bool
	YUV      bool
	Full     bool
	Alpha    bool
	Packed   int
}

func desc(format PixelFormat, chromaW, chromaH, comp, depth int, planar, rgb, yuv, full, alpha bool, packed int) PixDesc {
	return PixDesc{
		Format: format, ChromaW: chromaW, ChromaH: chromaH, Comp: comp, Depth: depth,
		Planar: planar, RGB: rgb, YUV: yuv, Full: full, Alpha: alpha, Packed: packed,
	}
}

// Descs is every pixel format this port can name.
func Descs() []PixDesc {
	return []PixDesc{
		desc(PixYUV420P, 1, 1, 3, 8, true, false, true, false, false, 0),
		desc(PixYUV422P, 1, 0, 3, 8, true, false, true, false, false, 0),
		desc(PixYUV444P, 0, 0, 3, 8, true, false, true, false, false, 0),
		desc(PixYUV410P, 2, 2, 3, 8, true, false, true, false, false, 0),
		desc(PixYUV411P, 2, 0, 3, 8, true, false, true, false, false, 0),
		desc(PixYUVJ420P, 1, 1, 3, 8, true, false, true, true, false, 0),
		desc(PixYUVJ422P, 1, 0, 3, 8, true, false, true, true, false, 0),
		desc(PixYUVJ444P, 0, 0, 3, 8, true, false, true, true, false, 0),
		desc(PixYUYV422, 1, 0, 3, 8, false, false, true, false, false, 2),
		desc(PixUYVY422, 1, 0, 3, 8, false, false, true, false, false, 2),
		desc(PixNV12, 1, 1, 3, 8, true, false, true, false, false, 0),
		desc(PixNV21, 1, 1, 3, 8, true, false, true, false, false, 0),
		desc(PixRGB24, 0, 0, 3, 8, false, true, false, true, false, 3),
		desc(PixBGR24, 0, 0, 3, 8, false, true, false, true, false, 3),
		desc(PixRGBA, 0, 0, 4, 8, false, true, false, true, true, 4),
		desc(PixBGRA, 0, 0, 4, 8, false, true, false, true, true, 4),
		desc(PixARGB, 0, 0, 4, 8, false, true, false, true, true, 4),
		desc(PixABGR, 0, 0, 4, 8, false, true, false, true, true, 4),
		desc(PixRGB0, 0, 0, 4, 8, false, true, false, true, false, 4),
		desc(PixBGR0, 0, 0, 4, 8, false, true, false, true, false, 4),
		desc(Pix0RGB, 0, 0, 4, 8, false, true, false, true, false, 4),
		desc(Pix0BGR, 0, 0, 4, 8, false, true, false, true, false, 4),
		desc(PixGray, 0, 0, 1, 8, true, false, false, true, false, 1),
		desc(PixGBRP, 0, 0, 3, 8, true, true, false, true, false, 0),
		desc(PixRGB565LE, 0, 0, 3, 8, false, true, false, true, false, 2),
		desc(PixRGB555LE, 0, 0, 3, 8, false, true, false, true, false, 2),
		desc(PixYA8, 0, 0, 2, 8, false, false, false, true, true, 2),
		desc(PixYUVA420P, 1, 1, 4, 8, true, false, true, false, true, 0),
		desc(PixPAL8, 0, 0, 1, 8, true, false, false, true, false, 1),
		desc(PixMonoW, 0, 0, 1, 1, false, false, false, true, false, 0),
		desc(PixMonoB, 0, 0, 1, 1, false, false, false, true, false, 0),
	}
}

// DescMust returns the descriptor. An unknown format is a zero descriptor.
func DescMust(format PixelFormat) PixDesc {
	item, _ := Desc(format)
	return item
}

// Desc returns the descriptor of format.
func Desc(format PixelFormat) (PixDesc, bool) {
	for _, item := range Descs() {
		if item.Format == format {
			return item, true
		}
	}
	return PixDesc{}, false
}

// ChromaWidth is the chroma plane width.
func ChromaWidth(format PixelFormat, width int) int {
	item, ok := Desc(format)
	if !ok {
		return width
	}
	return (width + (1 << item.ChromaW) - 1) >> item.ChromaW
}

// ChromaHeight is the chroma plane height.
func ChromaHeight(format PixelFormat, height int) int {
	item, ok := Desc(format)
	if !ok {
		return height
	}
	return (height + (1 << item.ChromaH) - 1) >> item.ChromaH
}

// IsFullRange reports JPEG-range YUV.
func IsFullRange(format PixelFormat) bool {
	item, ok := Desc(format)
	return ok && item.Full
}
