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

func TestShellPictureDefaultSizes(t *testing.T) {
	got := window.ShellPicture(nil)
	require.GreaterOrEqual(t, len(got), 2)
	require.Equal(t, uint32(32), got[0])
	require.Equal(t, uint32(32), got[1])
	at128 := 2 + 32*32
	require.Greater(t, len(got), at128+1)
	require.Equal(t, uint32(128), got[at128])
	require.Equal(t, uint32(128), got[at128+1])
	at256 := at128 + 2 + 128*128
	require.Greater(t, len(got), at256+1)
	require.Equal(t, uint32(256), got[at256])
	require.Equal(t, uint32(256), got[at256+1])
	require.Len(t, got, at256+2+256*256)
}
