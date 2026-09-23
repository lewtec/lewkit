package tray

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"

	xdraw "golang.org/x/image/draw"
)

// LinuxSizes are the StatusNotifier pixmap sizes hosts pick from.
var LinuxSizes = []int{16, 22, 24, 32, 48}

// ICOSizes are the PNG entries written into an ICO.
var ICOSizes = []int{16, 24, 32, 48, 64, 128, 256}

// ICNSSpecs are modern ICNS type codes with PNG payloads.
var ICNSSpecs = []struct {
	Type string
	Size int
}{
	{"icp4", 16},
	{"icp5", 32},
	{"icp6", 64},
	{"ic07", 128},
	{"ic08", 256},
	{"ic09", 512},
	{"ic10", 1024},
}

// Raster decodes icon and returns a square size×size image.
// A nil image and a nil error means the icon is a name only.
func Raster(icon Icon, size int) (*image.NRGBA, error) {
	src, err := decodeIcon(icon)
	if err != nil || src == nil {
		return nil, err
	}
	if size <= 0 {
		return nil, fmt.Errorf("%w: size %d", ErrIcon, size)
	}
	return resize(padSquare(src), size), nil
}

// Rasters decodes icon once and returns one square image per size.
func Rasters(icon Icon, sizes []int) ([]*image.NRGBA, error) {
	src, err := decodeIcon(icon)
	if err != nil || src == nil {
		return nil, err
	}
	square := padSquare(src)
	out := make([]*image.NRGBA, 0, len(sizes))
	for _, size := range sizes {
		if size <= 0 {
			return nil, fmt.Errorf("%w: size %d", ErrIcon, size)
		}
		out = append(out, resize(square, size))
	}
	return out, nil
}

// ARGB encodes img as StatusNotifier ARGB32, network byte order, straight alpha.
func ARGB(img image.Image) []byte {
	if img == nil {
		return nil
	}
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	out := make([]byte, 0, width*height*4)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			a, r, g, b := straight(img.At(x, y))
			out = append(out, a, r, g, b)
		}
	}
	return out
}

// EncodePNG encodes img as PNG bytes.
func EncodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// EncodeICO writes a multi-size ICO whose entries are PNG images.
func EncodeICO(src image.Image, sizes []int) ([]byte, error) {
	square := padSquare(src)
	type entry struct {
		size int
		png  []byte
	}
	var entries []entry
	for _, size := range sizes {
		if size <= 0 {
			continue
		}
		pngBytes, err := EncodePNG(resize(square, size))
		if err != nil {
			return nil, fmt.Errorf("ico %d: %w", size, err)
		}
		entries = append(entries, entry{size: size, png: pngBytes})
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("%w: ico has no sizes", ErrIcon)
	}
	var buf bytes.Buffer
	if err := writeLE(&buf, uint16(0), uint16(1), uint16(len(entries))); err != nil {
		return nil, err
	}
	offset := 6 + 16*len(entries)
	for _, entry := range entries {
		width, height := entry.size, entry.size
		if width >= 256 {
			width = 0
		}
		if height >= 256 {
			height = 0
		}
		buf.WriteByte(byte(width))
		buf.WriteByte(byte(height))
		buf.WriteByte(0)
		buf.WriteByte(0)
		if err := writeLE(&buf, uint16(1), uint16(32), uint32(len(entry.png)), uint32(offset)); err != nil {
			return nil, err
		}
		offset += len(entry.png)
	}
	for _, entry := range entries {
		buf.Write(entry.png)
	}
	return buf.Bytes(), nil
}

// EncodeICNS writes a modern ICNS whose chunks are PNG images.
func EncodeICNS(src image.Image, specs []struct {
	Type string
	Size int
}) ([]byte, error) {
	square := padSquare(src)
	var body bytes.Buffer
	for _, spec := range specs {
		if spec.Size <= 0 || len(spec.Type) != 4 {
			return nil, fmt.Errorf("%w: icns %q %d", ErrIcon, spec.Type, spec.Size)
		}
		pngBytes, err := EncodePNG(resize(square, spec.Size))
		if err != nil {
			return nil, err
		}
		body.WriteString(spec.Type)
		if err := binary.Write(&body, binary.BigEndian, uint32(8+len(pngBytes))); err != nil {
			return nil, err
		}
		body.Write(pngBytes)
	}
	if body.Len() == 0 {
		return nil, fmt.Errorf("%w: icns has no sizes", ErrIcon)
	}
	var out bytes.Buffer
	out.WriteString("icns")
	if err := binary.Write(&out, binary.BigEndian, uint32(8+body.Len())); err != nil {
		return nil, err
	}
	out.Write(body.Bytes())
	return out.Bytes(), nil
}

func writeLE(w io.Writer, values ...any) error {
	for _, value := range values {
		if err := binary.Write(w, binary.LittleEndian, value); err != nil {
			return err
		}
	}
	return nil
}

