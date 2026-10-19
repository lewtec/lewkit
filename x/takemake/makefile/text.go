package makefile

import (
	"strings"
	"unicode/utf8"
)

// collapse joins escaped newlines the way GNU make 4.3 does.
// An odd run of backslashes before a newline is a continuation:
// half of those backslashes stay (rounded down), surrounding
// whitespace disappears, and one space is inserted.
func collapse(line string) string {
	if !strings.Contains(line, "\n") {
		return line
	}
	var out strings.Builder
	out.Grow(len(line))
	in := 0
	for {
		rel := strings.IndexByte(line[in:], '\n')
		if rel < 0 {
			out.WriteString(line[in:])
			break
		}
		q := in + rel
		n := 0
		for k := q - 1; k >= 0 && line[k] == '\\'; k-- {
			n++
		}
		out.WriteString(line[in : q-n])
		out.WriteString(strings.Repeat("\\", n/2))
		in = q + 1
		if n%2 == 1 {
			for in < len(line) && (line[in] == ' ' || line[in] == '\t') {
				in++
			}
			s := strings.TrimRight(out.String(), " \t")
			out.Reset()
			out.WriteString(s)
			out.WriteByte(' ')
			continue
		}
		out.WriteByte('\n')
	}
	return out.String()
}

// removeComments cuts an unquoted # and drops the backslashes that quoted one.
// A # inside $() or ${} stays. This matches GNU make remove_comments.
func removeComments(s string) string {
	b := []byte(s)
	i := 0
	for i < len(b) {
		c := b[i]
		if c == '$' {
			if i+1 >= len(b) {
				break
			}
			nxt := b[i+1]
			i += 2
			if nxt == '(' || nxt == '{' {
				closer := byte(')')
				if nxt == '{' {
					closer = '}'
				}
				count := 1
				for i < len(b) && count > 0 {
					switch b[i] {
					case nxt:
						count++
					case closer:
						count--
					}
					i++
				}
			}
			continue
		}
		if c != '#' {
			i++
			continue
		}
		n := 0
		for j := i - 1; j >= 0 && b[j] == '\\'; j-- {
			n++
		}
		swallow := (n + 1) / 2
		if swallow > 0 {
			copy(b[i-swallow:], b[i:])
			b = b[:len(b)-swallow]
			i -= swallow
		}
		if n%2 == 0 {
			return string(b[:i])
		}
		i++
	}
	return string(b)
}

// escapeLine is the print form of s.
// removeComments(escapeLine(s)) == s, and the result does not end in an
// odd backslash, so a following line is not swallowed as a continuation.
func escapeLine(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 4)
	depth := 0
	var open byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if depth > 0 {
			b.WriteByte(c)
			switch c {
			case open:
				depth++
			case ')', '}':
				if (open == '(' && c == ')') || (open == '{' && c == '}') {
					depth--
				}
			}
			continue
		}
		if c == '$' && i+1 < len(s) {
			b.WriteByte('$')
			b.WriteByte(s[i+1])
			i++
			if s[i] == '(' || s[i] == '{' {
				depth = 1
				open = s[i]
			}
			continue
		}
		if c != '#' {
			b.WriteByte(c)
			continue
		}
		k := 0
		for j := i - 1; j >= 0 && s[j] == '\\'; j-- {
			k++
		}
		for n := 0; n < k+1; n++ {
			b.WriteByte('\\')
		}
		b.WriteByte('#')
	}
	n := trailingBackslashes(s)
	if n%2 == 1 {
		b.WriteString(strings.Repeat("\\", n))
		b.WriteByte('#')
	}
	return b.String()
}

func trailingBackslashes(s string) int {
	n := 0
	for i := len(s) - 1; i >= 0 && s[i] == '\\'; i-- {
		n++
	}
	return n
}

func trimLeft(s string) string {
	return strings.TrimLeft(s, " \t")
}

func trimRight(s string) string {
	return strings.TrimRight(s, " \t")
}

func trimSpace(s string) string {
	return strings.Trim(s, " \t")
}

// cutWord splits one makefile word.
// Spaces inside $() and ${} stay in the word. $$ is a literal dollar.
func cutWord(s string) (string, string) {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	start := i
	depth := 0
	var open byte
	for i < len(s) {
		if depth == 0 && s[i] == '$' && i+1 < len(s) {
			nxt := s[i+1]
			i += 2
			if nxt == '(' || nxt == '{' {
				depth = 1
				open = nxt
			}
			continue
		}
		if depth > 0 {
			switch s[i] {
			case open:
				depth++
			case ')', '}':
				if (open == '(' && s[i] == ')') || (open == '{' && s[i] == '}') {
					depth--
				}
			}
			i++
			continue
		}
		if s[i] == ' ' || s[i] == '\t' {
			break
		}
		i++
	}
	if start == i {
		return "", ""
	}
	return s[start:i], s[i:]
}

func words(s string) []string {
	var out []string
	for {
		w, rest := cutWord(s)
		if w == "" {
			return out
		}
		out = append(out, w)
		s = rest
	}
}

type varDef struct {
	name  string
	op    Op
	value string
}

