// Package icns encodes and decodes modern ICNS bytes. Chunks are PNG images.
package icns

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"

	"github.com/lewtec/lewkit/x/image/convert"
	"github.com/lewtec/lewkit/x/image/convert/png"
)

// Spec is one ICNS chunk: a four-byte type code and a pixel size.
type Spec struct {
	Type string
	Size int
}

// Specs are PNG-in-ICNS type codes.
var Specs = []Spec{
	{"icp4", 16},
	{"icp5", 32},
	{"icp6", 64},
	{"ic07", 128},
	{"ic08", 256},
	{"ic09", 512},
	{"ic10", 1024},
}

// Match reports whether raw starts with an ICNS header.
func Match(raw []byte) bool {
	return bytes.HasPrefix(raw, []byte("icns"))
}

// Decode reads an ICNS and returns the largest PNG chunk.
func Decode(raw []byte) (image.Image, error) {
	if len(raw) < 8 {
		return nil, fmt.Errorf("%w: short icns", convert.ErrFormat)
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
		if png.Match(payload) {
			img, err := png.Decode(payload)
			if err == nil && img.Bounds().Dx() >= bestSide {
				best = img
				bestSide = img.Bounds().Dx()
			}
		}
		at += size
	}
	if best == nil {
		return nil, fmt.Errorf("%w: icns has no png", convert.ErrFormat)
	}
	return best, nil
}

// Encode writes a modern ICNS whose chunks are PNG images.
func Encode(src image.Image, specs []Spec) ([]byte, error) {
	square := convert.Pad(src)
	var body bytes.Buffer
	for _, spec := range specs {
		if spec.Size <= 0 || len(spec.Type) != 4 {
			return nil, fmt.Errorf("%w: icns %q %d", convert.ErrFormat, spec.Type, spec.Size)
		}
		pngBytes, err := png.Encode(convert.Resize(square, spec.Size))
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
		return nil, fmt.Errorf("%w: icns has no sizes", convert.ErrFormat)
	}
	var out bytes.Buffer
	out.WriteString("icns")
	if err := binary.Write(&out, binary.BigEndian, uint32(8+body.Len())); err != nil {
		return nil, err
	}
	out.Write(body.Bytes())
	return out.Bytes(), nil
}
