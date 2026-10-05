package main

import (
	"context"
	"image"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/audio_play"
	"github.com/lewtec/lewkit/x/driver/audio_play/mem"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/ui/gui"
	"github.com/lewtec/lewkit/x/ui/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMallardFrame(t *testing.T) {
	const w, h = 498, 280
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	paintDuck(img, mallard(), 0.9)
	var duck, fin, beak int
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.RGBAAt(x, y)
			if c.R == uint8(card.r) && c.G == uint8(card.g) && c.B == uint8(card.b) {
				continue
			}
			duck++
			if c.G > 60 && int(c.G) > int(c.R)+20 && int(c.G) > int(c.B) {
				fin++
			}
			if c.R > 150 && c.G > 40 && c.G < 170 && c.B < 90 {
				beak++
			}
		}
	}
	path := filepath.Join(t.TempDir(), "duck.png")
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, png.Encode(f, img))
	require.NoError(t, f.Close())
	t.Log(path, "duck", duck, "fin", fin, "beak", beak)
	assert.Greater(t, duck, 1500)
	assert.Greater(t, fin, 30)
	assert.Greater(t, beak, 8)
}

func TestOverlayHoldsWhileDuckTurns(t *testing.T) {
	screen, err := newScreen(t.Context())
	require.NoError(t, err)
	screen.size = image.Pt(498, 280)
	require.NoError(t, screen.sample(0))
	require.NotNil(t, screen.pixels)
	before := append([]float32(nil), screen.pixels.Buffer()...)
	lines, reds, rasters, images := cardParts(screen.View())
	assert.Contains(t, lines, "Cyberpunk")
	assert.Contains(t, lines, "Thank you,")
	assert.Contains(t, lines, "CD PROJEKT RED")
	assert.Greater(t, reds, 0)
	assert.Equal(t, 1, rasters)
	assert.Equal(t, 0, images)
	corner := screen.frame.RGBAAt(0, 0)
	assert.Equal(t, uint8(card.r), corner.R)
	assert.Equal(t, uint8(card.g), corner.G)
	assert.Equal(t, uint8(card.b), corner.B)

	require.NoError(t, screen.sample(2.4))
	again, redsAgain, rastersAgain, imagesAgain := cardParts(screen.View())
	assert.Equal(t, lines, again)
	assert.Equal(t, reds, redsAgain)
	assert.Equal(t, rasters, rastersAgain)
	assert.Equal(t, images, imagesAgain)
	assert.NotEqual(t, before, append([]float32(nil), screen.pixels.Buffer()...))
}

func cardParts(root gui.Node) (lines []string, reds, rasters, images int) {
	var walk func(gui.Node)
	walk = func(node gui.Node) {
		switch n := node.(type) {
		case *gui.Stack:
			for _, child := range n.Children {
				walk(child)
			}
		case *gui.Positioned:
			walk(n.Child)
		case *gui.Box:
			if n.Fill != nil && n.Fill.Red > 150 && n.Fill.Green < 80 && n.Fill.Blue < 80 {
				reds++
			}
			walk(n.Child)
		case *gui.Text:
			lines = append(lines, n.Value)
		case *gui.Raster:
			rasters++
		case *gui.Image:
			images++
		}
	}
	walk(root)
	return lines, reds, rasters, images
}

func TestBackdropFillsTheWindow(t *testing.T) {
	screen, err := newScreen(t.Context())
	require.NoError(t, err)
	next, cmd := screen.Update(window.Resize{Size: image.Pt(1200, 700)})
	require.NotNil(t, next)
	assert.Nil(t, cmd)
	require.NotNil(t, screen.pixels)
	assert.True(t, screen.pixels.Shape().Equal(ndarray.Shape{700, 1200, 4}))
	buf := screen.pixels.Buffer()
	corner := (4*1200 + 1180) * 4
	assert.Equal(t, float32(card.r), buf[corner])
	assert.Equal(t, float32(card.g), buf[corner+1])
	assert.Equal(t, float32(card.b), buf[corner+2])
	y, ok := textTop(screen.View(), "Thank you,")
	require.True(t, ok)
	assert.InDelta(t, 228*700/280, y, 1)
}

