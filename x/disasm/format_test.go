package disasm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatInstructionLabel(t *testing.T) {
	instruction := Instruction{
		Address:  0x1000,
		Bytes:    []byte{0xe8, 0x00, 0x00, 0x00, 0x00},
		Mnemonic: "call",
		Operands: "0x2000",
	}
	names := map[uint64]string{0x1000: "start", 0x2000: "puts"}
	got := FormatInstruction(instruction, names)
	assert.Contains(t, got, "<start>:")
	assert.Contains(t, got, "call 0x2000 <puts>")
}

func TestFormatInstructionNoSymbol(t *testing.T) {
	instruction := Instruction{
		Address:  0,
		Bytes:    []byte{0x90},
		Mnemonic: "nop",
	}
	got := FormatInstruction(instruction, nil)
	assert.Equal(t, "0x00000000  90               nop\n", got)
}

func TestSymbolsLookupPrefersFirst(t *testing.T) {
	symbols := Symbols{
		{Name: "fn", Address: 0x10},
		{Name: "alias", Address: 0x10},
	}
	assert.Equal(t, "fn", symbols.Lookup()[0x10])
}
