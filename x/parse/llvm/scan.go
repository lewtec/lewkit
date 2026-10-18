package llvm

import xparse "github.com/lewtec/lewkit/x/parse"

func scanLLVM(src []byte, at int, _ func(xparse.SymbolID) bool, st *xparse.LexState) (xparse.SymbolID, int, int, bool) {
	if at < 0 {
		at = 0
	}
	i := skip(src, at)
	if i >= len(src) {
		return xparse.SymEOF, i, i, true
	}
	start := i
	c := src[i]
	switch c {
	case '"':
		end := scanString(src, i)
		if end < 0 {
			return 0, start, start, false
		}
		return sym(st, TermString.String(), start, end)
	case '%', '@':
		return scanSigil(src, i, c, st)
	case '!':
		return scanBang(src, i, st)
	case '{':
		if opaqueBrace(src, i) {
			end := balance(src, i, '{', '}')
			if end < 0 {
				return 0, start, start, false
			}
			return sym(st, TermBraced.String(), start, end)
		}
		return sym(st, "{", start, i+1)
	case '.':
		if i+2 < len(src) && src[i+1] == '.' && src[i+2] == '.' {
			return sym(st, TermEllipsis.String(), start, i+3)
		}
		if i+1 < len(src) && isDigit(src[i+1]) {
			return scanNumber(src, i, st)
		}
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		if c == '-' && !numberAfterDash(src, i) {
			return 0, start, start, false
		}
		if c != '-' || numberAfterDash(src, i) {
			if c == '-' || isDigit(c) {
				return scanNumber(src, i, st)
			}
		}
	}
	if c == 'c' && i+1 < len(src) && src[i+1] == '"' {
		end := scanString(src, i+1)
		if end < 0 {
			return 0, start, start, false
		}
		return sym(st, TermCString.String(), start, end)
	}
	switch c {
	case ',', '=', '(', ')', '}', '[', ']', '<', '>', '*', '#':
		return sym(st, string(c), start, i+1)
	}
	if isIdentStart(c) {
		return scanIdent(src, i, st)
	}
	return 0, start, start, false
}

func sym(st *xparse.LexState, name string, start, end int) (xparse.SymbolID, int, int, bool) {
	id, ok := st.Symbol(name)
	if !ok {
		return 0, start, end, false
	}
	return id, start, end, true
}

func skip(src []byte, i int) int {
	for i < len(src) {
		if src[i] == ';' {
			for i < len(src) && src[i] != '\n' {
				i++
			}
			continue
		}
		if isSpace(src[i]) {
			i++
			continue
		}
		return i
	}
	return i
}

func prevNonspace(src []byte, at int) byte {
	i := at
	for i > 0 {
		i--
		if isSpace(src[i]) {
			continue
		}
		return src[i]
	}
	return 0
}

func opaqueBrace(src []byte, at int) bool {
	i := at
	for i > 0 {
		i--
		if isSpace(src[i]) {
			continue
		}
		return src[i] == '!' || src[i] == '='
	}
	return false
}

func scanBang(src []byte, i int, st *xparse.LexState) (xparse.SymbolID, int, int, bool) {
	j := skip(src, i+1)
	if j < len(src) && (isIdentStart(src[j]) || isDigit(src[j])) {
		k := j
		if isDigit(src[j]) && !isIdentStart(src[j]) {
			for k < len(src) && isDigit(src[k]) {
				k++
			}
		} else {
			k++
			for k < len(src) && isIdentCont(src[k]) {
				k++
			}
		}
		if skip(src, k) < len(src) && src[skip(src, k)] == '=' {
			return sym(st, TermMetadataExclamation.String(), i, i+1)
		}
	}
	return sym(st, TermExclamation.String(), i, i+1)
}

func scanSigil(src []byte, i int, sig byte, st *xparse.LexState) (xparse.SymbolID, int, int, bool) {
	end, ok := sigilEnd(src, i)
	if !ok {
		return 0, i, i, false
	}
	definition := false
	if j := skip(src, end); j < len(src) && src[j] == '=' {
		definition = true
	}
	name := TermLocalReference.String()
	if sig == '@' {
		name = TermGlobalReference.String()
	}
	if definition {
		name = TermLocalDefinition.String()
		if sig == '@' {
			name = TermGlobalDefinition.String()
		}
	}
	return sym(st, name, i, end)
}

func sigilEnd(src []byte, i int) (int, bool) {
	if i >= len(src) {
		return 0, false
	}
	j := i + 1
	if j >= len(src) {
		return 0, false
	}
	if src[j] == '"' {
		end := scanString(src, j)
		if end < 0 {
			return 0, false
		}
		return end, true
	}
	if isDigit(src[j]) {
		j++
		for j < len(src) && isDigit(src[j]) {
			j++
		}
		return j, true
	}
	if !isSigilStart(src[j]) {
		return 0, false
	}
	j++
	for j < len(src) && isSigilCont(src[j]) {
		j++
	}
	return j, true
}

