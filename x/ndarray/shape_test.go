package ndarray

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWindowLengthFloor(t *testing.T) {
	require.Equal(t, 2, WindowLength(5, 0, 0, 3, 2))
	require.Equal(t, 0, WindowLength(2, 0, 0, 3, 2))
	require.Equal(t, 1, WindowLength(3, 0, 0, 3, 1))
}

func TestShapeRankSize(t *testing.T) {
	s := Shape{2, 3, 4}
	require.Equal(t, 3, s.Rank())
	require.Equal(t, 24, s.Size())
	require.Equal(t, "[2 3 4]", s.String())
	require.True(t, s.Equal(Shape{2, 3, 4}))
	require.False(t, s.Equal(Shape{2, 3}))
}

func TestShapeInfer(t *testing.T) {
	got, err := Shape{2, -1}.Infer(6)
	require.NoError(t, err)
	require.Equal(t, Shape{2, 3}, got)

	got, err = Shape{-1}.Infer(6)
	require.NoError(t, err)
	require.Equal(t, Shape{6}, got)

	_, err = Shape{-1, -1}.Infer(6)
	require.ErrorIs(t, err, ErrShape)

	_, err = Shape{5}.Infer(6)
	require.ErrorIs(t, err, ErrSize)

	_, err = Shape{-2}.Infer(6)
	require.ErrorIs(t, err, ErrShape)
}

func TestShapeClone(t *testing.T) {
	s := Shape{2, 3}
	c := s.Clone()
	c[0] = 9
	require.Equal(t, Shape{2, 3}, s)
}
