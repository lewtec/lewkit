package ndarray

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
