package render

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/image/convert/svg"
	"github.com/stretchr/testify/require"
)

func TestDecodeEmptyFile(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(filepath.Join("..", "testdata", "svg", "empty.svg"))
	require.NoError(t, err)
	img, err := Decode(raw)
	require.NoError(t, err)
	require.Equal(t, 256, img.Bounds().Dx())
	require.Equal(t, 256, img.Bounds().Dy())
}

func TestRenderEmptyIsZeros(t *testing.T) {
	t.Parallel()
	img, err := Render(svg.NewDocument(256, 256))
	require.NoError(t, err)
	require.Equal(t, img.Rect, img.Bounds())
	require.Equal(t, 256, img.Rect.Dx())
	require.Equal(t, 256, img.Rect.Dy())
	for _, b := range img.Pix {
		require.Equal(t, byte(0), b)
	}
}

func TestRenderViewBoxIdentity(t *testing.T) {
	t.Parallel()
	d := svg.NewDocument(256, 256).WithViewBox(0, 0, 256, 256)
	_, err := Render(d)
	require.NoError(t, err)
}

func TestRenderRejectsBadCanvas(t *testing.T) {
	t.Parallel()
	_, err := Render(svg.NewDocument(256.5, 256))
	require.Error(t, err)
}

func TestRenderLinearLerp(t *testing.T) {
	t.Parallel()
	g := svg.NewLinearFill(0, 8, 64, 8,
		color.NRGBA{R: 255, A: 255},
		color.NRGBA{A: 255},
	)
	d := svg.NewDocument(64, 16).Append(
		svg.NewRect().WithWidth(64).WithHeight(16).WithLinearFill(g).Node(),
	)
	img, err := Render(d)
	require.NoError(t, err)
	left := img.NRGBAAt(0, 8)
	require.GreaterOrEqual(t, left.R, uint8(240))
	require.Equal(t, uint8(0), left.G)
	require.Equal(t, uint8(0), left.B)
	require.Equal(t, uint8(255), left.A)
	right := img.NRGBAAt(63, 8)
	require.LessOrEqual(t, right.R, uint8(16))
	require.Equal(t, uint8(0), right.G)
	require.Equal(t, uint8(0), right.B)
	require.Equal(t, uint8(255), right.A)
	mid := img.NRGBAAt(32, 8)
	require.GreaterOrEqual(t, mid.R, uint8(100))
	require.LessOrEqual(t, mid.R, uint8(160))
	require.Equal(t, uint8(0), mid.G)
	require.Equal(t, uint8(0), mid.B)
}

func TestRenderFilledRectDoesNotPanic(t *testing.T) {
	t.Parallel()
	d := svg.NewDocument(256, 256).Append(
		svg.NewRect().WithWidth(100).WithHeight(80).WithX(10).WithY(10).
			WithFill(color.NRGBA{R: 255, A: 255}).Node(),
	)
	img, err := Render(d)
	require.NoError(t, err)
	// Interior sample; not an oracle. Just prove paint happened.
	c := img.NRGBAAt(50, 50)
	require.False(t, c.R == 0 && c.A == 0)
}

func TestCircleEdgeCount(t *testing.T) {
	p, ok := flattenEllipse(0, 0, 10, 10)
	require.True(t, ok)
	t.Logf("segs=%d bounds=(%v,%v)-(%v,%v)", len(p.segs), p.minX, p.minY, p.maxX, p.maxY)
	e := buildLineEdges(p, supersampleShift)
	t.Logf("edges=%d", len(e))
	for i, ed := range e {
		t.Logf("edge[%d] firstY=%d lastY=%d x=%d dx=%d wind=%d cub=%v", i, ed.firstY, ed.lastY, ed.x, ed.dx, ed.winding, ed.cub != nil)
	}
	require.GreaterOrEqual(t, len(e), 2)
}

