package icons

import (
	"image"

	"github.com/lewtec/lewkit/x/image/convert/icns"
)

// WriteICNS writes a modern ICNS (PNG payloads) to path.
func WriteICNS(path string, square image.Image, specs []struct {
	Type string
	Size int
}) error {
	chunks := make([]icns.Spec, len(specs))
	for i, spec := range specs {
		chunks[i] = icns.Spec{Type: spec.Type, Size: spec.Size}
	}
	raw, err := icns.Encode(square, chunks)
	if err != nil {
		return err
	}
	return writeIconFile(path, raw)
}
