package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStagePaintsSceneThroughTheDeadZone(t *testing.T) {
	root := &Stage{
		Scene: &Box{Fill: &RGB{255, 0, 0, 255}},
		Child: &Box{Fill: &RGB{0, 0, 255, 255}},
	}
	picture, err := NewPicture()
	require.NoError(t, err)
	picture.dead = []Rect{{0, 0, 80, 12}}
	_, err = picture.Render(root, Size{80, 40})
	require.NoError(t, err)
	require.Len(t, picture.fills, 2)
	assert.Equal(t, float32(0), picture.fills[0].Y)
	assert.Equal(t, float32(40), picture.fills[0].Height)
	assert.Equal(t, float32(255), picture.fills[0].Red)
	assert.Equal(t, float32(12), picture.fills[1].Y)
	assert.Equal(t, float32(28), picture.fills[1].Height)
	assert.Equal(t, float32(80), picture.fills[1].Width)
	assert.Equal(t, float32(255), picture.fills[1].Blue)
	assert.Equal(t, Rect{0, 12, 80, 28}, root.Shown())
}

func TestRenderInsetsAPlainRootToTheAliveBox(t *testing.T) {
	picture, err := NewPicture()
	require.NoError(t, err)
	picture.dead = []Rect{{0, 0, 40, 8}}
	_, err = picture.Render(&Box{Fill: &RGB{0, 255, 0, 255}}, Size{40, 20})
	require.NoError(t, err)
	require.Len(t, picture.fills, 1)
	assert.Equal(t, float32(0), picture.fills[0].X)
	assert.Equal(t, float32(8), picture.fills[0].Y)
	assert.Equal(t, float32(40), picture.fills[0].Width)
	assert.Equal(t, float32(12), picture.fills[0].Height)
}

func TestRenderKeepsRasterAcrossTheDeadZone(t *testing.T) {
	raster := &Raster{}
	root := &Stack{Children: []Node{
		raster,
		&Box{Fill: &RGB{0, 0, 255, 255}},
	}}
	picture, err := NewPicture()
	require.NoError(t, err)
	picture.dead = []Rect{{0, 0, 80, 12}}
	_, err = picture.Render(root, Size{80, 40})
	require.NoError(t, err)
	assert.Equal(t, Size{80, 40}, raster.size)
	require.Len(t, picture.fills, 1)
	assert.Equal(t, float32(12), picture.fills[0].Y)
	assert.Equal(t, float32(28), picture.fills[0].Height)
}

func TestStageCopiesScale(t *testing.T) {
	picture, err := NewPicture()
	require.NoError(t, err)
	picture.SetScale(2)
	_, err = picture.Render(&Stage{Child: &Positioned{Child: &Box{
		Width: 10, Height: 8, Fill: &RGB{1, 2, 3, 255},
	}}}, Size{200, 200})
	require.NoError(t, err)
	require.NotEmpty(t, picture.fills)
	assert.Equal(t, float32(20), picture.fills[0].Width)
	assert.Equal(t, float32(16), picture.fills[0].Height)
}

func TestStageShownIsEmptyWhenCovered(t *testing.T) {
	stage := &Stage{Child: &Box{Fill: &RGB{255, 0, 0, 255}}}
	constraints := Tight(8, 8)
	constraints.dead = []Rect{{0, 0, 8, 8}}
	assert.Equal(t, Size{8, 8}, stage.Layout(constraints))
	assert.Equal(t, Rect{}, stage.Shown())
	var bare *Stage
	assert.Equal(t, Rect{}, bare.Shown())
	assert.Equal(t, Size{}, bare.Layout(Tight(8, 8)))
}
