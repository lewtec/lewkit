// Package ico encodes and decodes ICO bytes. Entries are PNG images.
package ico

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"

	"github.com/lewtec/lewkit/x/image/convert"
	"github.com/lewtec/lewkit/x/image/convert/png"
)

// Sizes are the PNG entries written into an ICO.
var Sizes = []int{16, 24, 32, 48, 64, 128, 256}

// Match reports whether raw is an ICO or CUR header.
func Match(raw []byte) bool {
	if len(raw) < 6 {
		return false
	}
	reserved := binary.LittleEndian.Uint16(raw[0:2])
	kind := binary.LittleEndian.Uint16(raw[2:4])
	return reserved == 0 && (kind == 1 || kind == 2)
}

// Decode reads an ICO and returns the largest image.
func Decode(raw []byte) (image.Image, error) {
	if len(raw) < 6 {
		return nil, fmt.Errorf("%w: short ico", convert.ErrFormat)
	}
	count := int(binary.LittleEndian.Uint16(raw[4:6]))
	bestSize := -1
	var payload []byte
	for i := range count {
		offset := 6 + 16*i
		if offset+16 > len(raw) {
			return nil, fmt.Errorf("%w: truncated ico", convert.ErrFormat)
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
		return nil, fmt.Errorf("%w: empty ico", convert.ErrFormat)
	}
	if png.Match(payload) {
		img, err := png.Decode(payload)
		if err != nil {
			return nil, fmt.Errorf("%w: ico png: %w", convert.ErrFormat, err)
		}
		return img, nil
	}
	return decodeDIB(payload)
}

// Encode writes a multi-size ICO whose entries are PNG images.
func Encode(src image.Image, sizes []int) ([]byte, error) {
	square := convert.Pad(src)
	type entry struct {
		size int
		png  []byte
	}
	var entries []entry
	for _, size := range sizes {
		if size <= 0 {
			continue
		}
		pngBytes, err := png.Encode(convert.Resize(square, size))
		if err != nil {
			return nil, fmt.Errorf("%w: ico %d: %w", convert.ErrFormat, size, err)
		}
		entries = append(entries, entry{size: size, png: pngBytes})
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("%w: ico has no sizes", convert.ErrFormat)
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

func writeLE(w interface{ Write([]byte) (int, error) }, values ...any) error {
	for _, value := range values {
		if err := binary.Write(w, binary.LittleEndian, value); err != nil {
			return err
		}
	}
	return nil
}

func decodeDIB(raw []byte) (image.Image, error) {
	if len(raw) < 40 {
		return nil, fmt.Errorf("%w: short dib", convert.ErrFormat)
	}
	header := binary.LittleEndian.Uint32(raw[0:4])
	if header < 40 || int(header) > len(raw) {
		return nil, fmt.Errorf("%w: dib header %d", convert.ErrFormat, header)
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
		return nil, fmt.Errorf("%w: dib %dx%d %d-bit compression %d", convert.ErrFormat, width, height, bitCount, compression)
	}
	pixels := raw[header:]
	stride := width * 4
	need := stride * height
	if len(pixels) < need {
		return nil, fmt.Errorf("%w: dib pixels", convert.ErrFormat)
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
