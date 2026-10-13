package disasm

import (
	"encoding/hex"
	"fmt"
	"strings"

	hexscan "github.com/lewtec/lewkit/x/disasm/internal/hex"
)

// FormatInstruction prints one instruction. A symbol at the instruction
// address is emitted as a label. Operand hex addresses that match a
// symbol are annotated as 0xADDR <name>.
func FormatInstruction(instruction Instruction, names map[uint64]string) string {
	var b strings.Builder
	if name, ok := names[instruction.Address]; ok {
		fmt.Fprintf(&b, "<%s>:\n", name)
	}
	operands := annotateOperands(instruction.Operands, names)
	fmt.Fprintf(&b, "0x%08x  %-16s %s", instruction.Address, hex.EncodeToString(instruction.Bytes), instruction.Mnemonic)
	if operands != "" {
		b.WriteByte(' ')
		b.WriteString(operands)
	}
	b.WriteByte('\n')
	return b.String()
}

func annotateOperands(operands string, names map[uint64]string) string {
	if operands == "" || len(names) == 0 {
		return operands
	}
	var b strings.Builder
	i := 0
	for i < len(operands) {
		if start, end, address, ok := hexscan.Token(operands, i); ok {
			b.WriteString(operands[i:start])
			token := operands[start:end]
			if name, found := names[address]; found {
				fmt.Fprintf(&b, "%s <%s>", token, name)
			} else {
				b.WriteString(token)
			}
			i = end
			continue
		}
		b.WriteByte(operands[i])
		i++
	}
	return b.String()
}
