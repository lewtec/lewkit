package wasm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
)

func TestCallAndReadCString(t *testing.T) {
	ctx := t.Context()
	runtime := wazero.NewRuntime(ctx)
	t.Cleanup(func() { require.NoError(t, runtime.Close(ctx)) })
	// (module (memory (export "m") 1) (func (export "seven") (result i32) i32.const 7))
	bin := []byte{
		0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00,
		0x01, 0x05, 0x01, 0x60, 0x00, 0x01, 0x7f,
		0x03, 0x02, 0x01, 0x00,
		0x05, 0x03, 0x01, 0x00, 0x01,
		0x07, 0x0d, 0x02,
		0x01, 0x6d, 0x02, 0x00,
		0x05, 0x73, 0x65, 0x76, 0x65, 0x6e, 0x00, 0x00,
		0x0a, 0x06, 0x01, 0x04, 0x00, 0x41, 0x07, 0x0b,
	}
	compiled, err := runtime.CompileModule(ctx, bin)
	require.NoError(t, err)
	mod, err := runtime.InstantiateModule(ctx, compiled, wazero.NewModuleConfig())
	require.NoError(t, err)

	got, err := Call(ctx, mod.ExportedFunction("seven"))
	require.NoError(t, err)
	assert.Equal(t, uint64(7), got)

	mem := mod.Memory()
	require.NotNil(t, mem)
	s, ok := ReadCString(mem, 0)
	require.True(t, ok)
	assert.Equal(t, "", s)
	require.True(t, mem.Write(16, []byte("hi\x00")))
	s, ok = ReadCString(mem, 16)
	require.True(t, ok)
	assert.Equal(t, "hi", s)
	s, ok = ReadCString(mem, mem.Size())
	assert.False(t, ok)
	assert.Equal(t, "", s)
}