// parseVariableDefinition recognizes name, operator, and value.
// A colon that is not := or ::= means this is not an assignment.
// The name is not expanded.
func parseVariableDefinition(s string) (varDef, bool) {
	p := 0
	for p < len(s) && (s[p] == ' ' || s[p] == '\t') {
		p++
	}
	if p >= len(s) {
		return varDef{}, false
	}
	nameAt := p
	endName := -1
	wspace := false
	for {
		if p >= len(s) {
			return varDef{}, false
		}
		c := s[p]
		p++
		if c == '#' {
			return varDef{}, false
		}
		if c == '$' {
			if p >= len(s) {
				return varDef{}, false
			}
			c = s[p]
			p++
			if c != '(' && c != '{' {
				continue
			}
			closer := byte(')')
			if c == '{' {
				closer = '}'
			}
			count := 1
			for p < len(s) && count > 0 {
				switch s[p] {
				case c:
					count++
				case closer:
					count--
				}
				p++
			}
			continue
		}
		if c == ' ' || c == '\t' {
			wspace = true
			endName = p - 1
			for p < len(s) && (s[p] == ' ' || s[p] == '\t') {
				p++
			}
			if p >= len(s) {
				return varDef{}, false
			}
			c = s[p]
			p++
		}
		if c == '=' {
			if endName < 0 {
				endName = p - 1
			}
			return varDef{name: s[nameAt:endName], op: OpRecursive, value: trimLeft(s[p:])}, true
		}
		if p < len(s) && s[p] == '=' {
			var op Op
			switch c {
			case ':':
				op = OpSimple
			case '+':
				op = OpAppend
			case '?':
				op = OpConditional
			case '!':
				op = OpShell
			default:
				if wspace {
					return varDef{}, false
				}
				continue
			}
			if endName < 0 {
				endName = p - 1
			}
			p++
			return varDef{name: s[nameAt:endName], op: op, value: trimLeft(s[p:])}, true
		}
		if c == ':' {
			if p+1 >= len(s) || s[p] != ':' || s[p+1] != '=' {
				return varDef{}, false
			}
			if endName < 0 {
				endName = p - 1
			}
			p += 2
			return varDef{name: s[nameAt:endName], op: OpImmediate, value: trimLeft(s[p:])}, true
		}
		if wspace {
			return varDef{}, false
		}
	}
}

func eachDepth0(s string, fn func(i int) bool) {
	depth := 0
	var open byte
	for i := 0; i < len(s); i++ {
		if depth == 0 && s[i] == '$' && i+1 < len(s) {
			nxt := s[i+1]
			i++
			if nxt == '(' || nxt == '{' {
				depth = 1
				open = nxt
			}
			continue
		}
		if depth > 0 {
			switch s[i] {
			case open:
				depth++
			case ')', '}':
				if (open == '(' && s[i] == ')') || (open == '{' && s[i] == '}') {
					depth--
				}
			}
			continue
		}
		if fn(i) {
			return
		}
	}
}

type ruleSep struct {
	start  int
	end    int
	double bool
	group  bool
}

func findRuleSep(s string) (ruleSep, bool) {
	found := -1
	eachDepth0(s, func(i int) bool {
		if s[i] != ':' {
			return false
		}
		if i+1 < len(s) && s[i+1] == '=' {
			return false
		}
		if i+2 < len(s) && s[i+1] == ':' && s[i+2] == '=' {
			return false
		}
		found = i
		return true
	})
	if found < 0 {
		return ruleSep{}, false
	}
	sep := ruleSep{start: found, end: found + 1}
	if found > 0 && s[found-1] == '&' {
		sep.start = found - 1
		sep.group = true
	}
	if sep.end < len(s) && s[sep.end] == ':' {
		sep.end++
		sep.double = true
	}
	return sep, true
}

func splitSemi(s string) (head, recipe string, ok bool) {
	at := -1
	eachDepth0(s, func(i int) bool {
		if s[i] != ';' {
			return false
		}
		n := 0
		for j := i - 1; j >= 0 && s[j] == '\\'; j-- {
			n++
		}
		if n%2 == 1 {
			return false
		}
		at = i
		return true
	})
	if at < 0 {
		return s, "", false
	}
	return s[:at], s[at+1:], true
}

func splitBar(s string) (head, order string, ok bool) {
	at := -1
	eachDepth0(s, func(i int) bool {
		if s[i] != '|' {
			return false
		}
		left := i == 0 || s[i-1] == ' ' || s[i-1] == '\t'
		right := i+1 == len(s) || s[i+1] == ' ' || s[i+1] == '\t'
		if !left || !right {
			return false
		}
		at = i
		return true
	})
	if at < 0 {
		return s, "", false
	}
	return s[:at], s[at+1:], true
}

func splitStatic(s string) (pattern, rest string, ok bool) {
	at := -1
	eachDepth0(s, func(i int) bool {
		if s[i] != ':' {
			return false
		}
		n := 0
		for j := i - 1; j >= 0 && s[j] == '\\'; j-- {
			n++
		}
		if n%2 == 1 {
			return false
		}
		at = i
		return true
	})
	if at < 0 {
		return "", s, false
	}
	return s[:at], s[at+1:], true
}

func quoteArg(s string) (string, error) {
	if !strings.ContainsAny(s, `"'`) {
		if strings.Contains(s, `"`) {
			return "", errSyntax
		}
		return `"` + s + `"`, nil
	}
	if !strings.Contains(s, "'") {
		return "'" + s + "'", nil
	}
	if !strings.Contains(s, `"`) {
		return `"` + s + `"`, nil
	}
	return "", errSyntax
}

func needsQuotedTest(left, right string) bool {
	return strings.ContainsAny(left, ",()\"'") || strings.ContainsAny(right, ",()\"'")
}

// short quotes a prefix of s for a syntax error.
func short(s string) string {
	s = trimSpace(s)
	if len(s) > 60 {
		for len(s) > 60 {
			_, size := utf8.DecodeLastRuneInString(s)
			if size <= 0 {
				break
			}
			s = s[:len(s)-size]
		}
	}
	return strconvQuote(s)
}

func strconvQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\', '"':
			b.WriteByte('\\')
			b.WriteByte(s[i])
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteByte(s[i])
		}
	}
	b.WriteByte('"')
	return b.String()
}
