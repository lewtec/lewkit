package experiments

import (
	stdimage "image"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/stretchr/testify/require"
)

func TestPerlinEval(t *testing.T) {
	expr, err := perlinAt(16, 16, ndarray.Const(float32(0.3)))
	require.NoError(t, err)
	dst := stdimage.NewRGBA(stdimage.Rect(0, 0, 16, 16))
	require.NoError(t, window.Present(t.Context(), expr.Cast[uint8](), ndarray.CPU, dst))
	minV, maxV := 255, 0
	var sum int
	for i := 0; i < len(dst.Pix); i += 4 {
		g := int(dst.Pix[i])
		require.Equal(t, dst.Pix[i], dst.Pix[i+1])
		require.Equal(t, dst.Pix[i], dst.Pix[i+2])
		require.Equal(t, uint8(255), dst.Pix[i+3])
		if g < minV {
			minV = g
		}
		if g > maxV {
			maxV = g
		}
		sum += g
	}
	require.Greater(t, maxV-minV, 8)
	require.NotEqual(t, 0, sum)
}

func TestPerlinOneKernel(t *testing.T) {
	expr, err := perlinAt(8, 8, ndarray.Const(float32(0)))
	require.NoError(t, err)
	pixels := expr.Cast[uint8]()
	dst := make([]uint8, 8*8*4)
	require.NoError(t, pixels.Eval(t.Context(), ndarray.CPU, dst))
	k := pixels.Kernel()
	require.NotNil(t, k)
	src, err := k.GLSL()
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(src, "void main()"))
	require.Equal(t, 1, strings.Count(src, "gl_GlobalInvocationID"))
}
