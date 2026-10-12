package decomp

import (
	"strconv"
	"strings"
	"unicode"
)

type kind string

const (
	kindPlain  kind = "plain"
	kindBranch kind = "branch"
	kindJump   kind = "jump"
	kindCall   kind = "call"
	kindReturn kind = "return"
)

type control struct {
	kind   kind
	target uint64
	direct bool
}

func inspect(mnemonic, operands string) control {
	switch classify(mnemonic) {
	case kindReturn:
		return control{kind: kindReturn}
	case kindCall, kindJump, kindBranch:
		target, direct := directTarget(operands)
		return control{kind: classify(mnemonic), target: target, direct: direct}
	default:
		return control{kind: kindPlain}
	}
}

func classify(mnemonic string) kind {
	name := strings.ToLower(mnemonic)
	switch name {
	case "ret", "retn", "retf", "retfq":
		return kindReturn
	case "call", "callq", "bl", "blr", "blx", "jal", "jalr":
		return kindCall
	}
	if conditional(name) {
		return kindBranch
	}
	switch name {
	case "jmp", "jmpq", "b", "br", "bra", "j", "jr":
		return kindJump
	default:
		return kindPlain
	}
}

func conditional(name string) bool {
	switch name {
	case "cbz", "cbnz", "tbz", "tbnz":
		return true
	}
	if strings.HasPrefix(name, "b.") {
		return true
	}
	if !strings.HasPrefix(name, "j") {
		return false
	}
	switch name {
	case "jmp", "jmpq", "jal", "jalr", "jr", "j":
		return false
	default:
		return true
	}
}

func directTarget(operands string) (uint64, bool) {
	if strings.Contains(strings.ToLower(operands), "rip") {
		return 0, false
	}
	addresses := hexAddresses(operands)
	if len(addresses) == 0 {
		return 0, false
	}
	return addresses[0], true
}

func referencedAddresses(operands string, next uint64) []uint64 {
	var addresses []uint64
	if displacement, ok := ripDisplacement(operands); ok {
		addresses = append(addresses, uint64(int64(next)+displacement))
	}
	return append(addresses, hexAddresses(operands)...)
}

func ripDisplacement(operands string) (int64, bool) {
	index := strings.Index(strings.ToLower(operands), "rip")
	if index < 0 {
		return 0, false
	}
	rest := strings.TrimSpace(operands[index+3:])
	if rest == "" || rest[0] == ']' {
		return 0, true
	}
	sign := int64(1)
	switch rest[0] {
	case '+':
		rest = strings.TrimSpace(rest[1:])
	case '-':
		sign = -1
		rest = strings.TrimSpace(rest[1:])
	default:
		return 0, false
	}
	if len(rest) < 3 || rest[0] != '0' || (rest[1] != 'x' && rest[1] != 'X') {
		return 0, false
	}
	end := 2
	for end < len(rest) && isHex(rest[end]) {
		end++
	}
	if end == 2 {
		return 0, false
	}
	value, err := strconv.ParseUint(rest[2:end], 16, 64)
	if err != nil {
		return 0, false
	}
	return sign * int64(value), true
}

func hexAddresses(operands string) []uint64 {
	var addresses []uint64
	for index := 0; index < len(operands); {
		_, end, address, ok := hexToken(operands, index)
		if !ok {
			index++
			continue
		}
		addresses = append(addresses, address)
		index = end
	}
	return addresses
}

func hexToken(text string, index int) (start, end int, address uint64, ok bool) {
	if index+2 > len(text) || text[index] != '0' || (text[index+1] != 'x' && text[index+1] != 'X') {
		return 0, 0, 0, false
	}
	if index > 0 {
		previous := rune(text[index-1])
		if unicode.IsLetter(previous) || unicode.IsDigit(previous) || previous == '_' {
			return 0, 0, 0, false
		}
	}
	cursor := index + 2
	for cursor < len(text) && isHex(text[cursor]) {
		cursor++
	}
	if cursor == index+2 {
		return 0, 0, 0, false
	}
	if cursor < len(text) {
		next := rune(text[cursor])
		if unicode.IsLetter(next) || unicode.IsDigit(next) || next == '_' {
			return 0, 0, 0, false
		}
	}
	address, err := strconv.ParseUint(text[index+2:cursor], 16, 64)
	if err != nil {
		return 0, 0, 0, false
	}
	return index, cursor, address, true
}

func isHex(value byte) bool {
	return value >= '0' && value <= '9' || value >= 'a' && value <= 'f' || value >= 'A' && value <= 'F'
}
