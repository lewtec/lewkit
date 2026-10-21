package avcodec

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

type imageCodec struct {
	par    Parameters
	decode bool
}

func (c *imageCodec) Decode(pkt *avutil.Packet) ([]*avutil.Frame, error) {
	if pkt == nil || len(pkt.Data) == 0 {
		return nil, nil
	}
	var (
		img image.Image
		err error
	)
	switch c.par.ID {
	case IDPNG:
		img, err = png.Decode(bytes.NewReader(pkt.Data))
	case IDMJPEG:
		img, err = jpeg.Decode(bytes.NewReader(pkt.Data))
	case IDGIF:
		img, err = gif.Decode(bytes.NewReader(pkt.Data))
	case IDBMP:
		img, err = decodeBMP(pkt.Data)
	case IDPPM:
		img, err = decodePPM(pkt.Data)
	default:
		return nil, fmt.Errorf("%w: image %s", avutil.ErrUnsupported, c.par.ID)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %s", avutil.ErrInvalidData, err)
	}
	frame, err := frameFromImage(img)
	if err != nil {
		return nil, err
	}
	frame.PTS = pkt.PTS
	frame.Duration = 1
	return []*avutil.Frame{frame}, nil
}

func (c *imageCodec) Encode(frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil {
		return nil, nil
	}
	img, err := imageFromFrame(frame)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	switch c.par.ID {
	case IDPNG:
		err = png.Encode(&buf, img)
	case IDMJPEG:
		q := 90
		if c.par.QScale > 0 {
			q = 100 - c.par.QScale*3
			if q < 1 {
				q = 1
			}
		}
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: q})
	case IDGIF:
		err = gif.Encode(&buf, img, nil)
	case IDBMP:
		err = encodeBMP(&buf, img)
	case IDPPM:
		err = encodePPM(&buf, img)
	default:
		return nil, fmt.Errorf("%w: image %s", avutil.ErrUnsupported, c.par.ID)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %s", avutil.ErrInvalidData, err)
	}
	return []*avutil.Packet{{
		Data: buf.Bytes(), PTS: frame.PTS, Duration: 1, TimeBase: frame.TimeBase, Key: true,
	}}, nil
}

func frameFromImage(img image.Image) (*avutil.Frame, error) {
	if ycb, ok := img.(*image.YCbCr); ok {
		format := avutil.PixYUVJ444P
		switch ycb.SubsampleRatio {
		case image.YCbCrSubsampleRatio420:
			format = avutil.PixYUVJ420P
		case image.YCbCrSubsampleRatio422:
			format = avutil.PixYUVJ422P
		}
		frame, err := avutil.NewVideo(format, ycb.Bounds().Dx(), ycb.Bounds().Dy())
		if err != nil {
			return nil, err
		}
		copyPlane(frame, 0, ycb.Y, ycb.YStride, frame.Width, frame.Height)
		cw := avutil.ChromaWidth(format, frame.Width)
		ch := avutil.ChromaHeight(format, frame.Height)
		copyPlane(frame, 1, ycb.Cb, ycb.CStride, cw, ch)
		copyPlane(frame, 2, ycb.Cr, ycb.CStride, cw, ch)
		return frame, nil
	}
	b := img.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)
	format := avutil.PixRGB24
	for i := 3; i < len(rgba.Pix); i += 4 {
		if rgba.Pix[i] != 255 {
			format = avutil.PixRGBA
			break
		}
	}
	frame, err := avutil.NewVideo(format, rgba.Bounds().Dx(), rgba.Bounds().Dy())
	if err != nil {
		return nil, err
	}
	packed := 3
	if format == avutil.PixRGBA {
		packed = 4
	}
	for y := 0; y < frame.Height; y++ {
		row := frame.PlaneRow(0, y, frame.Width*packed)
		src := rgba.Pix[y*rgba.Stride : y*rgba.Stride+frame.Width*4]
		if format == avutil.PixRGBA {
			copy(row, src[:frame.Width*4])
			continue
		}
		for x := 0; x < frame.Width; x++ {
			row[x*3] = src[x*4]
			row[x*3+1] = src[x*4+1]
			row[x*3+2] = src[x*4+2]
		}
	}
	return frame, nil
}

