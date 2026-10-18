package glsl

import (
	"fmt"
	"math"
	"strconv"
)

// NumberKind is the shape of a numeric literal.
type NumberKind int

const (
	NumberInt NumberKind = iota
	NumberUint
	NumberFloat
	NumberDouble
)

// Number is a numeric literal. It is not an AST node.
type Number struct {
	Kind  NumberKind
	Int   int64
	Uint  uint64
	Float float64
}

// ScanNumber reads one numeric literal starting at i.
// The returned index is the first byte after the literal.
func ScanNumber(src string, i, line int) (Number, int, error) {
	if i+1 < len(src) && src[i] == '0' && (src[i+1] == 'x' || src[i+1] == 'X') {
		return scanHex(src, i, line)
	}
	j := i
	for j < len(src) && isDigit(src[j]) {
		j++
	}
	if j < len(src) && (src[j] == '.' || src[j] == 'e' || src[j] == 'E') {
		return scanFloat(src, i, line)
	}
	raw := src[i:j]
	base := 10
	if len(raw) > 1 && raw[0] == '0' && octalDigits(raw) {
		base = 8
	}
	value, err := strconv.ParseUint(raw, base, 64)
	if err != nil {
		return Number{}, i, fmt.Errorf("%w: line %d: %s", ErrSyntax, line, raw)
	}
	kind := NumberInt
	if j < len(src) && (src[j] == 'u' || src[j] == 'U') {
		kind = NumberUint
		j++
	}
	if kind == NumberInt && value > math.MaxInt32 {
		if value > math.MaxUint32 {
			return Number{}, i, fmt.Errorf("%w: line %d: %s", ErrSyntax, line, raw)
		}
		kind = NumberUint
	}
	if kind == NumberUint && value > math.MaxUint32 {
		return Number{}, i, fmt.Errorf("%w: line %d: %s", ErrSyntax, line, raw)
	}
	if kind == NumberInt {
		return Number{Kind: NumberInt, Int: int64(value)}, j, nil
	}
	return Number{Kind: NumberUint, Uint: value}, j, nil
}

func octalDigits(raw string) bool {
	for _, r := range raw {
		if r < '0' || r > '7' {
			return false
		}
	}
	return true
}

func scanHex(src string, i, line int) (Number, int, error) {
	j := i + 2
	digits := 0
	for j < len(src) && isHex(src[j]) {
		j++
		digits++
	}
	hexFloat := false
	if j < len(src) && src[j] == '.' {
		hexFloat = true
		j++
		for j < len(src) && isHex(src[j]) {
			j++
			digits++
		}
	}
	if j < len(src) && (src[j] == 'p' || src[j] == 'P') {
		hexFloat = true
		j++
		if j < len(src) && (src[j] == '+' || src[j] == '-') {
			j++
		}
		if j >= len(src) || !isDigit(src[j]) {
			return Number{}, i, fmt.Errorf("%w: line %d: hex float", ErrSyntax, line)
		}
		for j < len(src) && isDigit(src[j]) {
			j++
		}
	}
	if digits == 0 {
		return Number{}, i, fmt.Errorf("%w: line %d: hex", ErrSyntax, line)
	}
	if hexFloat {
		value, err := strconv.ParseFloat(src[i:j], 64)
		if err != nil {
			return Number{}, i, fmt.Errorf("%w: line %d: hex float", ErrSyntax, line)
		}
		return floatSuffix(src, j, value)
	}
	value, err := strconv.ParseUint(src[i:j], 0, 64)
	if err != nil {
		return Number{}, i, fmt.Errorf("%w: line %d: hex", ErrSyntax, line)
	}
	kind := NumberInt
	if j < len(src) && (src[j] == 'u' || src[j] == 'U') {
		kind = NumberUint
		j++
	}
	if kind == NumberInt && value > math.MaxInt32 {
		if value > math.MaxUint32 {
			return Number{}, i, fmt.Errorf("%w: line %d: hex", ErrSyntax, line)
		}
		kind = NumberUint
	}
	if kind == NumberInt {
		return Number{Kind: NumberInt, Int: int64(value)}, j, nil
	}
	return Number{Kind: NumberUint, Uint: value}, j, nil
}

func scanFloat(src string, i, line int) (Number, int, error) {
	j := i
	if src[j] == '.' {
		j++
		if j >= len(src) || !isDigit(src[j]) {
			return Number{}, i, fmt.Errorf("%w: line %d: float", ErrSyntax, line)
		}
		for j < len(src) && isDigit(src[j]) {
			j++
		}
	} else {
		for j < len(src) && isDigit(src[j]) {
			j++
		}
		if j < len(src) && src[j] == '.' {
			j++
			for j < len(src) && isDigit(src[j]) {
				j++
			}
		}
	}
	if j < len(src) && (src[j] == 'e' || src[j] == 'E') {
		j++
		if j < len(src) && (src[j] == '+' || src[j] == '-') {
			j++
		}
		if j >= len(src) || !isDigit(src[j]) {
			return Number{}, i, fmt.Errorf("%w: line %d: float", ErrSyntax, line)
		}
		for j < len(src) && isDigit(src[j]) {
			j++
		}
	}
	value, err := strconv.ParseFloat(src[i:j], 64)
	if err != nil {
		return Number{}, i, fmt.Errorf("%w: line %d: float", ErrSyntax, line)
	}
	return floatSuffix(src, j, value)
}

func floatSuffix(src string, j int, value float64) (Number, int, error) {
	if j+1 < len(src) && (src[j] == 'l' || src[j] == 'L') && (src[j+1] == 'f' || src[j+1] == 'F') {
		return Number{Kind: NumberDouble, Float: value}, j + 2, nil
	}
	if j < len(src) && (src[j] == 'f' || src[j] == 'F') {
		return Number{Kind: NumberFloat, Float: value}, j + 1, nil
	}
	return Number{Kind: NumberFloat, Float: value}, j, nil
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

func isHex(b byte) bool {
	return isDigit(b) || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}
