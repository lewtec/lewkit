package icons

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
)

var ErrICNSBadSpec = errors.New("icns: bad type/size")

// WriteICNS writes a modern ICNS (PNG payloads) to path.
func WriteICNS(path string, square image.Image, specs []struct {
	Type string
	Size int
}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var body bytes.Buffer
	for _, sp := range specs {
		if sp.Size <= 0 || len(sp.Type) != 4 {
			return fmt.Errorf("%w: %q %d", ErrICNSBadSpec, sp.Type, sp.Size)
		}
		img := Resize(square, sp.Size)
		pngb, err := EncodePNG(img)
		if err != nil {
			return err
		}
		// type + length (including 8-byte header) + data
		body.WriteString(sp.Type)
		if err := binary.Write(&body, binary.BigEndian, uint32(8+len(pngb))); err != nil {
			return err
		}
		body.Write(pngb)
	}
	var out bytes.Buffer
	out.WriteString("icns")
	if err := binary.Write(&out, binary.BigEndian, uint32(8+body.Len())); err != nil {
		return err
	}
	out.Write(body.Bytes())
	return os.WriteFile(path, out.Bytes(), 0o644)
}
