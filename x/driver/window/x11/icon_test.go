package x11

import (
	"image"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyXIconSkipsWithoutWindow(t *testing.T) {
	err := ApplyXIcon(0, image.NewNRGBA(image.Rect(0, 0, 1, 1)))
	require.NoError(t, err)
}
