package ninja

import (
	"fmt"
	"strings"
)

func expand(s string, resolve func(string) (string, bool, bool), keepSpace bool) (string, error) {
	return expandDepth(s, resolve, map[string]bool{}, keepSpace)
}

func expandDepth(s string, resolve func(string) (string, bool, bool), stack map[string]bool, keepSpace bool) (string, error) {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '$' {
			b.WriteByte(s[i])
			continue
		}
		if i+1 >= len(s) {
			return "", fmt.Errorf("%w: trailing $", ErrSyntax)
		}
		i++
		switch s[i] {
		case '$':
			b.WriteByte('$')
		case ' ':
			if keepSpace {
				b.WriteByte(0)
			} else {
				b.WriteByte(' ')
			}
		case ':':
			if keepSpace {
				b.WriteByte(1)
			} else {
				b.WriteByte(':')
			}
		case '|':
			if keepSpace {
				b.WriteByte(2)
			} else {
				b.WriteByte('|')
			}
		case '{':
			end := strings.IndexByte(s[i:], '}')
			if end < 0 {
				return "", fmt.Errorf("%w: unclosed ${", ErrSyntax)
			}
			name := s[i+1 : i+end]
			i += end
			part, err := expandName(name, resolve, stack, keepSpace)
			if err != nil {
				return "", err
			}
			b.WriteString(part)
		default:
			j := i
			for j < len(s) && nameChar(s[j]) {
				j++
			}
			if j == i {
				return "", fmt.Errorf("%w: bad $", ErrSyntax)
			}
			name := s[i:j]
			i = j - 1
			part, err := expandName(name, resolve, stack, keepSpace)
			if err != nil {
				return "", err
			}
			b.WriteString(part)
		}
	}
	return b.String(), nil
}

func expandName(name string, resolve func(string) (string, bool, bool), stack map[string]bool, keepSpace bool) (string, error) {
	if stack[name] {
		return "", fmt.Errorf("%w: %s", ErrCycle, name)
	}
	val, literal, ok := resolve(name)
	if !ok {
		return "", nil
	}
	if literal {
		return val, nil
	}
	stack[name] = true
	out, err := expandDepth(val, resolve, stack, keepSpace)
	delete(stack, name)
	return out, err
}

func nameChar(c byte) bool {
	return c == '_' || c == '-' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}
