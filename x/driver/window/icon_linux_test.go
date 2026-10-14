//go:build linux && !android

package window

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadIconPNG(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	path := filepath.Join(t.TempDir(), "icon.png")
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0o644))
	got, err := readIcon(path)
	require.NoError(t, err)
	r, g, b, a := got.At(0, 0).RGBA()
	if r>>8 != 10 || g>>8 != 20 || b>>8 != 30 || a>>8 != 255 {
		require.FailNowf(t, "", "pixel %d %d %d %d", r, g, b, a)
	}
}
