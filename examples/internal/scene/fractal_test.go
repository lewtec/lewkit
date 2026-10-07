package scene

import (
	stdimage "image"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/ui/gui"
	"github.com/stretchr/testify/require"
)

func TestFractalEval(t *testing.T) {
	const width, height = 32, 32
	dst := evalFractal(t, height, width, 0.5)
	minSum, maxSum := 255*3, 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			p := dst.RGBAAt(x, y)
			require.Equal(t, uint8(255), p.A)
			sum := int(p.R) + int(p.G) + int(p.B)
			if sum < minSum {
				minSum = sum
			}
			if sum > maxSum {
				maxSum = sum
			}
		}
	}
	require.Equal(t, 0, minSum, "filled Julia is black")
	require.Greater(t, maxSum, 80, "escaped pixels take the palette")
	center := dst.RGBAAt(width/2, height/2)
	require.Equal(t, uint8(0), center.R)
	require.Equal(t, uint8(0), center.G)
	require.Equal(t, uint8(0), center.B)
	corner := dst.RGBAAt(0, 0)
	require.Greater(t, int(corner.R)+int(corner.G)+int(corner.B), 40)
}

func TestFractalAnimates(t *testing.T) {
	a := evalFractal(t, 24, 24, 0)
	b := evalFractal(t, 24, 24, 0.5)
	require.NotEqual(t, a.Pix, b.Pix)
}

func TestFractalOneKernel(t *testing.T) {
	expr, err := fractalAt(8, 8, ndarray.Const(float32(0)))
	require.NoError(t, err)
	pixels := expr.Cast[uint8]()
	dst := make([]uint8, 8*8*4)
	require.NoError(t, pixels.Eval(t.Context(), ndarray.CPU, dst))
	k := pixels.Kernel()
	require.NotNil(t, k)
	code, err := k.Code()
	require.NoError(t, err)
	require.Equal(t, 1, stores(code))
}

func stores(c ndarray.Code) int {
	n := 0
	for _, s := range c.Stmts {
		if s.Op == ndarray.StmtStore {
			n++
		}
	}
	return n
}

func TestFractalRejectsShape(t *testing.T) {
	_, err := fractalAt(0, 4, ndarray.Const(float32(0)))
	require.ErrorIs(t, err, ndarray.ErrShape)
	turn, err := ndarray.New([]float32{0, 1}, ndarray.Shape{2})
	require.NoError(t, err)
	_, err = fractalAt(4, 4, turn)
	require.ErrorIs(t, err, ndarray.ErrShape)
}

func TestFractalModel(t *testing.T) {
	model, err := FractalModel(800, 600)
	require.NoError(t, err)
	require.NotNil(t, model.Init())
	next, cmd := model.Update(gui.TickMsg{
		Elapsed: time.Second,
		Size:    stdimage.Pt(800, 600),
		Period:  time.Second / 60,
	})
	require.NotNil(t, cmd)
	raster, ok := next.View().(*gui.Raster)
	require.True(t, ok)
	require.Equal(t, ndarray.Shape{1, 1, 4}, raster.Pixels.Shape())
}

func evalFractal(t *testing.T, h, w int, turn float32) *stdimage.RGBA {
	t.Helper()
	expr, err := fractalAt(h, w, ndarray.Const(turn))
	require.NoError(t, err)
	dst := stdimage.NewRGBA(stdimage.Rect(0, 0, w, h))
	require.NoError(t, window.Present(t.Context(), expr.Cast[uint8](), ndarray.CPU, dst))
	return dst
}
