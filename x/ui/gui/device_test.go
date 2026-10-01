package gui

import (
	"context"
	"testing"

	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type captureCanvas struct{ frame Frame }

func (canvas *captureCanvas) Draw(_ context.Context, frame Frame) error {
	canvas.frame = frame
	return nil
}

func TestPlayAttachesFillsToCanvas(t *testing.T) {
	picture, err := NewPicture()
	require.NoError(t, err)
	picture.Fill(Rect{1, 2, 10, 8}, 3, RGB{4, 5, 6, 255}, Rect{0, 0, 20, 20})
	picture.Fill(Rect{12, 2, 6, 8}, 0, RGB{7, 8, 9, 255}, Rect{0, 0, 20, 20})
	canvas := &captureCanvas{}
	err = Play(t.Context(), canvas, Frame{Width: 20, Height: 20, Fills: picture.fills})
	require.NoError(t, err)
	require.Len(t, canvas.frame.Fills, 2)
	assert.Equal(t, float32(1), canvas.frame.Fills[0].X)
	assert.Equal(t, float32(12), canvas.frame.Fills[1].X)
	picture.fills[0].X = 99
	assert.Equal(t, float32(1), canvas.frame.Fills[0].X)
}

func TestPlayRejectsAMissingCanvas(t *testing.T) {
	err := Play(t.Context(), nil, Frame{Width: 2, Height: 2})
	require.ErrorIs(t, err, ErrView)
	err = Play(t.Context(), &captureCanvas{}, Frame{})
	require.ErrorIs(t, err, ndarray.ErrShape)
}
