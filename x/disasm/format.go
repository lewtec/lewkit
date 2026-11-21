package disasm

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"unicode"
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
		if start, end, address, ok := hexToken(operands, i); ok {
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

func hexToken(s string, i int) (start, end int, address uint64, ok bool) {
	if i+2 > len(s) || s[i] != '0' || (s[i+1] != 'x' && s[i+1] != 'X') {
		return 0, 0, 0, false
	}
	if i > 0 {
		prev := rune(s[i-1])
		if unicode.IsLetter(prev) || unicode.IsDigit(prev) || prev == '_' {
			return 0, 0, 0, false
		}
	}
	j := i + 2
	for j < len(s) {
		c := s[j]
		if c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F' {
			j++
			continue
		}
		break
	}
	if j == i+2 {
		return 0, 0, 0, false
	}
	if j < len(s) {
		next := rune(s[j])
		if unicode.IsLetter(next) || unicode.IsDigit(next) || next == '_' {
			return 0, 0, 0, false
		}
	}
	address, err := strconv.ParseUint(s[i+2:j], 16, 64)
	if err != nil {
		return 0, 0, 0, false
	}
	return i, j, address, true
}
