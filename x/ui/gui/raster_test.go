package gui

import (
	"testing"

	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRasterPaintsTensor(t *testing.T) {
	shape := ndarray.Shape{1, 1, 4}
	channel := ndarray.Coord(2, shape)
	source := channel.Equal(ndarray.Const(int32(0))).Where(ndarray.Const(float32(210)), ndarray.Const(float32(20)))
	picture, err := NewPicture()
	require.NoError(t, err)
	pixels, err := picture.Render(&Raster{Pixels: source}, Size{4, 4})
	require.NoError(t, err)
	out := make([]uint8, 4*4*4)
	require.NoError(t, pixels.Eval(t.Context(), ndarray.CPU, out))
	assert.Equal(t, uint8(210), out[0])
	assert.Equal(t, uint8(20), out[1])
	assert.Equal(t, uint8(210), out[(3*4+3)*4])
}

func TestRasterBytesVaryAcrossFrame(t *testing.T) {
	shape := ndarray.Shape{1, 1, 4}
	source := ndarray.Coord(1, shape).Cast[float32]()
	picture, err := NewPicture()
	require.NoError(t, err)
	picture.raster = source
	buffer, err := picture.rasterBytes(t.Context(), ndarray.CPU, 4, 4)
	require.NoError(t, err)
	require.Len(t, buffer, 4*4*4)
	assert.Equal(t, uint8(0), buffer[0])
	assert.NotEqual(t, buffer[0], buffer[3*4])
}

func TestRasterUnderFill(t *testing.T) {
	shape := ndarray.Shape{1, 1, 4}
	channel := ndarray.Coord(2, shape)
	source := channel.Equal(ndarray.Const(int32(0))).Where(ndarray.Const(float32(200)), ndarray.Const(float32(0)))
	root := &Stack{Children: []Node{
		&Raster{Pixels: source},
		&Box{Width: 2, Height: 2, Fill: &Color{0, 0, 255, 255}},
	}}
	picture, err := NewPicture()
	require.NoError(t, err)
	pixels, err := picture.Render(root, Size{4, 4})
	require.NoError(t, err)
	out := make([]uint8, 4*4*4)
	require.NoError(t, pixels.Eval(t.Context(), ndarray.CPU, out))
	corner := out[0:4]
	assert.Equal(t, uint8(0), corner[0])
	assert.Equal(t, uint8(0), corner[1])
	assert.Equal(t, uint8(255), corner[2])
	far := (3*4 + 3) * 4
	assert.Equal(t, uint8(200), out[far])
}