func scanIdent(src []byte, i int, st *xparse.LexState) (xparse.SymbolID, int, int, bool) {
	j := i + 1
	for j < len(src) && isIdentCont(src[j]) {
		j++
	}
	word := string(src[i:j])
	if k := skip(src, j); k < len(src) && src[k] == ':' {
		st.At = k + 1
		return sym(st, TermLabelName.String(), i, j)
	}
	// A word after '!' is a metadata name, even when it spells a keyword.
	if prevNonspace(src, i) == '!' {
		if k := skip(src, j); k < len(src) && src[k] == '(' {
			end := balance(src, k, '(', ')')
			if end < 0 {
				return 0, i, i, false
			}
			return sym(st, TermAttributeGroupReference.String(), i, end)
		}
		return sym(st, TermIdentifier.String(), i, j)
	}
	if word[0] == '$' {
		end := j
		if word == "$" {
			k := skip(src, j)
			if k < len(src) && src[k] == '"' {
				if e := scanString(src, k); e > 0 {
					end = e
				}
			} else if k < len(src) && isIdentStart(src[k]) {
				e := k + 1
				for e < len(src) && isIdentCont(src[e]) {
					e++
				}
				end = e
			}
		}
		return sym(st, TermComdatName.String(), i, end)
	}
	if _, ok := intBits(word); ok {
		return sym(st, TermIntegerType.String(), i, j)
	}
	if _, ok := keywords[word]; ok {
		end := j
		if word == "inrange" || word == "comdat" {
			if k := skip(src, j); k < len(src) && src[k] == '(' {
				if e := balance(src, k, '(', ')'); e > 0 {
					end = e
				}
			}
		}
		return sym(st, word, i, end)
	}
	if k := skip(src, j); k < len(src) && src[k] == '(' {
		end := balance(src, k, '(', ')')
		if end < 0 {
			return 0, i, i, false
		}
		return sym(st, TermAttributeGroupReference.String(), i, end)
	}
	return sym(st, TermIdentifier.String(), i, j)
}

func scanNumber(src []byte, i int, st *xparse.LexState) (xparse.SymbolID, int, int, bool) {
	end, ok := numberEnd(src, i)
	if !ok {
		return 0, i, i, false
	}
	if k := skip(src, end); k < len(src) && src[k] == ':' {
		st.At = k + 1
		return sym(st, TermLabelName.String(), i, end)
	}
	return sym(st, TermNumber.String(), i, end)
}

func numberAfterDash(src []byte, i int) bool {
	if i+1 >= len(src) {
		return false
	}
	if isDigit(src[i+1]) {
		return true
	}
	return src[i+1] == '.' && i+2 < len(src) && isDigit(src[i+2])
}

func numberEnd(src []byte, i int) (int, bool) {
	if i < len(src) && src[i] == '-' {
		i++
	}
	if i+1 < len(src) && src[i] == '0' && (src[i+1] == 'x' || src[i+1] == 'X') {
		i += 2
		if i < len(src) {
			switch src[i] {
			case 'K', 'L', 'M', 'H', 'R':
				i++
			}
		}
		saw := false
		for i < len(src) && isHex(src[i]) {
			saw = true
			i++
		}
		if i+1 < len(src) && src[i] == '.' && isHex(src[i+1]) {
			i++
			for i < len(src) && isHex(src[i]) {
				saw = true
				i++
			}
		}
		if i < len(src) && (src[i] == 'p' || src[i] == 'P') {
			i++
			if i < len(src) && (src[i] == '+' || src[i] == '-') {
				i++
			}
			if i >= len(src) || !isDigit(src[i]) {
				return 0, false
			}
			for i < len(src) && isDigit(src[i]) {
				i++
			}
			saw = true
		}
		if !saw {
			return 0, false
		}
		return i, true
	}
	saw := false
	for i < len(src) && isDigit(src[i]) {
		saw = true
		i++
	}
	if i+1 < len(src) && src[i] == '.' && isDigit(src[i+1]) {
		i++
		for i < len(src) && isDigit(src[i]) {
			saw = true
			i++
		}
	}
	if i < len(src) && (src[i] == 'e' || src[i] == 'E') {
		i++
		if i < len(src) && (src[i] == '+' || src[i] == '-') {
			i++
		}
		if i >= len(src) || !isDigit(src[i]) {
			return 0, false
		}
		for i < len(src) && isDigit(src[i]) {
			i++
		}
		saw = true
	}
	if !saw {
		return 0, false
	}
	return i, true
}

func scanString(src []byte, i int) int {
	if i >= len(src) || src[i] != '"' {
		return -1
	}
	i++
	for i < len(src) {
		c := src[i]
		if c == '"' {
			return i + 1
		}
		if c == '\n' {
			return -1
		}
		if c == '\\' {
			i++
			if i >= len(src) {
				return -1
			}
			n := src[i]
			if n == '\\' || n == '"' {
				i++
				continue
			}
			if i+1 >= len(src) || unhex(src[i]) < 0 || unhex(src[i+1]) < 0 {
				return -1
			}
			i += 2
			continue
		}
		i++
	}
	return -1
}

func balance(src []byte, openAt int, open, close byte) int {
	if openAt >= len(src) || src[openAt] != open {
		return -1
	}
	depth := 0
	i := openAt
	for i < len(src) {
		c := src[i]
		if c == '"' {
			end := scanString(src, i)
			if end < 0 {
				return -1
			}
			i = end
			continue
		}
		if c == ';' {
			for i < len(src) && src[i] != '\n' {
				i++
			}
			continue
		}
		if c == open {
			depth++
		} else if c == close {
			depth--
			if depth == 0 {
				return i + 1
			}
		}
		i++
	}
	return -1
}
