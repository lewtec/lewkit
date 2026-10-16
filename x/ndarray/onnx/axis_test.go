package onnx

import (
	"testing"

	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/stretchr/testify/require"
)

func TestAxisIndex(t *testing.T) {
	t.Parallel()
	got, err := axisIndex(-1, 4)
	require.NoError(t, err)
	require.Equal(t, 3, got)

	got, err = axisIndex(0, 4)
	require.NoError(t, err)
	require.Equal(t, 0, got)

	got, err = axisIndex(3, 4)
	require.NoError(t, err)
	require.Equal(t, 3, got)

	_, err = axisIndex(4, 4)
	require.ErrorIs(t, err, ndarray.ErrShape)

	_, err = axisIndex(-5, 4)
	require.ErrorIs(t, err, ndarray.ErrShape)

	_, err = axisIndex(0, 0)
	require.ErrorIs(t, err, ndarray.ErrShape)
}

func TestNegativeAxisMatchesPositive(t *testing.T) {
	t.Parallel()
	x, err := ndarray.New([]float32{1, 2, 3, 4}, ndarray.Shape{2, 2})
	require.NoError(t, err)
	values := map[string]*ndarray.Tensor[float32]{"x": x}
	neg, err := applySoftmax(values, Node{Inputs: []string{"x"}, Attributes: []Attribute{{Name: "axis", Integer: -1}}})
	require.NoError(t, err)
	pos, err := applySoftmax(values, Node{Inputs: []string{"x"}, Attributes: []Attribute{{Name: "axis", Integer: 1}}})
	require.NoError(t, err)
	negData := make([]float32, 4)
	posData := make([]float32, 4)
	require.NoError(t, neg.Eval(t.Context(), ndarray.CPU, negData))
	require.NoError(t, pos.Eval(t.Context(), ndarray.CPU, posData))
	require.Equal(t, posData, negData)

	_, err = applySoftmax(values, Node{Inputs: []string{"x"}, Attributes: []Attribute{{Name: "axis", Integer: 2}}})
	require.ErrorIs(t, err, ndarray.ErrShape)
}

func TestShapeAxisHelpers(t *testing.T) {
	t.Parallel()
	out, err := unsqueezeShape(ndarray.Shape{2, 3}, []int64{-1})
	require.NoError(t, err)
	require.Equal(t, ndarray.Shape{2, 3, 1}, out)

	out, err = squeezeShape(ndarray.Shape{2, 1, 3}, []int64{-2})
	require.NoError(t, err)
	require.Equal(t, ndarray.Shape{2, 3}, out)

	_, err = squeezeShape(ndarray.Shape{2, 3}, []int64{5})
	require.ErrorIs(t, err, ndarray.ErrShape)
	_, err = squeezeShape(ndarray.Shape{2, 3}, []int64{0})
	require.ErrorIs(t, err, ndarray.ErrShape)
}