func textTop(root gui.Node, value string) (float32, bool) {
	var y float32
	found := false
	var walk func(gui.Node, float32)
	walk = func(node gui.Node, top float32) {
		switch n := node.(type) {
		case *gui.Stack:
			for _, child := range n.Children {
				walk(child, top)
			}
		case *gui.Positioned:
			walk(n.Child, top+n.Y)
		case *gui.Box:
			walk(n.Child, top)
		case *gui.Text:
			if n.Value == value {
				y = top
				found = true
			}
		}
	}
	walk(root, 0)
	return y, found
}

func TestPatoFrame(t *testing.T) {
	model, err := pato()
	require.NoError(t, err)
	require.NotNil(t, model.tex)
	require.Greater(t, len(model.i), 1000)
	const w, h = 498, 280
	a := image.NewRGBA(image.Rect(0, 0, w, h))
	started := time.Now()
	paintDuck(a, model, 0.6)
	b := image.NewRGBA(image.Rect(0, 0, w, h))
	paintDuck(b, model, 2.4)
	var colored, diff int
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ca, cb := a.RGBAAt(x, y), b.RGBAAt(x, y)
			if ca.R != uint8(card.r) || ca.G != uint8(card.g) || ca.B != uint8(card.b) {
				colored++
			}
			if ca != cb {
				diff++
			}
		}
	}
	t.Log("paint", time.Since(started), "tex", model.tex.Bounds().Dx(), model.tex.Bounds().Dy(), "colored", colored, "diff", diff)
	assert.Greater(t, colored, 1500)
	assert.Greater(t, diff, 200)
}

func TestSpinAdvancesOnTick(t *testing.T) {
	screen, err := newScreen(t.Context())
	require.NoError(t, err)
	next, cmd := screen.Update(gui.TickMsg{
		Elapsed: time.Second,
		Period:  time.Second / 30,
		Size:    image.Pt(498, 280),
	})
	require.NotNil(t, next)
	require.NotNil(t, cmd)
	sp, ok := world.Read[spin](screen.sim.World)
	require.True(t, ok)
	assert.InDelta(t, turnsPerLoop*2*math.Pi, sp.Yaw, 1e-6)
}

func TestClipLoops(t *testing.T) {
	pcm, format, err := clip()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(pcm), format.Rate/5)

	buf, err := mem.Open(audio_play.Config{Format: format})
	require.NoError(t, err)
	limit := len(pcm) + len(pcm)/2
	err = loopClip(t.Context(), pcm, format, func(context.Context, audio_play.Config) (io.WriteCloser, error) {
		return &stopAfter{w: buf, limit: limit}, nil
	})
	require.ErrorIs(t, err, io.EOF)
	assert.GreaterOrEqual(t, len(buf.PCM()), len(pcm))
}

type stopAfter struct {
	w     io.WriteCloser
	n     int
	limit int
}

func (s *stopAfter) Write(p []byte) (int, error) {
	if s.n >= s.limit {
		return 0, io.EOF
	}
	n, err := s.w.Write(p)
	s.n += n
	return n, err
}

func (s *stopAfter) Close() error { return s.w.Close() }

type clipMark struct{}

func testSession(t *testing.T) (*taskgroup.Session, context.Context) {
	t.Helper()
	sess, ctx := taskgroup.New(t.Context(), taskgroup.Limits{IO: 1, CPU: 1, Internet: 1})
	t.Cleanup(func() { sess.Cancel(context.Canceled) })
	return sess, ctx
}

func TestClipKeepsTheParentContext(t *testing.T) {
	sess, ctx := testSession(t)
	parent, cancelParent := context.WithCancel(context.WithValue(ctx, clipMark{}, "kept"))
	defer cancelParent()
	entered := make(chan struct{})
	clip, stop := startClip(parent, func(ctx context.Context) error {
		if ctx.Value(clipMark{}) != "kept" {
			return io.ErrUnexpectedEOF
		}
		close(entered)
		<-ctx.Done()
		return nil
	})
	defer stop()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("clip did not start")
	}
	cancelParent()
	select {
	case <-clip.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("parent cancel did not reach the clip")
	}
	waitSession(t, sess)
}

func TestWindowReturnStopsTheClip(t *testing.T) {
	sess, ctx := testSession(t)
	entered := make(chan struct{})
	_, stop := startClip(ctx, func(ctx context.Context) error {
		close(entered)
		<-ctx.Done()
		return nil
	})
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("clip did not start")
	}
	stop()
	assert.NoError(t, ctx.Err())
	waitSession(t, sess)
}

func waitSession(t *testing.T, sess *taskgroup.Session) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- sess.Wait() }()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("session still waiting on the clip")
	}
}
