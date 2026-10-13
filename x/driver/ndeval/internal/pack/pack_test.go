package pack

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/ndarray"
)

func TestBytes(t *testing.T) {
	require.Equal(t, 12, Bytes(ndarray.U8, 3))
	require.Equal(t, 3, Bytes(ndarray.F32, 3))
}

func TestWiden(t *testing.T) {
	raw := []byte{1, 2}
	data, next := Widen(ndarray.F32, raw, []byte{9})
	require.Equal(t, []byte{1, 2}, data)
	require.Equal(t, []byte{9}, next)
	data, next = Widen(ndarray.U8, raw, nil)
	require.Len(t, data, 8)
	require.Len(t, next, 8)
	require.Equal(t, byte(1), data[0])
	require.Equal(t, byte(2), data[4])
	require.Same(t, &data[0], &next[0])
	dirty := []byte{7, 7, 7, 7, 7, 7, 7, 7, 7}
	data, next = Widen(ndarray.U8, raw, dirty)
	require.GreaterOrEqual(t, cap(next), 9)
	require.Equal(t, byte(0), data[1])
	require.Equal(t, byte(0), data[5])
}