func copyPlane(frame *avutil.Frame, plane int, src []byte, stride, width, height int) {
	for y := 0; y < height; y++ {
		row := frame.PlaneRow(plane, y, width)
		from := y * stride
		if from+width > len(src) {
			return
		}
		copy(row, src[from:from+width])
	}
}

func imageFromFrame(frame *avutil.Frame) (image.Image, error) {
	switch frame.Format {
	case avutil.PixYUVJ420P, avutil.PixYUV420P, avutil.PixYUVJ422P, avutil.PixYUV422P, avutil.PixYUVJ444P, avutil.PixYUV444P:
		ratio := image.YCbCrSubsampleRatio444
		switch frame.Format {
		case avutil.PixYUV420P, avutil.PixYUVJ420P:
			ratio = image.YCbCrSubsampleRatio420
		case avutil.PixYUV422P, avutil.PixYUVJ422P:
			ratio = image.YCbCrSubsampleRatio422
		}
		img := image.NewYCbCr(image.Rect(0, 0, frame.Width, frame.Height), ratio)
		copyOut(frame, 0, img.Y, img.YStride, frame.Width, frame.Height)
		copyOut(frame, 1, img.Cb, img.CStride, avutil.ChromaWidth(frame.Format, frame.Width), avutil.ChromaHeight(frame.Format, frame.Height))
		copyOut(frame, 2, img.Cr, img.CStride, avutil.ChromaWidth(frame.Format, frame.Width), avutil.ChromaHeight(frame.Format, frame.Height))
		return img, nil
	case avutil.PixGray:
		img := image.NewGray(image.Rect(0, 0, frame.Width, frame.Height))
		copyOut(frame, 0, img.Pix, img.Stride, frame.Width, frame.Height)
		return img, nil
	}
	packed := 4
	img := image.NewRGBA(image.Rect(0, 0, frame.Width, frame.Height))
	for y := 0; y < frame.Height; y++ {
		dst := img.Pix[y*img.Stride : y*img.Stride+frame.Width*4]
		switch frame.Format {
		case avutil.PixRGB24:
			row := frame.PlaneRow(0, y, frame.Width*3)
			for x := 0; x < frame.Width; x++ {
				dst[x*4] = row[x*3]
				dst[x*4+1] = row[x*3+1]
				dst[x*4+2] = row[x*3+2]
				dst[x*4+3] = 255
			}
		case avutil.PixBGR24:
			row := frame.PlaneRow(0, y, frame.Width*3)
			for x := 0; x < frame.Width; x++ {
				dst[x*4] = row[x*3+2]
				dst[x*4+1] = row[x*3+1]
				dst[x*4+2] = row[x*3]
				dst[x*4+3] = 255
			}
		case avutil.PixRGBA:
			copy(dst, frame.PlaneRow(0, y, frame.Width*packed))
		case avutil.PixBGRA:
			row := frame.PlaneRow(0, y, frame.Width*4)
			for x := 0; x < frame.Width; x++ {
				dst[x*4] = row[x*4+2]
				dst[x*4+1] = row[x*4+1]
				dst[x*4+2] = row[x*4]
				dst[x*4+3] = row[x*4+3]
			}
		default:
			return nil, fmt.Errorf("%w: image %s", avutil.ErrUnsupported, frame.Format)
		}
	}
	return img, nil
}

func copyOut(frame *avutil.Frame, plane int, dst []byte, stride, width, height int) {
	for y := 0; y < height; y++ {
		copy(dst[y*stride:y*stride+width], frame.PlaneRow(plane, y, width))
	}
}

