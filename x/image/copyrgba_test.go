package image

import (
	"image"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCopyRGBA(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 2, 1))
	src := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	CopyRGBA(dst, src)
	require.Equal(t, src, dst.Pix[:8])
}
