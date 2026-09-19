package image

import (
	stdimage "image"
	"image/color"
	"runtime"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func raster(t *testing.T, expr *ndarray.Node) *stdimage.RGBA {
	t.Helper()
	k, err := ndarray.Compile(expr)
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(k.GLSL(), "void main()"))
	pix, err := k.Eval()
	require.NoError(t, err)
	sh := k.Shape()
	return RGBA(sh[0], sh[1], pix)
}

func TestTriangleColors(t *testing.T) {
	expr, err := Triangle(200, 200, nil)
	require.NoError(t, err)
	dst := raster(t, expr)

	top := dst.RGBAAt(100, 50)
	assert.Greater(t, int(top.R), 180, "top=%v", top)
	assert.Less(t, int(top.G)+int(top.B), 80, "top=%v", top)

	br := dst.RGBAAt(135, 125)
	assert.Greater(t, int(br.G), 180, "br=%v", br)
	assert.Less(t, int(br.R)+int(br.B), 80, "br=%v", br)

	bl := dst.RGBAAt(65, 125)
	assert.Greater(t, int(bl.B), 180, "bl=%v", bl)
	assert.Less(t, int(bl.R)+int(bl.G), 80, "bl=%v", bl)

	assert.Equal(t, color.RGBA{A: 255}, dst.RGBAAt(0, 0))
	assert.Equal(t, color.RGBA{A: 255}, dst.RGBAAt(199, 0))
}

func TestTriangleTurnHalf(t *testing.T) {
	expr, err := Triangle(200, 200, ndarray.Const(0.5))
	require.NoError(t, err)
	dst := raster(t, expr)
	bot := dst.RGBAAt(100, 150)
	assert.Greater(t, int(bot.R), 180, "half turn bottom=%v", bot)
	assert.Equal(t, color.RGBA{A: 255}, dst.RGBAAt(100, 20))
}

func TestTriangleTurnInput(t *testing.T) {
	st, err := ndarray.Of()
	require.NoError(t, err)
	expr, err := Triangle(32, 32, ndarray.In(0, st))
	require.NoError(t, err)
	k, err := ndarray.Compile(expr)
	require.NoError(t, err)
	pix := make([]float32, 32*32*4)
	require.NoError(t, k.EvalInto(pix, [][]float32{{0.5}}))
	dst := RGBA(32, 32, pix)
	bot := dst.RGBAAt(16, 24)
	assert.Greater(t, int(bot.R), 180, "half turn bottom=%v", bot)
}

func TestFill(t *testing.T) {
	expr, err := Fill(2, 2, 10, 20, 30, 255)
	require.NoError(t, err)
	dst := raster(t, expr)
	assert.Equal(t, color.RGBA{10, 20, 30, 255}, dst.RGBAAt(0, 0))
	assert.Equal(t, color.RGBA{10, 20, 30, 255}, dst.RGBAAt(1, 1))
}

func TestPainterHeap(t *testing.T) {
	p, err := New(t.Context())
	require.NoError(t, err)
	test.CloseOnCleanup(t, p)
	dst := stdimage.NewRGBA(stdimage.Rect(0, 0, 64, 64))
	require.NoError(t, p.Draw(t.Context(), dst, 0))
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := range 30 {
		require.NoError(t, p.Draw(t.Context(), dst, float64(i)/30))
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	grew := int64(after.HeapInuse) - int64(before.HeapInuse)
	require.Less(t, grew, int64(8<<20), "heap grew %d bytes over 30 frames", grew)
}

func TestPainterDraw(t *testing.T) {
	p, err := New(t.Context())
	require.NoError(t, err)
	test.CloseOnCleanup(t, p)
	dst := stdimage.NewRGBA(stdimage.Rect(0, 0, 32, 32))
	require.NoError(t, p.Draw(t.Context(), dst, 0))
	require.NoError(t, p.Draw(t.Context(), dst, 0.5))
	bot := dst.RGBAAt(16, 24)
	assert.Greater(t, int(bot.R), 180, "painter half turn=%v", bot)
}

func TestTriangleExec(t *testing.T) {
	d, err := vulkan.Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	test.CloseOnCleanup(t, d)
	expr, err := Triangle(32, 32, nil)
	require.NoError(t, err)
	k, err := ndarray.Compile(expr)
	require.NoError(t, err)
	test.CloseOnCleanup(t, k)
	cpu, err := k.Eval()
	require.NoError(t, err)
	gpu, err := k.Exec(t.Context(), d)
	require.NoError(t, err)
	require.Equal(t, cpu, gpu)
}
