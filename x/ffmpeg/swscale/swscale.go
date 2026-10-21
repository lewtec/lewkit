package swscale

import (
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// Flags select the resampler kernel. The values match libswscale.
type Flags int

const (
	// FastBilinear is a bilinear kernel.
	FastBilinear Flags = 1
	// Bilinear is the same kernel with accurate rounding.
	Bilinear Flags = 2
	// Bicubic is the default kernel.
	Bicubic Flags = 4
	// Point is nearest neighbor.
	Point Flags = 0x10
	// Area averages source pixels when shrinking.
	Area Flags = 0x20
	// Lanczos is a 3-lobe windowed sinc.
	Lanczos Flags = 0x200
	// Spline is a 4-tap bicubic.
	Spline Flags = 0x400
	// Gauss is a Gaussian kernel.
	Gauss Flags = 0x80
	// AccurateRnd asks the scaler to round to nearest.
	AccurateRnd Flags = 0x40000
)

// Context is one source and destination picture size.
type Context struct {
	SrcW, SrcH int
	DstW, DstH int
	Src        avutil.PixelFormat
	Dst        avutil.PixelFormat
	Flags      Flags
	fullSrc    bool
	tables     *yuvTables
	fullTables *yuvTables
}

// New builds a converter. A zero flag set means bicubic.
func New(srcW, srcH int, src avutil.PixelFormat, dstW, dstH int, dst avutil.PixelFormat, flags Flags) (*Context, error) {
	if srcW <= 0 || srcH <= 0 || dstW <= 0 || dstH <= 0 {
		return nil, fmt.Errorf("%w: scale size", avutil.ErrRange)
	}
	if _, ok := avutil.Desc(src); !ok {
		return nil, fmt.Errorf("%w: %s", avutil.ErrUnsupported, src)
	}
	if _, ok := avutil.Desc(dst); !ok {
		return nil, fmt.Errorf("%w: %s", avutil.ErrUnsupported, dst)
	}
	if flags&^AccurateRnd == 0 {
		flags |= Bicubic
	}
	return &Context{
		SrcW: srcW, SrcH: srcH, DstW: dstW, DstH: dstH,
		Src: src, Dst: dst, Flags: flags,
		fullSrc:    avutil.IsFullRange(src),
		tables:     yuvToRGBTables(false),
		fullTables: yuvToRGBTables(true),
	}, nil
}

// Scale writes src into dst. Both frames must already be allocated.
func (c *Context) Scale(dst, src *avutil.Frame) error {
	if c == nil || dst == nil || src == nil {
		return fmt.Errorf("%w: scale", avutil.ErrInvalidData)
	}
	if src.Width != c.SrcW || src.Height != c.SrcH || src.Format != c.Src {
		return fmt.Errorf("%w: source %s %dx%d", avutil.ErrInvalidData, src.Format, src.Width, src.Height)
	}
	if dst.Width != c.DstW || dst.Height != c.DstH || dst.Format != c.Dst {
		return fmt.Errorf("%w: destination %s %dx%d", avutil.ErrInvalidData, dst.Format, dst.Width, dst.Height)
	}
	if c.Src == c.Dst && c.SrcW == c.DstW && c.SrcH == c.DstH {
		copyPlanes(dst, src)
		return nil
	}
	if c.SrcW == c.DstW && c.SrcH == c.DstH && isYUV8(c.Src) && isPackedRGB(c.Dst) {
		return c.yuvToPacked(dst, src)
	}
	if c.SrcW == c.DstW && c.SrcH == c.DstH && isPackedRGB(c.Src) && isPackedRGB(c.Dst) {
		return shufflePacked(dst, src)
	}
	workSrc := c.Src
	if isRGBFamily(c.Src) {
		workSrc = avutil.PixGBRP
	} else {
		workSrc = avutil.PixYUV444P
		if c.fullSrc {
			workSrc = avutil.PixYUVJ444P
		}
	}
	planar, err := avutil.NewVideo(workSrc, c.SrcW, c.SrcH)
	if err != nil {
		return err
	}
	if err := unpack(planar, src, c); err != nil {
		return err
	}
	scaled := planar
	if c.SrcW != c.DstW || c.SrcH != c.DstH {
		scaled, err = avutil.NewVideo(workSrc, c.DstW, c.DstH)
		if err != nil {
			return err
		}
		if err := scalePlanar(scaled, planar, kernelOf(c.Flags)); err != nil {
			return err
		}
	}
	return pack(dst, scaled, c)
}

func copyPlanes(dst, src *avutil.Frame) {
	for i := 0; i < 8; i++ {
		if len(src.Data[i]) == 0 {
			continue
		}
		n := len(src.Data[i])
		if n > len(dst.Data[i]) {
			n = len(dst.Data[i])
		}
		copy(dst.Data[i][:n], src.Data[i][:n])
	}
	if len(src.Palette) > 0 {
		dst.Palette = append([]byte(nil), src.Palette...)
	}
}

func isPackedRGB(format avutil.PixelFormat) bool {
	switch format {
	case avutil.PixRGB24, avutil.PixBGR24, avutil.PixRGBA, avutil.PixBGRA, avutil.PixARGB, avutil.PixABGR,
		avutil.PixRGB0, avutil.PixBGR0, avutil.Pix0RGB, avutil.Pix0BGR, avutil.PixRGB565LE, avutil.PixRGB555LE:
		return true
	default:
		return false
	}
}

func isRGBFamily(format avutil.PixelFormat) bool {
	if isPackedRGB(format) || format == avutil.PixGBRP || format == avutil.PixGray || format == avutil.PixPAL8 {
		return true
	}
	return false
}

func isYUV8(format avutil.PixelFormat) bool {
	switch format {
	case avutil.PixYUV420P, avutil.PixYUV422P, avutil.PixYUV444P, avutil.PixYUV410P, avutil.PixYUV411P,
		avutil.PixYUVJ420P, avutil.PixYUVJ422P, avutil.PixYUVJ444P, avutil.PixNV12, avutil.PixNV21,
		avutil.PixYUYV422, avutil.PixUYVY422, avutil.PixYUVA420P:
		return true
	default:
		return false
	}
}
