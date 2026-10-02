package icons

import (
	"image"
	"os"
	"path/filepath"

	"github.com/lewtec/lewkit/x/image/convert"
)

// WriteICO writes a multi-size ICO (PNG-compressed entries) to path.
func WriteICO(path string, square image.Image, sizes []int) error {
	raw, err := convert.EncodeICO(square, sizes)
	if err != nil {
		return err
	}
	return writeIconFile(path, raw)
}

func writeIconFile(path string, raw []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}
