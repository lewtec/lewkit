package disasm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisassembleX86(t *testing.T) {
	engine, err := Open(t.Context(), ArchitectureX86, Mode64, WithSyntax(SyntaxIntel))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, engine.Close(t.Context())) })

	got, err := engine.Disassemble(t.Context(), []byte{0x90, 0xc3}, 0)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "nop", got[0].Mnemonic)
	assert.Equal(t, uint64(0), got[0].Address)
	assert.Equal(t, []byte{0x90}, got[0].Bytes)
	assert.Equal(t, "ret", got[1].Mnemonic)
	assert.Equal(t, uint64(1), got[1].Address)
}

func TestDisassembleATT(t *testing.T) {
	engine, err := Open(t.Context(), ArchitectureX86, Mode64, WithSyntax(SyntaxATT))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, engine.Close(t.Context())) })

	got, err := engine.Disassemble(t.Context(), []byte{0x41, 0x88, 0x70, 0x01}, 0)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "movb", got[0].Mnemonic)
	assert.Equal(t, "%sil, 1(%r8)", got[0].Operands)
}

func TestDisassembleAArch64(t *testing.T) {
	engine, err := Open(t.Context(), ArchitectureAArch64, ModeARM)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, engine.Close(t.Context())) })

	got, err := engine.Disassemble(t.Context(), []byte{0xff, 0x03, 0xff, 0xb8}, 0)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "ldaddal", got[0].Mnemonic)
	assert.Equal(t, "wzr, wzr, [sp]", got[0].Operands)
}

func TestDisassembleEmpty(t *testing.T) {
	engine, err := Open(t.Context(), ArchitectureX86, Mode64)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, engine.Close(t.Context())) })

	got, err := engine.Disassemble(t.Context(), nil, 0)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestOpenBadArchitecture(t *testing.T) {
	_, err := Open(t.Context(), 255, 0)
	require.Error(t, err)
}

func TestEngineClosed(t *testing.T) {
	engine, err := Open(t.Context(), ArchitectureX86, Mode64)
	require.NoError(t, err)
	require.NoError(t, engine.Close(t.Context()))
	_, err = engine.Disassemble(t.Context(), []byte{0x90}, 0)
	require.Error(t, err)
}
