// Package hex reads 0x addresses out of instruction text.
package hex

import (
	"strconv"
	"unicode"
)

// Token parses one 0x address at index.
// A letter, digit, or underscore on either side rejects the token.
func Token(text string, index int) (start, end int, address uint64, ok bool) {
	if index+2 > len(text) || text[index] != '0' || (text[index+1] != 'x' && text[index+1] != 'X') {
		return 0, 0, 0, false
	}
	if index > 0 {
		previous := rune(text[index-1])
		if unicode.IsLetter(previous) || unicode.IsDigit(previous) || previous == '_' {
			return 0, 0, 0, false
		}
	}
	end = index + 2
	for end < len(text) && Digit(text[end]) {
		end++
	}
	if end == index+2 {
		return 0, 0, 0, false
	}
	if end < len(text) {
		next := rune(text[end])
		if unicode.IsLetter(next) || unicode.IsDigit(next) || next == '_' {
			return 0, 0, 0, false
		}
	}
	address, err := strconv.ParseUint(text[index+2:end], 16, 64)
	if err != nil {
		return 0, 0, 0, false
	}
	return index, end, address, true
}

// Digit reports whether value is an ASCII hex digit.
func Digit(value byte) bool {
	return value >= '0' && value <= '9' || value >= 'a' && value <= 'f' || value >= 'A' && value <= 'F'
}
