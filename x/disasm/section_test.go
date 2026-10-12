package disasm

import (
	"bytes"
	"debug/elf"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestELFSectionsAndEntry(t *testing.T) {
	raw := ELF64(elf.EM_X86_64, []byte{0x90, 0xc3})
	file, format, err := OpenObject(bytes.NewReader(raw), int64(len(raw)))
	require.NoError(t, err)
	defer file.Close()
	assert.Equal(t, "elf", format)
	assert.Equal(t, uint64(0), file.Entry())
	sections, err := file.Sections()
	require.NoError(t, err)
	require.Len(t, sections, 1)
	assert.Equal(t, ".text", sections[0].Name)
	assert.Equal(t, uint64(0x1000), sections[0].Address)
	assert.Equal(t, []byte{0x90, 0xc3}, sections[0].Bytes)
	assert.True(t, sections[0].Exec)
	assert.False(t, sections[0].Write)
}
