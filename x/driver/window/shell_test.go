package window_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/stretchr/testify/require"
)

func TestShellPicturePixel(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 255, A: 128})
	got := window.ShellPicture(img)
	require.Equal(t, []uint32{2, 1, 0xFFFF0000, 0x80FF0000}, got)
}

func TestShellPictureNil(t *testing.T) {
	require.Empty(t, window.ShellPicture(nil))
}
