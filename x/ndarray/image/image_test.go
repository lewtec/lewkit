package image

import (
	stdimage "image"
	"image/color"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func raster(t *testing.T, expr *ndarray.Tensor) *stdimage.RGBA {
	t.Helper()
	require.NoError(t, expr.Eval(t.Context(), ndarray.CPU))
	pixels, err := expr.Data()
	require.NoError(t, err)
	shape := expr.Shape()
	return RGBA(shape[0], shape[1], pixels)
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
	turn, err := ndarray.New([]float32{0.5}, nil)
	require.NoError(t, err)
	expr, err := Triangle(32, 32, turn)
	require.NoError(t, err)
	dst := raster(t, expr)
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

func vmSizeKB(t *testing.T) int64 {
	t.Helper()
	b, err := os.ReadFile("/proc/self/status")
	require.NoError(t, err)
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(line, "VmSize:") {
			continue
		}
		f := strings.Fields(line)
		n, err := strconv.ParseInt(f[1], 10, 64)
		require.NoError(t, err)
		return n
	}
	t.Fatal("no VmSize")
	return 0
}

func TestPainterVirt(t *testing.T) {
	p, err := New(t.Context())
	require.NoError(t, err)
	test.CloseOnCleanup(t, p)
	dst := stdimage.NewRGBA(stdimage.Rect(0, 0, 256, 256))
	require.NoError(t, p.Draw(t.Context(), dst, 0))
	v0 := vmSizeKB(t)
	for i := range 40 {
		require.NoError(t, p.Draw(t.Context(), dst, float64(i)/40))
	}
	v1 := vmSizeKB(t)
	t.Logf("VmSize %d -> %d kB (%+d) over 40 painter frames", v0, v1, v1-v0)
	require.Less(t, v1-v0, int64(64*1024), "virtual size grew %d kB", v1-v0)
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

func TestPainterDrawAllocs(t *testing.T) {
	p, err := New(t.Context())
	require.NoError(t, err)
	test.CloseOnCleanup(t, p)
	dst := stdimage.NewRGBA(stdimage.Rect(0, 0, 32, 32))
	require.NoError(t, p.Draw(t.Context(), dst, 0))
	gpu := p.eval
	p.eval = ndarray.CPU
	require.NoError(t, p.Draw(t.Context(), dst, 0))
	defer func() { p.eval = gpu }()
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	n := testing.AllocsPerRun(30, func() {
		if err := p.Draw(t.Context(), dst, 0.25); err != nil {
			panic(err)
		}
	})
	require.Equal(t, 0.0, n)
}

func TestTriangleExec(t *testing.T) {
	d, err := vulkan.Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	test.CloseOnCleanup(t, d)
	expr, err := Triangle(32, 32, nil)
	require.NoError(t, err)
	require.NoError(t, expr.Eval(t.Context(), ndarray.CPU))
	cpu, err := expr.Data()
	require.NoError(t, err)
	cpu = append([]float32(nil), cpu...)
	require.NoError(t, expr.Eval(t.Context(), &ndarray.Vulkan{Device: d}))
	gpu, err := expr.Data()
	require.NoError(t, err)
	require.Equal(t, cpu, gpu)
}
