package proto

import (
	"strconv"
	"unicode"
	"unicode/utf8"
)

func identifierStart(r rune) bool {
	return r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z'
}

func identifierContinue(r rune) bool {
	return identifierStart(r) || r >= '0' && r <= '9'
}

func matchSpace(src []byte, at int) int {
	i := at
	for i < len(src) {
		switch src[i] {
		case ' ', '\t', '\r', '\v', '\f', '\n':
			i++
		default:
			return i - at
		}
	}
	return i - at
}

func matchComment(src []byte, at int) int {
	if at+1 >= len(src) || src[at] != '/' {
		return 0
	}
	switch src[at+1] {
	case '/':
		i := at + 2
		for i < len(src) && src[i] != '\n' {
			i++
		}
		return i - at
	case '*':
		i := at + 2
		for i+1 < len(src) {
			if src[i] == '*' && src[i+1] == '/' {
				return i + 2 - at
			}
			i++
		}
		return 0
	default:
		return 0
	}
}

// classifyNumber splits integers from floats so the two tokens never cover
// the same span. Equal length and equal lexical precedence would keep only
// the lower symbol id, and the parser would never see the other one.
func classifyNumber(src []byte, at int) (int, bool) {
	if at >= len(src) {
		return 0, false
	}
	if src[at] == '.' {
		n := scanFloat(src, at)
		if n <= 0 {
			return 0, false
		}
		return n, true
	}
	if !isDigit(src[at]) {
		return 0, false
	}
	if src[at] == '0' && at+1 < len(src) && (src[at+1] == 'x' || src[at+1] == 'X') {
		i := at + 2
		if i >= len(src) || !isHex(src[i]) {
			return 0, false
		}
		i++
		for i < len(src) && isHex(src[i]) {
			i++
		}
		return i - at, false
	}
	i := at + 1
	if src[at] == '0' {
		sawDigit := false
		octal := true
		for i < len(src) && isDigit(src[i]) {
			sawDigit = true
			if src[i] > '7' {
				octal = false
			}
			i++
		}
		if i < len(src) && isFloatTail(src[i]) {
			n := scanFloat(src, at)
			if n <= 0 {
				return 0, false
			}
			return n, true
		}
		if sawDigit && !octal {
			return 0, false
		}
		return i - at, false
	}
	for i < len(src) && isDigit(src[i]) {
		i++
	}
	if i < len(src) && isFloatTail(src[i]) {
		n := scanFloat(src, at)
		if n <= 0 {
			return 0, false
		}
		return n, true
	}
	return i - at, false
}

func matchInteger(src []byte, at int) int {
	n, float := classifyNumber(src, at)
	if n <= 0 || float {
		return 0
	}
	return n
}

func matchFloat(src []byte, at int) int {
	n, float := classifyNumber(src, at)
	if n <= 0 || !float {
		return 0
	}
	return n
}

func isFloatTail(c byte) bool {
	return c == '.' || c == 'e' || c == 'E'
}

func scanFloat(src []byte, start int) int {
	i := start
	if i < len(src) && src[i] == '.' {
		i++
		if i >= len(src) || !isDigit(src[i]) {
			return 0
		}
		for i < len(src) && isDigit(src[i]) {
			i++
		}
	} else {
		if i >= len(src) || !isDigit(src[i]) {
			return 0
		}
		for i < len(src) && isDigit(src[i]) {
			i++
		}
		if i < len(src) && src[i] == '.' {
			i++
			for i < len(src) && isDigit(src[i]) {
				i++
			}
		}
	}
	if i < len(src) && (src[i] == 'e' || src[i] == 'E') {
		j := i + 1
		if j < len(src) && (src[j] == '+' || src[j] == '-') {
			j++
		}
		if j >= len(src) || !isDigit(src[j]) {
			return 0
		}
		j++
		for j < len(src) && isDigit(src[j]) {
			j++
		}
		i = j
	}
	if i <= start {
		return 0
	}
	return i - start
}

func parseIntToken(text string) (int64, error) {
	switch {
	case len(text) >= 2 && text[0] == '0' && (text[1] == 'x' || text[1] == 'X'):
		return strconv.ParseInt(text[2:], 16, 64)
	case len(text) > 1 && text[0] == '0':
		return strconv.ParseInt(text, 8, 64)
	default:
		return strconv.ParseInt(text, 10, 64)
	}
}

