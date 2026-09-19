package vulkan

import (
	"encoding/binary"
	"testing"

	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSmokeSPIRVMagic(t *testing.T) {
	b := SmokeSPIRV()
	require.GreaterOrEqual(t, len(b), 20)
	require.Equal(t, uint32(0x07230203), binary.LittleEndian.Uint32(b[:4]))
	require.Equal(t, 0, len(b)%4)
}

func TestOpenInvalid(t *testing.T) {
	d, err := Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	test.CloseOnCleanup(t, d)
	_, err = d.Buffer(0)
	require.ErrorIs(t, err, ErrSize)
	_, err = d.Shader(t.Context(), nil, 1)
	require.ErrorIs(t, err, ErrShader)
	_, err = d.Shader(t.Context(), SmokeSPIRV(), 0)
	require.ErrorIs(t, err, ErrShader)
}

func TestHostRoundTrip(t *testing.T) {
	d, err := Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	test.CloseOnCleanup(t, d)
	buf, err := d.Buffer(16)
	require.NoError(t, err)
	test.CloseOnCleanup(t, buf)
	in := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	require.NoError(t, buf.Write(in))
	got := make([]byte, 16)
	require.NoError(t, buf.Read(got))
	require.Equal(t, in, got)
}

func TestGLSLShader(t *testing.T) {
	d, err := Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	test.CloseOnCleanup(t, d)
	src := []byte(`#version 450
layout(local_size_x = 1) in;
layout(set = 0, binding = 0) buffer Data { uint v; } data;
void main() { data.v = 2u; }
`)
	buf, err := d.Buffer(4)
	require.NoError(t, err)
	test.CloseOnCleanup(t, buf)
	require.NoError(t, buf.Write(make([]byte, 4)))
	sh, err := d.Shader(t.Context(), src, 1)
	require.NoError(t, err)
	test.CloseOnCleanup(t, sh)
	require.NoError(t, d.Run(sh, 1, 1, 1, buf))
	got := make([]byte, 4)
	require.NoError(t, buf.Read(got))
	require.Equal(t, uint32(2), binary.LittleEndian.Uint32(got))
}

func TestDispatch(t *testing.T) {
	d, err := Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	test.CloseOnCleanup(t, d)
	require.NotEmpty(t, d.Name())

	buf, err := d.Buffer(4)
	require.NoError(t, err)
	test.CloseOnCleanup(t, buf)
	require.NoError(t, buf.Write(make([]byte, 4)))

	sh, err := d.Shader(t.Context(), SmokeSPIRV(), 1)
	require.NoError(t, err)
	test.CloseOnCleanup(t, sh)
	require.NoError(t, d.Run(sh, 1, 1, 1, buf))

	got := make([]byte, 4)
	require.NoError(t, buf.Read(got))
	have := binary.LittleEndian.Uint32(got)
	if have != 2 {
		t.Fatalf("got %d, want 2 (device %s)", have, d.Name())
	}
}

func TestRunBufferCount(t *testing.T) {
	d, err := Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	test.CloseOnCleanup(t, d)
	buf, err := d.Buffer(256)
	require.NoError(t, err)
	test.CloseOnCleanup(t, buf)
	sh, err := d.Shader(t.Context(), SmokeSPIRV(), 1)
	require.NoError(t, err)
	test.CloseOnCleanup(t, sh)
	err = d.Run(sh, 1, 1, 1)
	require.ErrorIs(t, err, ErrShader)
	assert.Contains(t, err.Error(), "want 1")
}

func TestWriteTooLong(t *testing.T) {
	d, err := Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	test.CloseOnCleanup(t, d)
	buf, err := d.Buffer(4)
	require.NoError(t, err)
	test.CloseOnCleanup(t, buf)
	require.ErrorIs(t, buf.Write([]byte{1, 2, 3, 4, 5}), ErrSize)
}

func TestCloseIdempotent(t *testing.T) {
	d, err := Open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	require.NoError(t, d.Close())
	require.NoError(t, d.Close())
	_, err = d.Buffer(16)
	require.ErrorIs(t, err, ErrClosed)
}
