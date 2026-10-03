package present

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompositeOpaqueFillCoversItsRect(t *testing.T) {
	frame, err := Composite(instance(0, 0, 2, 2, 255, 0, 0, 255, 0, 0, 0, 0, 0), nil, nil, 4, 4)
	require.NoError(t, err)
	require.Equal(t, []byte{255, 0, 0, 255}, frame[0:4])
	require.Equal(t, []byte{255, 0, 0, 255}, frame[4:8])
	require.Equal(t, []byte{0, 0, 0, 255}, frame[8:12])
	require.Equal(t, []byte{0, 0, 0, 255}, frame[len(frame)-4:])
}

func TestCompositeRoundsTheCornerOff(t *testing.T) {
	frame, err := Composite(instance(0, 0, 10, 10, 0, 255, 0, 255, 8, 0, 0, 0, 0), nil, nil, 10, 10)
	require.NoError(t, err)
	require.Equal(t, []byte{0, 0, 0, 255}, pixel(frame, 10, 0, 0))
	require.Equal(t, []byte{0, 255, 0, 255}, pixel(frame, 10, 5, 5))
}

func TestCompositeClips(t *testing.T) {
	frame, err := Composite(instance(0, 0, 4, 4, 0, 0, 255, 255, 0, 1, 0, 2, 4), nil, nil, 4, 4)
	require.NoError(t, err)
	require.Equal(t, []byte{0, 0, 0, 255}, pixel(frame, 4, 0, 1))
	require.Equal(t, []byte{0, 0, 255, 255}, pixel(frame, 4, 1, 1))
}

func TestCompositePaintsUnderThenInk(t *testing.T) {
	under := solid(4, 4, 0, 0, 255, 255)
	ink := make([]byte, 4*4*4)
	binary.LittleEndian.PutUint32(ink[0:4], 255) // red, alpha 0, treated as opaque
	frame, err := Composite(nil, under, ink, 4, 4)
	require.NoError(t, err)
	require.Equal(t, []byte{255, 0, 0, 255}, pixel(frame, 4, 0, 0))
	require.Equal(t, []byte{0, 0, 255, 255}, pixel(frame, 4, 3, 3))
}

func TestCompositeRejectsAShortInstance(t *testing.T) {
	_, err := Composite([]byte{1, 2, 3}, nil, nil, 2, 2)
	require.ErrorIs(t, err, ErrSize)
	_, err = Composite(nil, nil, nil, 0, 2)
	require.ErrorIs(t, err, ErrSize)
}

func instance(x, y, w, h, r, g, b, a, radius, clipX, clipY, clipW, clipH float32) []byte {
	vals := [...]float32{x, y, w, h, r, g, b, a, radius, clipX, clipY, clipW, clipH, 0, 0, 0}
	raw := make([]byte, InstanceStride)
	for i, v := range vals {
		binary.LittleEndian.PutUint32(raw[i*4:], math.Float32bits(v))
	}
	return raw
}

func solid(w, h int, r, g, b, a byte) []byte {
	raw := make([]byte, w*h*4)
	for i := 0; i < len(raw); i += 4 {
		raw[i] = r
		raw[i+1] = g
		raw[i+2] = b
		raw[i+3] = a
	}
	return raw
}

func pixel(frame []byte, width, x, y int) []byte {
	i := (y*width + x) * 4
	return frame[i : i+4]
}