func matchString(src []byte, at int) int {
	n := stringLen(src, at)
	if n <= 0 {
		return 0
	}
	return n
}

func stringLen(src []byte, at int) int {
	if at >= len(src) || (src[at] != '"' && src[at] != '\'') {
		return 0
	}
	quote := src[at]
	i := at + 1
	for i < len(src) {
		char := src[i]
		if char == '\n' || char == '\r' {
			return 0
		}
		if char == quote {
			return i + 1 - at
		}
		if char != '\\' {
			i++
			continue
		}
		width := escapeWidth(src, i)
		if width == 0 {
			return 0
		}
		i += width
	}
	return 0
}

func escapeWidth(src []byte, at int) int {
	if at >= len(src) || src[at] != '\\' || at+1 >= len(src) {
		return 0
	}
	switch src[at+1] {
	case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', '\'', '"':
		return 2
	case 'x', 'X':
		n := hexRun(src, at+2, 2)
		if n < 1 {
			return 0
		}
		return 2 + n
	case 'u':
		if hexRun(src, at+2, 4) != 4 {
			return 0
		}
		return 6
	case 'U':
		if hexRun(src, at+2, 8) != 8 {
			return 0
		}
		return 10
	case '0', '1', '2', '3', '4', '5', '6', '7':
		i := at + 2
		digits := 1
		for digits < 3 && i < len(src) && src[i] >= '0' && src[i] <= '7' {
			i++
			digits++
		}
		return i - at
	default:
		return 0
	}
}

func hexRun(src []byte, at, maxDigits int) int {
	n := 0
	for n < maxDigits && at+n < len(src) && isHex(src[at+n]) {
		n++
	}
	return n
}

func decodeString(raw string) (string, error) {
	src := []byte(raw)
	if len(src) < 2 || stringLen(src, 0) != len(src) {
		return "", ErrSyntax
	}
	var text []byte
	for i := 1; i < len(src)-1; {
		char := src[i]
		if char != '\\' {
			text = append(text, char)
			i++
			continue
		}
		escaped := src[i+1]
		switch escaped {
		case 'a':
			text = append(text, '\a')
			i += 2
		case 'b':
			text = append(text, '\b')
			i += 2
		case 'f':
			text = append(text, '\f')
			i += 2
		case 'n':
			text = append(text, '\n')
			i += 2
		case 'r':
			text = append(text, '\r')
			i += 2
		case 't':
			text = append(text, '\t')
			i += 2
		case 'v':
			text = append(text, '\v')
			i += 2
		case '\\', '\'', '"':
			text = append(text, escaped)
			i += 2
		case 'x', 'X':
			value, n := takeHex(src, i+2, 1, 2)
			text = append(text, byte(value))
			i = n
		case 'u':
			value, n := takeHex(src, i+2, 4, 4)
			text = appendRune(text, rune(value))
			i = n
		case 'U':
			value, n := takeHex(src, i+2, 8, 8)
			if rune(value) > unicode.MaxRune {
				return "", ErrSyntax
			}
			text = appendRune(text, rune(value))
			i = n
		case '0', '1', '2', '3', '4', '5', '6', '7':
			octal := int(escaped - '0')
			j := i + 2
			digits := 1
			for digits < 3 && j < len(src) && src[j] >= '0' && src[j] <= '7' {
				octal = octal*8 + int(src[j]-'0')
				j++
				digits++
			}
			text = append(text, byte(octal))
			i = j
		default:
			return "", ErrSyntax
		}
	}
	return string(text), nil
}

func takeHex(src []byte, at, minDigits, maxDigits int) (int, int) {
	value := 0
	digits := 0
	for digits < maxDigits && at+digits < len(src) && isHex(src[at+digits]) {
		value = value<<4 | hexValue(src[at+digits])
		digits++
	}
	if digits < minDigits {
		return 0, at
	}
	return value, at + digits
}

func appendRune(text []byte, char rune) []byte {
	encoded := make([]byte, utf8.UTFMax)
	size := utf8.EncodeRune(encoded, char)
	return append(text, encoded[:size]...)
}

func isDigit(char byte) bool {
	return char >= '0' && char <= '9'
}

func isHex(char byte) bool {
	return isDigit(char) || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')
}

func hexValue(char byte) int {
	switch {
	case char >= '0' && char <= '9':
		return int(char - '0')
	case char >= 'a' && char <= 'f':
		return int(char-'a') + 10
	default:
		return int(char-'A') + 10
	}
}
