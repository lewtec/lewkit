package gui

import (
	"testing"

	_ "github.com/lewtec/lewkit/x/driver/ndeval"
	"github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestPictureTileGPUMatchesCPU(t *testing.T) {
	if _, err := vulkan.List(t.Context()); err != nil {
		t.Skip(err)
	}
	evaluator, err := ndarray.Open(t.Context())
	require.NoError(t, err)
	test.CloseOnCleanup(t, evaluator)
	picture, err := NewPicture()
	require.NoError(t, err)
	red := Color{220, 40, 40, 255}
	blue := Color{40, 40, 220, 255}
	root := &Stack{Children: []Node{
		&Positioned{X: 0, Y: 0, Child: &Box{Width: 3, Height: 3, Fill: &red, Radius: 1}},
		&Positioned{X: 5, Y: 5, Child: &Box{Width: 3, Height: 3, Fill: &blue, Radius: 1}},
	}}
	view, err := picture.Render(root, Size{8, 8})
	require.NoError(t, err)
	cpu := make([]uint8, 8*8*4)
	gpu := make([]uint8, len(cpu))
	require.NoError(t, view.Eval(t.Context(), ndarray.CPU, cpu))
	require.NoError(t, view.Eval(t.Context(), evaluator, gpu))
	require.Equal(t, cpu, gpu)
}
