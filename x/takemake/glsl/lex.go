package glsl

import (
	"fmt"
	"strings"

	syntax "github.com/lewtec/lewkit/x/parse/glsl"
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
			lit, next, err := scanNumber(src, i, line)
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
	num, end, err := syntax.ScanNumber(src, i, line)
	if err != nil {
		return Literal{}, i, err
	}
	switch num.Kind {
	case syntax.NumberInt:
		return Literal{Kind: KindInt, Int: num.Int}, end, nil
	case syntax.NumberUint:
		return Literal{Kind: KindUint, Uint: num.Uint}, end, nil
	case syntax.NumberFloat:
		return Literal{Kind: KindFloat, Float: num.Float}, end, nil
	case syntax.NumberDouble:
		return Literal{Kind: KindDouble, Float: num.Float}, end, nil
	default:
		return Literal{}, i, fmt.Errorf("%w: line %d: number", ErrSyntax, line)
	}
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }
