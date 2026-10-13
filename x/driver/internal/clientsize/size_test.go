package clientsize

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefault(t *testing.T) {
	invalid := errors.New("invalid size")
	w, h, err := Default(0, 0, invalid)
	require.NoError(t, err)
	require.Equal(t, 640, w)
	require.Equal(t, 480, h)
	w, h, err = Default(10, 20, invalid)
	require.NoError(t, err)
	require.Equal(t, 10, w)
	require.Equal(t, 20, h)
	_, _, err = Default(-1, 1, invalid)
	require.ErrorIs(t, err, invalid)
}
