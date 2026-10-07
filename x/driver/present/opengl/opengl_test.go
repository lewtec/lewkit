package opengl

import (
	"encoding/binary"
	"math"
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/present"
	ffiopengl "github.com/lewtec/lewkit/x/ffi/native/opengl"
	"github.com/stretchr/testify/require"
)

func TestFactoryIdentity(t *testing.T) {
	require.Equal(t, "present_opengl", factory{}.ID())
	require.Equal(t, "OpenGL", factory{}.Name())
	require.Equal(t, 15, factory{}.Weight())
}

func TestAppleIsIncompatible(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "ios" {
		t.Skip()
	}
	err := factory{}.CheckCompatibility(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
}

func TestDrawMatchesComposite(t *testing.T) {
	if err := ffiopengl.Available(); err != nil {
		t.Skip(err)
	}
	screen, err := ffiopengl.OpenOffscreen(10, 10)
	require.NoError(t, err)
	t.Cleanup(func() { _ = screen.Close() })

	cases := []struct {
		name                  string
		w, h                  int
		instances, under, ink []byte
	}{
		{
			name:      "opaque",
			w:         4,
			h:         4,
			instances: instance(0, 0, 2, 2, 255, 0, 0, 255, 0, 0, 0, 0, 0),
		},
		{
			name:      "round",
			w:         10,
			h:         10,
			instances: instance(0, 0, 10, 10, 0, 255, 0, 255, 8, 0, 0, 0, 0),
		},
		{
			name:      "clip",
			w:         4,
			h:         4,
			instances: instance(0, 0, 4, 4, 0, 0, 255, 255, 0, 1, 0, 2, 4),
		},
		{
			name:  "under-ink",
			w:     4,
			h:     4,
			under: solid(4, 4, 0, 0, 255, 255),
			ink:   inkTop(4, 4),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want, err := present.Composite(tc.instances, tc.under, tc.ink, tc.w, tc.h)
			require.NoError(t, err)
			require.NoError(t, screen.Draw(tc.instances, tc.under, tc.ink, tc.w, tc.h))
			got, err := screen.Read()
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}
}

func instance(x, y, w, h, r, g, b, a, radius, clipX, clipY, clipW, clipH float32) []byte {
	vals := [...]float32{x, y, w, h, r, g, b, a, radius, clipX, clipY, clipW, clipH, 0, 0, 0}
	raw := make([]byte, present.InstanceStride)
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

func inkTop(w, h int) []byte {
	raw := make([]byte, w*h*4)
	binary.LittleEndian.PutUint32(raw[0:4], 255)
	return raw
}