func TestAcquireDoesNotBlockWhenBusy(t *testing.T) {
	n := workerN()*2 + 2
	pix := make([]*pixmap, n)
	imgs := make([]*image.NRGBA, n)
	done := make(chan struct{})
	go func() {
		for i := 0; i < n; i++ {
			pix[i] = acquirePixmap(8, 8)
			imgs[i] = acquireImage(8, 8)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		require.Fail(t, "pixmap or image acquire blocked with the cache full")
	}
	for _, p := range pix {
		releasePixmap(p)
	}
	for _, img := range imgs {
		Release(img)
	}
}

func TestFillPathSmallCircle(t *testing.T) {
	p, ok := flattenEllipse(0, 0, 10, 10)
	require.True(t, ok)
	pm := newPixmap(256, 256)
	defer releasePixmap(pm)
	fillPath(pm, p, true, color.NRGBA{A: 255}, 255)
	img := pm.toNRGBA()
	defer Release(img)
	var nz int
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] != 0 {
			nz++
		}
	}
	require.NotZero(t, nz)
	t.Logf("nonzero=%d a(5,5)=%d", nz, img.NRGBAAt(5, 5).A)
}

func TestNestedCircleFillsOrigin(t *testing.T) {
	t.Parallel()
	d := svg.NewDocument(256, 256).Append(
		svg.NewGroup().Append(svg.NewCircle().WithR(10).Node()).Node(),
	)
	img, err := Render(d)
	require.NoError(t, err)
	require.NotZero(t, img.NRGBAAt(5, 5).A)
	require.NotZero(t, img.NRGBAAt(0, 0).A)
}

func TestAlphaRunsAddPastWidthDoesNotHang(t *testing.T) {
	a := newAlphaRuns(10)
	done := make(chan struct{})
	go func() {
		a.add(0, 0, 1000, 0, 64, 0)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		require.Fail(t, "alphaRuns.add hung on middleCount > width")
	}
}

func TestBlitHPastBBoxDoesNotHang(t *testing.T) {
	// walkEdges blits open winding to the canvas clip; runs are bbox-sized.
	pm := newPixmap(64, 64)
	defer releasePixmap(pm)
	real := &solidBlitter{pm: pm, pr: 255, pa: 255}
	sb := newSuperBlitter(20, 20, 40, 40, 64, 64, real)
	require.NotNil(t, sb)
	done := make(chan struct{})
	go func() {
		sb.blitH(0, uint32(20<<supersampleShift), 64<<supersampleShift)
		sb.flush()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		require.Fail(t, "blitH hung on span wider than bbox")
	}
}

func TestBlitAntiHMatchesBlend(t *testing.T) {
	const w, h = 12, 3
	runs := make([]uint16, 16)
	alpha := make([]uint8, 16)
	runs[0], alpha[0] = 2, 0
	runs[2], alpha[2] = 4, 255
	runs[6], alpha[6] = 5, 100
	runs[11] = 0
	for _, pa := range []uint8{255, 128, 1} {
		slow := newPixmap(w, h)
		fast := newPixmap(w, h)
		for i := range slow.pix {
			slow.pix[i] = uint8(i * 17)
			fast.pix[i] = uint8(i * 17)
		}
		// pr/pg/pb/pa are the premul source the blitter already holds.
		bSlow := &solidBlitter{pm: slow, pr: 20, pg: 180, pb: 90, pa: pa}
		bFast := &solidBlitter{pm: fast, pr: 20, pg: 180, pb: 90, pa: pa}
		oracleAnti(bSlow, 8, 1, alpha, runs)
		bFast.blitAntiH(8, 1, alpha, runs)
		bFast.blitH(0, 0, 100)
		oracleH(bSlow, 0, 0, 100)
		require.Equal(t, string(slow.pix), string(fast.pix))
		releasePixmap(slow)
		releasePixmap(fast)
	}
}

func oracleAnti(b *solidBlitter, x, y uint32, alpha []uint8, runs []uint16) {
	i := 0
	px := x
	for {
		n := runs[i]
		if n == 0 {
			return
		}
		a := alpha[i]
		if a != 0 {
			sr, sg, sb, sa := scalePremul(b.pr, b.pg, b.pb, b.pa, a)
			for k := uint16(0); k < n; k++ {
				b.pm.blend(int(px+uint32(k)), int(y), sr, sg, sb, sa)
			}
		}
		px += uint32(n)
		i += int(n)
	}
}

func oracleH(b *solidBlitter, x, y, width uint32) {
	for i := uint32(0); i < width; i++ {
		b.pm.blend(int(x+i), int(y), b.pr, b.pg, b.pb, b.pa)
	}
}

func TestPremultiplyU8(t *testing.T) {
	t.Parallel()
	require.Equal(t, uint8(255), premultiplyU8(255, 255))
	require.Equal(t, uint8(0), premultiplyU8(255, 0))
	require.Equal(t, uint8(0), demultiplyU8(0, 0))
}
