package gui

import (
	"image/color"
	"testing"

	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBoxExpands(t *testing.T) {
	box := &Box{Fill: &RGB{255, 0, 0, 255}}
	size := box.Layout(Tight(40, 20))
	assert.Equal(t, Size{40, 20}, size)
}

func TestBoxAlignsChild(t *testing.T) {
	label := &Text{Value: "-"}
	box := &Box{Width: 56, Height: 56, Align: Alignment{0.5, 0.5}, Child: label}
	_ = box.Layout(Tight(56, 56))
	picture, err := NewPicture()
	require.NoError(t, err)
	box.Paint(Offset{}, Rect{0, 0, 56, 56}, picture)
	require.NotEmpty(t, picture.texts)
	assert.Greater(t, picture.texts[0].box.X, float32(4))
	assert.Greater(t, picture.texts[0].box.Y, float32(4))
}

func TestBoxSpacerDoesNotFillCross(t *testing.T) {
	spacer := &Box{Width: 24}
	size := spacer.Layout(BoxConstraints{MaxWidth: 100, MaxHeight: 80})
	assert.Equal(t, Size{24, 0}, size)
}

func TestRowPacksAndCenters(t *testing.T) {
	a := &Box{Width: 10, Height: 8, Fill: &RGB{1, 0, 0, 255}}
	gap := &Box{Width: 4}
	box := &Box{Width: 10, Height: 8, Fill: &RGB{2, 0, 0, 255}}
	row := Row(a, gap, box)
	outer := &Box{Align: Alignment{0.5, 0.5}, Child: row}
	size := outer.Layout(Tight(100, 40))
	assert.Equal(t, Size{100, 40}, size)
	assert.Equal(t, Size{24, 8}, row.size)
	picture, err := NewPicture()
	require.NoError(t, err)
	outer.Paint(Offset{}, Rect{0, 0, 100, 40}, picture)
	require.GreaterOrEqual(t, len(picture.fills), 2)
	assert.InDelta(t, 38, picture.fills[0].X, 1)
	assert.InDelta(t, 16, picture.fills[0].Y, 1)
}

func TestRowRootFillsTight(t *testing.T) {
	a := &Box{Width: 10, Height: 8, Fill: &RGB{1, 0, 0, 255}}
	row := Row(a)
	size := row.Layout(Tight(100, 40))
	assert.Equal(t, Size{100, 40}, size)
}

func TestFlexSplitsExpanded(t *testing.T) {
	left := &Box{Fill: &RGB{255, 0, 0, 255}}
	right := &Box{Fill: &RGB{0, 255, 0, 255}}
	row := &Flex{Axis: Horizontal, Children: []FlexChild{Expanded(left), Expanded(right)}}
	size := row.Layout(Tight(100, 10))
	assert.Equal(t, Size{100, 10}, size)
	assert.Equal(t, Size{50, 10}, left.size)
	assert.Equal(t, Size{50, 10}, right.size)
	assert.Equal(t, float32(0), row.Children[0].position)
	assert.Equal(t, float32(50), row.Children[1].position)
}

func TestWithGapCopies(t *testing.T) {
	a := &Box{Width: 4, Height: 4, Fill: &RGB{1, 0, 0, 255}}
	b := &Box{Width: 4, Height: 4, Fill: &RGB{2, 0, 0, 255}}
	row := Row(a, b)
	gapped := WithGap(6, row)
	assert.Equal(t, float32(0), row.Gap)
	assert.Equal(t, float32(6), gapped.Gap)
	assert.Equal(t, CrossCenter, gapped.Cross)
	started := WithCross(CrossStart, gapped)
	assert.Equal(t, CrossCenter, gapped.Cross)
	assert.Equal(t, CrossStart, started.Cross)
}

func TestFlexGapSeparatesChildren(t *testing.T) {
	a := &Box{Width: 10, Height: 8, Fill: &RGB{1, 0, 0, 255}}
	b := &Box{Width: 10, Height: 8, Fill: &RGB{2, 0, 0, 255}}
	row := Row(a, b)
	row.Gap = 4
	size := row.Layout(BoxConstraints{MaxWidth: 100, MaxHeight: 20})
	assert.Equal(t, Size{24, 8}, size)
	assert.Equal(t, float32(0), row.Children[0].position)
	assert.Equal(t, float32(14), row.Children[1].position)
}

func TestColumnStacks(t *testing.T) {
	a := &Box{Width: 10, Height: 8, Fill: &RGB{1, 0, 0, 255}}
	box := &Box{Width: 10, Height: 8, Fill: &RGB{2, 0, 0, 255}}
	column := Column(a, box)
	size := column.Layout(Tight(10, 20))
	assert.Equal(t, Size{10, 20}, size)
	assert.Equal(t, float32(0), column.Children[0].position)
	assert.Equal(t, float32(8), column.Children[1].position)
}

func TestStackPositions(t *testing.T) {
	child := &Box{Width: 4, Height: 4, Fill: &RGB{255, 255, 255, 255}}
	stack := &Stack{Children: []Node{&Positioned{X: 3, Y: 5, Child: child}}}
	size := stack.Layout(Tight(20, 20))
	assert.Equal(t, Size{20, 20}, size)
	picture, err := NewPicture()
	require.NoError(t, err)
	stack.Paint(Offset{}, Rect{0, 0, 20, 20}, picture)
	require.Len(t, picture.fills, 1)
	assert.Equal(t, float32(3), picture.fills[0].X)
	assert.Equal(t, float32(5), picture.fills[0].Y)
}

func raster(t *testing.T, root Node, width, height int) []uint8 {
	t.Helper()
	picture, err := NewPicture()
	require.NoError(t, err)
	pixels, err := picture.Render(root, Size{float32(width), float32(height)})
	require.NoError(t, err)
	out := make([]uint8, width*height*4)
	require.NoError(t, pixels.Eval(t.Context(), ndarray.CPU, out))
	return out
}

func at(pixels []uint8, width, x, y int) color.RGBA {
	i := (y*width + x) * 4
	return color.RGBA{pixels[i], pixels[i+1], pixels[i+2], pixels[i+3]}
}

func TestPaintFillAndAlpha(t *testing.T) {
	root := &Box{Fill: &RGB{255, 0, 0, 128}}
	pixels := raster(t, root, 8, 8)
	sampled := at(pixels, 8, 3, 3)
	assert.InDelta(t, 128, sampled.R, 2)
	assert.Equal(t, uint8(0), sampled.G)
	assert.Equal(t, uint8(255), sampled.A)
}

func TestPaintRoundCorner(t *testing.T) {
	root := &Box{Fill: &RGB{0, 255, 0, 255}, Radius: 20}
	pixels := raster(t, root, 40, 40)
	outside := at(pixels, 40, 0, 0)
	inside := at(pixels, 40, 20, 20)
	assert.Equal(t, uint8(0), outside.G)
	assert.InDelta(t, 255, inside.G, 2)
}

func TestPaintStackTransparent(t *testing.T) {
	root := &Stack{Children: []Node{
		&Box{Fill: &RGB{255, 0, 0, 255}},
		&Positioned{X: 0, Y: 0, Child: &Box{Width: 8, Height: 8, Fill: &RGB{0, 0, 255, 128}}},
	}}
	pixels := raster(t, root, 8, 8)
	sampled := at(pixels, 8, 2, 2)
	assert.Greater(t, sampled.R, uint8(80))
	assert.Greater(t, sampled.B, uint8(80))
}

func TestPictureRecordSkipsKernel(t *testing.T) {
	picture, err := NewPicture()
	require.NoError(t, err)
	picture.recordOnly = true
	pixels, err := picture.Render(&Box{Fill: &RGB{10, 20, 30, 255}}, Size{8, 8})
	require.NoError(t, err)
	assert.Nil(t, pixels)
	require.Len(t, picture.fills, 1)
	assert.InDelta(t, float32(8), picture.fills[0].Width, 0)
	assert.NotZero(t, picture.frameSig())
	assert.Nil(t, picture.params)
}

func TestPictureReuseKernel(t *testing.T) {
	picture, err := NewPicture()
	require.NoError(t, err)
	root := &Box{Fill: &RGB{10, 20, 30, 255}}
	first, err := picture.Render(root, Size{6, 6})
	require.NoError(t, err)
	kernel := first.Kernel()
	second, err := picture.Render(root, Size{8, 8})
	require.NoError(t, err)
	assert.Same(t, kernel, second.Kernel())
	require.Equal(t, 3, kernel.Bindings())
}

func TestPictureGrowsLayers(t *testing.T) {
	picture, err := NewPicture()
	require.NoError(t, err)
	one := &Box{Fill: &RGB{255, 0, 0, 255}}
	first, err := picture.Render(one, Size{8, 8})
	require.NoError(t, err)
	kernel := first.Kernel()
	children := make([]Node, 10)
	for i := range children {
		children[i] = &Box{Fill: &RGB{0, 0, 255, 40}}
	}
	stack := &Stack{Children: children}
	grown, err := picture.Render(stack, Size{8, 8})
	require.NoError(t, err)
	require.NotSame(t, kernel, grown.Kernel())
	again, err := picture.Render(stack, Size{8, 8})
	require.NoError(t, err)
	assert.Same(t, grown.Kernel(), again.Kernel())
	pixels := make([]uint8, 8*8*4)
	require.NoError(t, grown.Eval(t.Context(), ndarray.CPU, pixels))
	sampled := at(pixels, 8, 4, 4)
	assert.Greater(t, sampled.B, uint8(80))
}

func TestInkLetter(t *testing.T) {
	picture, err := NewPicture()
	require.NoError(t, err)
	root := &Box{Fill: &RGB{0, 0, 0, 255}, Child: &Text{Value: "Hi"}}
	pixels, err := picture.Render(root, Size{80, 40})
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
