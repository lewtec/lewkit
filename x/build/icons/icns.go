package icons

import (
	"image"

	"github.com/lewtec/lewkit/x/image/convert"
)

// WriteICNS writes a modern ICNS (PNG payloads) to path.
func WriteICNS(path string, square image.Image, specs []struct {
	Type string
	Size int
}) error {
	icns := make([]convert.ICNSSpec, len(specs))
	for i, spec := range specs {
		icns[i] = convert.ICNSSpec{Type: spec.Type, Size: spec.Size}
	}
	raw, err := convert.EncodeICNS(square, icns)
	if err != nil {
		return err
	}
	return writeIconFile(path, raw)
}
