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

var ErrICONoSizes = errors.New("ico: no sizes")

// WriteICO writes a multi-size ICO (PNG-compressed entries) to path.
func WriteICO(path string, square image.Image, sizes []int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	type entry struct {
		size int
		png  []byte
	}
	var entries []entry
	for _, s := range sizes {
		if s <= 0 {
			continue
		}
		img := Resize(square, s)
		pngb, err := EncodePNG(img)
		if err != nil {
			return fmt.Errorf("ico %d: %w", s, err)
		}
		entries = append(entries, entry{size: s, png: pngb})
	}
	if len(entries) == 0 {
		return ErrICONoSizes
	}

	var buf bytes.Buffer
	// ICONDIR — bytes.Buffer never fails; still check so call sites stay honest.
	if err := binary.Write(&buf, binary.LittleEndian, uint16(0)); err != nil { // reserved
		return err
	}
	if err := binary.Write(&buf, binary.LittleEndian, uint16(1)); err != nil { // type icon
		return err
	}
	if err := binary.Write(&buf, binary.LittleEndian, uint16(len(entries))); err != nil {
		return err
	}

	// Offset after header + all directory entries (16 bytes each)
	offset := 6 + 16*len(entries)
	for _, e := range entries {
		w, h := e.size, e.size
		if w >= 256 {
			w = 0
		}
		if h >= 256 {
			h = 0
		}
		buf.WriteByte(byte(w))
		buf.WriteByte(byte(h))
		buf.WriteByte(0) // color palette
		buf.WriteByte(0) // reserved
		if err := binary.Write(&buf, binary.LittleEndian, uint16(1)); err != nil { // planes
			return err
		}
		if err := binary.Write(&buf, binary.LittleEndian, uint16(32)); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.LittleEndian, uint32(len(e.png))); err != nil {
			return err
		}
		if err := binary.Write(&buf, binary.LittleEndian, uint32(offset)); err != nil {
			return err
		}
		offset += len(e.png)
	}
	for _, e := range entries {
		buf.Write(e.png)
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
