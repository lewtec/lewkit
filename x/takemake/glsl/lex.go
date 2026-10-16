package glsl

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	tokEOF = iota
	tokIdent
	tokNumber
	tokOp
)

type token struct {
	kind tokenKind
	text string
	lit  *Literal
	line int
}

type tokenKind int

func lex(src string) ([]token, error) {
	var toks []token
	line := 1
	for i := 0; i < len(src); {
		switch src[i] {
		case '\n':
			line++
			i++
			continue
		case ' ', '\t', '\v', '\f':
			i++
			continue
		}
		if src[i] == '.' && i+1 < len(src) && isDigit(src[i+1]) {
			lit, next, err := scanFloat(src, i, line)
			if err != nil {
				return nil, err
			}
			toks = append(toks, token{kind: tokNumber, text: src[i:next], lit: &lit, line: line})
			i = next
			continue
		}
		if isDigit(src[i]) {
			lit, next, err := scanNumber(src, i, line)
			if err != nil {
				return nil, err
			}
			toks = append(toks, token{kind: tokNumber, text: src[i:next], lit: &lit, line: line})
			i = next
			continue
		}
		if isIdentStart(src[i]) {
			name, _ := takeIdent(src[i:])
			toks = append(toks, token{kind: tokIdent, text: name, line: line})
			i += len(name)
			continue
		}
		if op, size := matchOp(src[i:]); size > 0 {
			toks = append(toks, token{kind: tokOp, text: op, line: line})
			i += size
			continue
		}
		return nil, fmt.Errorf("%w: line %d: %q", ErrSyntax, line, src[i:i+1])
	}
	return toks, nil
}

func matchOp(src string) (string, int) {
	for _, op := range []string{"<<=", ">>="} {
		if strings.HasPrefix(src, op) {
			return op, len(op)
		}
	}
	for _, op := range []string{
		"<<", ">>", "<=", ">=", "==", "!=", "&&", "||", "^^",
		"++", "--", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=",
	} {
		if strings.HasPrefix(src, op) {
			return op, len(op)
		}
	}
	if src != "" && strings.ContainsRune("()[]{}.,;:?+-*/%&|^!~=<>", rune(src[0])) {
		return src[:1], 1
	}
	return "", 0
}

func scanNumber(src string, i, line int) (Literal, int, error) {
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
		return Literal{}, i, fmt.Errorf("%w: line %d: %s", ErrSyntax, line, raw)
	}
	kind := KindInt
	if j < len(src) && (src[j] == 'u' || src[j] == 'U') {
		kind = KindUint
		j++
	}
	if kind == KindInt && value > math.MaxInt32 {
		if value > math.MaxUint32 {
			return Literal{}, i, fmt.Errorf("%w: line %d: %s", ErrSyntax, line, raw)
		}
		kind = KindUint
	}
	if kind == KindUint && value > math.MaxUint32 {
		return Literal{}, i, fmt.Errorf("%w: line %d: %s", ErrSyntax, line, raw)
	}
	if kind == KindInt {
		return Literal{Kind: KindInt, Int: int64(value)}, j, nil
	}
	return Literal{Kind: KindUint, Uint: value}, j, nil
}

func octalDigits(raw string) bool {
	for _, r := range raw {
		if r < '0' || r > '7' {
			return false
		}
	}
	return true
}

func scanHex(src string, i, line int) (Literal, int, error) {
	j := i + 2
	digits := 0
	for j < len(src) && isHex(src[j]) {
		j++
		digits++
	}
	float := false
	if j < len(src) && src[j] == '.' {
		float = true
		j++
		for j < len(src) && isHex(src[j]) {
			j++
			digits++
		}
	}
	if j < len(src) && (src[j] == 'p' || src[j] == 'P') {
		float = true
		j++
		if j < len(src) && (src[j] == '+' || src[j] == '-') {
			j++
		}
		if j >= len(src) || !isDigit(src[j]) {
			return Literal{}, i, fmt.Errorf("%w: line %d: hex float", ErrSyntax, line)
		}
		for j < len(src) && isDigit(src[j]) {
			j++
		}
	}
	if digits == 0 {
		return Literal{}, i, fmt.Errorf("%w: line %d: hex", ErrSyntax, line)
	}
	if float {
		value, err := strconv.ParseFloat(src[i:j], 64)
		if err != nil {
			return Literal{}, i, fmt.Errorf("%w: line %d: hex float", ErrSyntax, line)
		}
		return floatSuffix(src, j, line, value)
	}
	value, err := strconv.ParseUint(src[i:j], 0, 64)
	if err != nil {
		return Literal{}, i, fmt.Errorf("%w: line %d: hex", ErrSyntax, line)
	}
	kind := KindInt
	if j < len(src) && (src[j] == 'u' || src[j] == 'U') {
		kind = KindUint
		j++
	}
	if kind == KindInt && value > math.MaxInt32 {
		if value > math.MaxUint32 {
			return Literal{}, i, fmt.Errorf("%w: line %d: hex", ErrSyntax, line)
		}
		kind = KindUint
	}
	if kind == KindInt {
		return Literal{Kind: KindInt, Int: int64(value)}, j, nil
	}
	return Literal{Kind: KindUint, Uint: value}, j, nil
}

func scanFloat(src string, i, line int) (Literal, int, error) {
	j := i
	if src[j] == '.' {
		j++
		if j >= len(src) || !isDigit(src[j]) {
			return Literal{}, i, fmt.Errorf("%w: line %d: float", ErrSyntax, line)
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
			return Literal{}, i, fmt.Errorf("%w: line %d: float", ErrSyntax, line)
		}
		for j < len(src) && isDigit(src[j]) {
			j++
		}
	}
	value, err := strconv.ParseFloat(src[i:j], 64)
	if err != nil {
		return Literal{}, i, fmt.Errorf("%w: line %d: float", ErrSyntax, line)
	}
	return floatSuffix(src, j, line, value)
}

func floatSuffix(src string, j, line int, value float64) (Literal, int, error) {
	if j+1 < len(src) && (src[j] == 'l' || src[j] == 'L') && (src[j+1] == 'f' || src[j+1] == 'F') {
		return Literal{Kind: KindDouble, Float: value}, j + 2, nil
	}
	if j < len(src) && (src[j] == 'f' || src[j] == 'F') {
		return Literal{Kind: KindFloat, Float: value}, j + 1, nil
	}
	return Literal{Kind: KindFloat, Float: value}, j, nil
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

func isHex(b byte) bool {
	return isDigit(b) || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}
