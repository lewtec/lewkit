package experiments

import (
	stdimage "image"
	"image/color"
	"os"
	"runtime"
	"runtime/debug"
	"testing"

	_ "github.com/lewtec/lewkit/x/driver/ndeval"
	"github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
	ndimage "github.com/lewtec/lewkit/x/ndarray/image"
	"github.com/lewtec/lewkit/x/test"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func raster(t *testing.T, expr *ndarray.Tensor[float32]) *stdimage.RGBA {
	t.Helper()
	dst, err := ndimage.Raster(t.Context(), expr, ndarray.CPU)
	require.NoError(t, err)
	return dst
}

func TestTriangleColors(t *testing.T) {
	expr, err := triangleAt(200, 200, nil)
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
	expr, err := triangleAt(200, 200, ndarray.Const(0.5))
	require.NoError(t, err)
	dst := raster(t, expr)
	bot := dst.RGBAAt(100, 150)
	assert.Greater(t, int(bot.R), 180, "half turn bottom=%v", bot)
	assert.Equal(t, color.RGBA{A: 255}, dst.RGBAAt(100, 20))
}

func TestTriangleTurnInput(t *testing.T) {
	turn, err := ndarray.New([]float32{0.5}, nil)
	require.NoError(t, err)
	expr, err := triangleAt(32, 32, turn)
	require.NoError(t, err)
	dst := raster(t, expr)
	bot := dst.RGBAAt(16, 24)
	assert.Greater(t, int(bot.R), 180, "half turn bottom=%v", bot)
}

func TestPainterVirt(t *testing.T) {
	p, err := newTrianglePainter(t.Context())
	require.NoError(t, err)
	test.CloseOnCleanup(t, p)
	dst := stdimage.NewRGBA(stdimage.Rect(0, 0, 256, 256))
	require.NoError(t, p.Draw(t.Context(), dst, 0))
	proc, err := process.NewProcess(int32(os.Getpid()))
	require.NoError(t, err)
	before, err := proc.MemoryInfo()
	require.NoError(t, err)
	for i := range 40 {
		require.NoError(t, p.Draw(t.Context(), dst, float64(i)/40))
	}
	after, err := proc.MemoryInfo()
	require.NoError(t, err)
	grew := int64(after.VMS) - int64(before.VMS)
	t.Logf("VMS %d -> %d (%+d) RSS %d -> %d over 40 painter frames", before.VMS, after.VMS, grew, before.RSS, after.RSS)
	require.Less(t, grew, int64(128<<20), "virtual size grew %d bytes", grew)
}

func TestPainterHeap(t *testing.T) {
	p, err := newTrianglePainter(t.Context())
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
	p, err := newTrianglePainter(t.Context())
	require.NoError(t, err)
	test.CloseOnCleanup(t, p)
	dst := stdimage.NewRGBA(stdimage.Rect(0, 0, 32, 32))
	require.NoError(t, p.Draw(t.Context(), dst, 0))
	require.NoError(t, p.Draw(t.Context(), dst, 0.5))
	bot := dst.RGBAAt(16, 24)
	assert.Greater(t, int(bot.R), 180, "painter half turn=%v", bot)
}

func TestPainterDrawAllocs(t *testing.T) {
	p, err := newTrianglePainter(t.Context())
	require.NoError(t, err)
	test.CloseOnCleanup(t, p)
	dst := stdimage.NewRGBA(stdimage.Rect(0, 0, 32, 32))
	require.NoError(t, p.Draw(t.Context(), dst, 0))
	gpu := p.evaluator
	p.evaluator = ndarray.CPU
	require.NoError(t, p.Draw(t.Context(), dst, 0))
	defer func() { p.evaluator = gpu }()
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	n := testing.AllocsPerRun(30, func() {
		if err := p.Draw(t.Context(), dst, 0.25); err != nil {
			panic(err)
		}
	})
	require.Equal(t, 0.0, n)
}

func TestTriangleExec(t *testing.T) {
	if _, err := vulkan.List(t.Context()); err != nil {
		t.Skip(err)
	}
	evaluator, err := ndarray.Open(t.Context())
	require.NoError(t, err)
	test.CloseOnCleanup(t, evaluator)
	expr, err := triangleAt(32, 32, nil)
	require.NoError(t, err)
	cpu := make([]float32, expr.Size())
	require.NoError(t, expr.Eval(t.Context(), ndarray.CPU, cpu))
	gpu := make([]float32, expr.Size())
	require.NoError(t, expr.Eval(t.Context(), evaluator, gpu))
	require.Equal(t, cpu, gpu)
}