func decodeIcon(icon Icon) (image.Image, error) {
	if icon.Image != nil {
		return icon.Image, nil
	}
	if len(icon.Bytes) == 0 {
		return nil, nil
	}
	img, err := decodeBytes(icon.Bytes)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func decodeBytes(raw []byte) (image.Image, error) {
	switch {
	case bytes.HasPrefix(raw, []byte{0x89, 'P', 'N', 'G'}):
		img, err := png.Decode(bytes.NewReader(raw))
		if err != nil {
			return nil, fmt.Errorf("%w: png: %w", ErrIcon, err)
		}
		return img, nil
	case bytes.HasPrefix(raw, []byte{0xff, 0xd8}):
		img, err := jpeg.Decode(bytes.NewReader(raw))
		if err != nil {
			return nil, fmt.Errorf("%w: jpeg: %w", ErrIcon, err)
		}
		return img, nil
	case bytes.HasPrefix(raw, []byte("icns")):
		return decodeICNS(raw)
	case isICO(raw):
		return decodeICO(raw)
	default:
		img, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil {
			return nil, fmt.Errorf("%w: png, jpeg, ico, or icns", ErrIcon)
		}
		return img, nil
	}
}

func isICO(raw []byte) bool {
	if len(raw) < 6 {
		return false
	}
	reserved := binary.LittleEndian.Uint16(raw[0:2])
	kind := binary.LittleEndian.Uint16(raw[2:4])
	return reserved == 0 && (kind == 1 || kind == 2)
}

func decodeICO(raw []byte) (image.Image, error) {
	count := int(binary.LittleEndian.Uint16(raw[4:6]))
	bestSize := -1
	var payload []byte
	for i := range count {
		offset := 6 + 16*i
		if offset+16 > len(raw) {
			return nil, fmt.Errorf("%w: truncated ico", ErrIcon)
		}
		width := int(raw[offset])
		if width == 0 {
			width = 256
		}
		size := int(binary.LittleEndian.Uint32(raw[offset+8 : offset+12]))
		at := int(binary.LittleEndian.Uint32(raw[offset+12 : offset+16]))
		if size < 0 || at < 0 || at+size > len(raw) {
			continue
		}
		if width >= bestSize {
			bestSize = width
			payload = raw[at : at+size]
		}
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("%w: empty ico", ErrIcon)
	}
	if bytes.HasPrefix(payload, []byte{0x89, 'P', 'N', 'G'}) {
		img, err := png.Decode(bytes.NewReader(payload))
		if err != nil {
			return nil, fmt.Errorf("%w: ico png: %w", ErrIcon, err)
		}
		return img, nil
	}
	return decodeDIB(payload)
}

func decodeDIB(raw []byte) (image.Image, error) {
	if len(raw) < 40 {
		return nil, fmt.Errorf("%w: short dib", ErrIcon)
	}
	header := binary.LittleEndian.Uint32(raw[0:4])
	if header < 40 || int(header) > len(raw) {
		return nil, fmt.Errorf("%w: dib header %d", ErrIcon, header)
	}
	width := int(int32(binary.LittleEndian.Uint32(raw[4:8])))
	height := int(int32(binary.LittleEndian.Uint32(raw[8:12])))
	if height < 0 {
		height = -height
	} else {
		height /= 2
	}
	bitCount := binary.LittleEndian.Uint16(raw[14:16])
	compression := binary.LittleEndian.Uint32(raw[16:20])
	if width <= 0 || height <= 0 || bitCount != 32 || compression != 0 {
		return nil, fmt.Errorf("%w: dib %dx%d %d-bit compression %d", ErrIcon, width, height, bitCount, compression)
	}
	pixels := raw[header:]
	stride := width * 4
	need := stride * height
	if len(pixels) < need {
		return nil, fmt.Errorf("%w: dib pixels", ErrIcon)
	}
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		srcRow := pixels[(height-1-y)*stride : (height-y)*stride]
		for x := range width {
			i := x * 4
			dst.SetNRGBA(x, y, color.NRGBA{B: srcRow[i], G: srcRow[i+1], R: srcRow[i+2], A: srcRow[i+3]})
		}
	}
	return dst, nil
}

func decodeICNS(raw []byte) (image.Image, error) {
	if len(raw) < 8 {
		return nil, fmt.Errorf("%w: short icns", ErrIcon)
	}
	total := int(binary.BigEndian.Uint32(raw[4:8]))
	if total > len(raw) {
		total = len(raw)
	}
	var best image.Image
	bestSide := 0
	for at := 8; at+8 <= total; {
		size := int(binary.BigEndian.Uint32(raw[at+4 : at+8]))
		if size < 8 || at+size > total {
			break
		}
		payload := raw[at+8 : at+size]
		if bytes.HasPrefix(payload, []byte{0x89, 'P', 'N', 'G'}) {
			img, err := png.Decode(bytes.NewReader(payload))
			if err == nil && img.Bounds().Dx() >= bestSide {
				best = img
				bestSide = img.Bounds().Dx()
			}
		}
		at += size
	}
	if best == nil {
		return nil, fmt.Errorf("%w: icns has no png", ErrIcon)
	}
	return best, nil
}

func padSquare(src image.Image) *image.NRGBA {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	side := width
	if height > side {
		side = height
	}
	dst := image.NewNRGBA(image.Rect(0, 0, side, side))
	offset := image.Pt((side-width)/2, (side-height)/2)
	rect := image.Rect(offset.X, offset.Y, offset.X+width, offset.Y+height)
	draw.Draw(dst, rect, src, bounds.Min, draw.Over)
	return dst
}

func resize(src image.Image, size int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

func straight(c color.Color) (a, r, g, b byte) {
	r16, g16, b16, a16 := c.RGBA()
	a = byte(a16 >> 8)
	if a16 == 0 {
		return 0, 0, 0, 0
	}
	r = byte((r16 * 65535 / a16) >> 8)
	g = byte((g16 * 65535 / a16) >> 8)
	b = byte((b16 * 65535 / a16) >> 8)
	return a, r, g, b
}