func decodeBMP(raw []byte) (image.Image, error) {
	if len(raw) < 54 || raw[0] != 'B' || raw[1] != 'M' {
		return nil, fmt.Errorf("%w: bmp", avutil.ErrInvalidData)
	}
	off := int(u32(raw[10:]))
	w := int(int32(u32(raw[18:])))
	h := int(int32(u32(raw[22:])))
	bpp := int(u16(raw[28:]))
	bottom := true
	if h < 0 {
		h = -h
		bottom = false
	}
	if w <= 0 || h <= 0 || (bpp != 24 && bpp != 32) || off > len(raw) {
		return nil, fmt.Errorf("%w: bmp", avutil.ErrInvalidData)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	stride := ((w*bpp/8 + 3) / 4) * 4
	for y := 0; y < h; y++ {
		srcY := y
		if bottom {
			srcY = h - 1 - y
		}
		row := raw[off+srcY*stride:]
		dst := img.Pix[y*img.Stride:]
		for x := 0; x < w; x++ {
			if bpp == 24 {
				dst[x*4] = row[x*3+2]
				dst[x*4+1] = row[x*3+1]
				dst[x*4+2] = row[x*3]
				dst[x*4+3] = 255
			} else {
				dst[x*4] = row[x*4+2]
				dst[x*4+1] = row[x*4+1]
				dst[x*4+2] = row[x*4]
				dst[x*4+3] = row[x*4+3]
			}
		}
	}
	return img, nil
}

func encodeBMP(buf *bytes.Buffer, img image.Image) error {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	stride := ((w*3 + 3) / 4) * 4
	size := 54 + stride*h
	hdr := make([]byte, 54)
	hdr[0], hdr[1] = 'B', 'M'
	putU32(hdr[2:], uint32(size))
	putU32(hdr[10:], 54)
	putU32(hdr[14:], 40)
	putU32(hdr[18:], uint32(w))
	putU32(hdr[22:], uint32(h))
	putU16(hdr[26:], 1)
	putU16(hdr[28:], 24)
	putU32(hdr[34:], uint32(stride*h))
	buf.Write(hdr)
	pad := make([]byte, stride-w*3)
	for y := h - 1; y >= 0; y-- {
		for x := 0; x < w; x++ {
			r, g, b, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			buf.WriteByte(byte(b >> 8))
			buf.WriteByte(byte(g >> 8))
			buf.WriteByte(byte(r >> 8))
		}
		buf.Write(pad)
	}
	return nil
}

func decodePPM(raw []byte) (image.Image, error) {
	if len(raw) < 3 || raw[0] != 'P' || raw[1] != '6' {
		return nil, fmt.Errorf("%w: ppm", avutil.ErrInvalidData)
	}
	i := 2
	var next func() (int, error)
	next = func() (int, error) {
		for i < len(raw) && (raw[i] == ' ' || raw[i] == '\n' || raw[i] == '\r' || raw[i] == '\t') {
			i++
		}
		if i < len(raw) && raw[i] == '#' {
			for i < len(raw) && raw[i] != '\n' {
				i++
			}
			return next()
		}
		n := 0
		start := i
		for i < len(raw) && raw[i] >= '0' && raw[i] <= '9' {
			n = n*10 + int(raw[i]-'0')
			i++
		}
		if i == start {
			return 0, fmt.Errorf("%w: ppm", avutil.ErrInvalidData)
		}
		return n, nil
	}
	w, err := next()
	if err != nil {
		return nil, err
	}
	h, err := next()
	if err != nil {
		return nil, err
	}
	maxv, err := next()
	if err != nil || maxv != 255 || w <= 0 || h <= 0 {
		return nil, fmt.Errorf("%w: ppm", avutil.ErrInvalidData)
	}
	if i < len(raw) && (raw[i] == '\n' || raw[i] == ' ') {
		i++
	}
	if len(raw) < i+w*h*3 {
		return nil, fmt.Errorf("%w: ppm", avutil.ErrInvalidData)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	pix := raw[i:]
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			o := (y*w + x) * 3
			img.SetRGBA(x, y, color.RGBA{R: pix[o], G: pix[o+1], B: pix[o+2], A: 255})
		}
	}
	return img, nil
}

func encodePPM(buf *bytes.Buffer, img image.Image) error {
	b := img.Bounds()
	fmt.Fprintf(buf, "P6\n%d %d\n255\n", b.Dx(), b.Dy())
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bv, _ := img.At(x, y).RGBA()
			buf.WriteByte(byte(r >> 8))
			buf.WriteByte(byte(g >> 8))
			buf.WriteByte(byte(bv >> 8))
		}
	}
	return nil
}

func u16(b []byte) uint16 { return uint16(b[0]) | uint16(b[1])<<8 }
func u32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}
func putU16(b []byte, v uint16) { b[0] = byte(v); b[1] = byte(v >> 8) }
func putU32(b []byte, v uint32) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}
