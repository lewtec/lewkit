package gui

import (
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBoxExpands(t *testing.T) {
	b := &Box{Fill: &Color{255, 0, 0, 255}}
	sz := b.Layout(Tight(40, 20))
	assert.Equal(t, Size{40, 20}, sz)
}

func TestBoxAlignsChild(t *testing.T) {
	label := &Text{Value: "-"}
	b := &Box{Width: 56, Height: 56, Align: Alignment{0.5, 0.5}, Child: label}
	_ = b.Layout(Tight(56, 56))
	out := &painter{}
	b.Paint(Offset{}, Rect{0, 0, 56, 56}, out)
	require.NotEmpty(t, out.texts)
	assert.Greater(t, out.texts[0].box.X, float32(4))
	assert.Greater(t, out.texts[0].box.Y, float32(4))
}

func TestBoxSpacerDoesNotFillCross(t *testing.T) {
	s := &Box{Width: 24}
	sz := s.Layout(BoxConstraints{MaxWidth: 100, MaxHeight: 80})
	assert.Equal(t, Size{24, 0}, sz)
}

func TestRowPacksAndCenters(t *testing.T) {
	a := &Box{Width: 10, Height: 8, Fill: &Color{1, 0, 0, 255}}
	gap := &Box{Width: 4}
	b := &Box{Width: 10, Height: 8, Fill: &Color{2, 0, 0, 255}}
	row := Row(a, gap, b)
	outer := &Box{Align: Alignment{0.5, 0.5}, Child: row}
	sz := outer.Layout(Tight(100, 40))
	assert.Equal(t, Size{100, 40}, sz)
	assert.Equal(t, Size{24, 8}, row.size)
	out := &painter{}
	outer.Paint(Offset{}, Rect{0, 0, 100, 40}, out)
	require.GreaterOrEqual(t, len(out.draws), 2)
	assert.InDelta(t, 38, out.draws[0].X, 1)
	assert.InDelta(t, 16, out.draws[0].Y, 1)
}

func TestRowRootFillsTight(t *testing.T) {
	a := &Box{Width: 10, Height: 8, Fill: &Color{1, 0, 0, 255}}
	row := Row(a)
	sz := row.Layout(Tight(100, 40))
	assert.Equal(t, Size{100, 40}, sz)
}

func TestFlexSplitsExpanded(t *testing.T) {
	left := &Box{Fill: &Color{255, 0, 0, 255}}
	right := &Box{Fill: &Color{0, 255, 0, 255}}
	row := &Flex{Axis: Horizontal, Children: []FlexChild{Expanded(left), Expanded(right)}}
	sz := row.Layout(Tight(100, 10))
	assert.Equal(t, Size{100, 10}, sz)
	assert.Equal(t, Size{50, 10}, left.size)
	assert.Equal(t, Size{50, 10}, right.size)
	assert.Equal(t, float32(0), row.Children[0].position)
	assert.Equal(t, float32(50), row.Children[1].position)
}

func TestColumnStacks(t *testing.T) {
	a := &Box{Width: 10, Height: 8, Fill: &Color{1, 0, 0, 255}}
	b := &Box{Width: 10, Height: 8, Fill: &Color{2, 0, 0, 255}}
	col := Column(a, b)
	sz := col.Layout(Tight(10, 20))
	assert.Equal(t, Size{10, 20}, sz)
	assert.Equal(t, float32(0), col.Children[0].position)
	assert.Equal(t, float32(8), col.Children[1].position)
}

func TestStackPositions(t *testing.T) {
	child := &Box{Width: 4, Height: 4, Fill: &Color{255, 255, 255, 255}}
	st := &Stack{Children: []Node{&Positioned{X: 3, Y: 5, Child: child}}}
	sz := st.Layout(Tight(20, 20))
	assert.Equal(t, Size{20, 20}, sz)
	out := &painter{}
	st.Paint(Offset{}, Rect{0, 0, 20, 20}, out)
	require.Len(t, out.draws, 1)
	assert.Equal(t, float32(3), out.draws[0].X)
	assert.Equal(t, float32(5), out.draws[0].Y)
}

func raster(t *testing.T, root Node, w, h int) []uint8 {
	t.Helper()
	p, err := NewPicture(8)
	require.NoError(t, err)
	pixels, err := p.Render(root, Size{float32(w), float32(h)})
	require.NoError(t, err)
	out := make([]uint8, w*h*4)
	require.NoError(t, pixels.Eval(t.Context(), ndarray.CPU, out))
	return out
}

func at(pix []uint8, w, x, y int) color.RGBA {
	i := (y*w + x) * 4
	return color.RGBA{pix[i], pix[i+1], pix[i+2], pix[i+3]}
}

func TestPaintFillAndAlpha(t *testing.T) {
	root := &Box{Fill: &Color{255, 0, 0, 128}}
	pix := raster(t, root, 8, 8)
	c := at(pix, 8, 3, 3)
	assert.InDelta(t, 128, c.R, 2)
	assert.Equal(t, uint8(0), c.G)
	assert.Equal(t, uint8(255), c.A)
}

func TestPaintRoundCorner(t *testing.T) {
	root := &Box{Fill: &Color{0, 255, 0, 255}, Radius: 20}
	pix := raster(t, root, 40, 40)
	outside := at(pix, 40, 0, 0)
	inside := at(pix, 40, 20, 20)
	assert.Equal(t, uint8(0), outside.G)
	assert.InDelta(t, 255, inside.G, 2)
}

func TestPaintStackTransparent(t *testing.T) {
	root := &Stack{Children: []Node{
		&Box{Fill: &Color{255, 0, 0, 255}},
		&Positioned{X: 0, Y: 0, Child: &Box{Width: 8, Height: 8, Fill: &Color{0, 0, 255, 128}}},
	}}
	pix := raster(t, root, 8, 8)
	c := at(pix, 8, 2, 2)
	assert.Greater(t, c.R, uint8(80))
	assert.Greater(t, c.B, uint8(80))
}

func TestWrapShift(t *testing.T) {
	assert.InDelta(t, 10, wrapShift(70+40, 100), 0.01)
	assert.InDelta(t, 90, wrapShift(10-20, 100), 0.01)
}

func TestMarqueeTreeReused(t *testing.T) {
	m, err := NewMarquee()
	require.NoError(t, err)
	m.size = image.Pt(80, 200)
	a := m.View()
	m.offset = 40
	b := m.View()
	require.Same(t, a, b)
	m.size = image.Pt(80, 400)
	c := m.View()
	require.NotSame(t, a, c)
}

func TestMarqueeLoops(t *testing.T) {
	m, err := NewMarquee()
	require.NoError(t, err)
	m.size = image.Pt(80, 120)
	ya := barYs(t, m)
	m.offset = 160
	yb := barYs(t, m)
	require.NotEmpty(t, ya)
	require.Equal(t, len(ya), len(yb))
	assert.Greater(t, yb[0], ya[0])
}

func TestMarqueeFillsViewport(t *testing.T) {
	const w, h = 80, 400
	top, bot := float32(marqueePad), float32(h-marqueePad)
	for _, delta := range []float32{0, 32, 96, 320} {
		m, err := NewMarquee()
		require.NoError(t, err)
		m.size = image.Pt(w, h)
		m.offset = delta
		bars := marqueeBars(t, m, w, h)
		require.NotEmpty(t, bars, "delta %v", delta)
		empty, maxEmpty := 0, 0
		for y := top + 1; y < bot-1; y++ {
			covered := false
			for _, d := range bars {
				if y >= d.Y && y < d.Y+d.Height {
					covered = true
					break
				}
			}
			if covered {
				empty = 0
				continue
			}
			empty++
			if empty > maxEmpty {
				maxEmpty = empty
			}
		}
		assert.LessOrEqual(t, maxEmpty, marqueeGap+1, "delta %v", delta)
	}
}

func TestMarqueeScrollThenTick(t *testing.T) {
	m, err := NewMarquee()
	require.NoError(t, err)
	m.size = image.Pt(80, 200)
	m.lastTick = 100 * time.Millisecond
	next, cmd := m.Update(window.Scroll{Delta: image.Pt(0, -20)})
	require.Same(t, m, next)
	require.Nil(t, cmd)
	assert.InDelta(t, m.wrap(-20), m.offset, 0.01)
	next, cmd = m.Update(TickMsg{Elapsed: 150 * time.Millisecond, Size: m.size})
	require.Same(t, m, next)
	require.NotNil(t, cmd)
	assert.InDelta(t, m.wrap(-20+4), m.offset, 0.01)
}

func TestMarqueeResizeClearsDrag(t *testing.T) {
	m, err := NewMarquee()
	require.NoError(t, err)
	m.size = image.Pt(80, 200)
	m.lastTick = 100 * time.Millisecond
	_, _ = m.Update(window.Pointer{Pos: image.Pt(10, 50), Button: 1, Pressed: true, Buttons: window.ButtonLeft})
	require.True(t, m.dragging)
	offset := m.offset
	_, cmd := m.Update(window.Resize{Size: image.Pt(80, 240)})
	require.Nil(t, cmd)
	assert.False(t, m.dragging)
	next, cmd := m.Update(TickMsg{Elapsed: 200 * time.Millisecond, Period: time.Second / 60, Size: image.Pt(80, 240)})
	require.Same(t, m, next)
	require.NotNil(t, cmd)
	assert.Greater(t, m.offset, offset)
}

func TestMarqueeDrag(t *testing.T) {
	m, err := NewMarquee()
	require.NoError(t, err)
	m.size = image.Pt(80, 200)
	next, cmd := m.Update(window.Pointer{Pos: image.Pt(10, 50), Button: 1, Pressed: true, Buttons: window.ButtonLeft})
	require.Same(t, m, next)
	require.Nil(t, cmd)
	next, cmd = m.Update(window.Pointer{Pos: image.Pt(10, 90), Buttons: window.ButtonLeft})
	require.Same(t, m, next)
	require.Nil(t, cmd)
	assert.InDelta(t, 40, m.offset, 0.01)
	assert.True(t, m.dragging)
	next, cmd = m.Update(window.Pointer{Pos: image.Pt(10, 90), Button: 1, Pressed: false})
	require.Same(t, m, next)
	require.Nil(t, cmd)
	assert.False(t, m.dragging)
}

func barYs(t *testing.T, m *Marquee) []float32 {
	t.Helper()
	var ys []float32
	for _, d := range marqueeBars(t, m, m.size.X, m.size.Y) {
		ys = append(ys, d.Y)
	}
	return ys
}

func marqueeBars(t *testing.T, m *Marquee, w, h int) []Draw {
	t.Helper()
	root := m.View()
	root.Layout(Tight(float32(w), float32(h)))
	out := &painter{}
	root.Paint(Offset{}, Rect{0, 0, float32(w), float32(h)}, out)
	var bars []Draw
	for _, d := range out.draws {
		if d.Alpha > 0 && d.Height <= marqueeItemH+1 && d.Width < float32(w) {
			bars = append(bars, d)
		}
	}
	return bars
}

func TestPictureReuseKernel(t *testing.T) {
	p, err := NewPicture(4)
	require.NoError(t, err)
	root := &Box{Fill: &Color{10, 20, 30, 255}}
	first, err := p.Render(root, Size{6, 6})
	require.NoError(t, err)
	k := first.Kernel()
	second, err := p.Render(root, Size{8, 8})
	require.NoError(t, err)
	assert.Same(t, k, second.Kernel())
	require.Equal(t, 3, k.Bindings())
}

func TestInkLetter(t *testing.T) {
	p, err := NewPicture(2)
	require.NoError(t, err)
	root := &Box{Fill: &Color{0, 0, 0, 255}, Child: &Text{Value: "Hi"}}
	pixels, err := p.Render(root, Size{80, 40})
	require.NoError(t, err)
	out := make([]uint8, 80*40*4)
	require.NoError(t, pixels.Eval(t.Context(), ndarray.CPU, out))
	var lit int
	for i := 0; i < len(out); i += 4 {
		if int(out[i])+int(out[i+1])+int(out[i+2]) > 40 {
			lit++
		}
	}
	assert.Greater(t, lit, 10)
}

func TestCounterButtons(t *testing.T) {
	c, err := NewCounter()
	require.NoError(t, err)
	c.size = image.Pt(400, 200)
	root := c.View()
	require.NotNil(t, root)
	root.Layout(Tight(400, 200))
	root.Paint(Offset{}, Rect{0, 0, 400, 200}, &painter{})
	require.NotNil(t, c.plus)
	pos := image.Pt(int(c.plus.origin.X+c.plus.size.Width/2), int(c.plus.origin.Y+c.plus.size.Height/2))
	next, cmd := c.Update(window.Pointer{Pos: pos, Button: 1, Pressed: true, Buttons: window.ButtonLeft})
	require.Same(t, c, next)
	require.Nil(t, cmd)
	assert.Equal(t, 1, c.count)
	pos = image.Pt(int(c.minus.origin.X+c.minus.size.Width/2), int(c.minus.origin.Y+c.minus.size.Height/2))
	next, cmd = c.Update(window.Pointer{Pos: pos, Button: 1, Pressed: true, Buttons: window.ButtonLeft})
	require.Same(t, c, next)
	require.Nil(t, cmd)
	assert.Equal(t, 0, c.count)
}

func TestNotepadTypes(t *testing.T) {
	n, err := NewNotepad()
	require.NoError(t, err)
	next, cmd := n.Update(window.Key{Rune: 'x', Pressed: true})
	require.Same(t, n, next)
	require.Nil(t, cmd)
	assert.Contains(t, string(n.body), "x")
}
