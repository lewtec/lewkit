package makefile

import (
	"strings"

	xparse "github.com/lewtec/lewkit/x/parse"
)

// maker is the scanner memory for one file.
// A tab is a recipe only when recipeOK is true: the open frame is not the
// top file, or the previous statement is a rule with targets.
type maker struct {
	conditionDepth int
	defineDepth    int
	afterRule      bool
}

func makerOf(st *xparse.LexState) *maker {
	if st.Data == nil {
		st.Data = &maker{}
	}
	return st.Data.(*maker)
}

func (m *maker) recipeOK() bool {
	return m.defineDepth == 0 && (m.conditionDepth > 0 || m.afterRule)
}

// note updates recipe context from a statement that is not itself a recipe.
// A target-specific assignment is kindAssign, so it does not arm recipes.
func (m *maker) note(k lineKind) {
	switch k {
	case kindIf:
		m.conditionDepth++
		m.afterRule = false
	case kindElse:
		m.afterRule = false
	case kindEndif:
		if m.conditionDepth > 0 {
			m.conditionDepth--
		}
		m.afterRule = false
	case kindRule:
		m.afterRule = true
	case kindDefine:
		m.defineDepth = 1
		m.afterRule = false
	default:
		m.afterRule = false
	}
}

func scan(src []byte, at int, _ func(xparse.SymbolID) bool, st *xparse.LexState) (xparse.SymbolID, int, int, bool) {
	m := makerOf(st)
	for {
		if at >= len(src) {
			return xparse.SymEOF, at, at, true
		}
		if m.defineDepth > 0 {
			return scanBody(src, at, st, m)
		}
		if m.recipeOK() && src[at] == '\t' {
			return scanRecipe(src, at, st)
		}
		end, next, ok := readLogical(src, at)
		if !ok {
			st.At = at
			return 0, at, at, false
		}
		if next <= at {
			st.At = at
			return 0, at, at, false
		}
		raw := string(src[at:end])
		if cookedLine(raw) == "" {
			at = next
			continue
		}
		k := classOf(cookedLine(raw))
		m.note(k)
		id, ok := st.Symbol(termOf(k).String())
		if !ok {
			st.At = at
			return 0, at, at, false
		}
		st.At = next
		return id, at, end, true
	}
}

func scanRecipe(src []byte, at int, st *xparse.LexState) (xparse.SymbolID, int, int, bool) {
	j := at
	for j < len(src) && src[j] != '\n' {
		j++
	}
	next := j
	if j < len(src) {
		next = j + 1
	}
	id, ok := st.Symbol(TermRecipe.String())
	if !ok {
		st.At = at
		return 0, at, at, false
	}
	st.At = next
	return id, at, j, true
}

func scanBody(src []byte, at int, st *xparse.LexState, m *maker) (xparse.SymbolID, int, int, bool) {
	end, next, ok := readLogical(src, at)
	if !ok || next <= at {
		st.At = at
		return 0, at, at, false
	}
	kind := TermBody
	if end == at {
		id, ok := st.Symbol(kind.String())
		if !ok {
			st.At = at
			return 0, at, at, false
		}
		st.At = next
		return id, at, next, true
	}
	raw := string(src[at:end])
	line := collapse(raw)
	if !strings.HasPrefix(line, "\t") {
		word, rest := cutWord(trimLeft(line))
		switch word {
		case "define":
			m.defineDepth++
		case "endef":
			extra := trimSpace(removeComments(rest)) != ""
			if m.defineDepth == 1 {
				m.defineDepth = 0
				m.afterRule = false
				kind = TermEndef
				break
			}
			if extra {
				st.At = at
				return 0, at, at, false
			}
			m.defineDepth--
		}
	}
	id, ok := st.Symbol(kind.String())
	if !ok {
		st.At = at
		return 0, at, at, false
	}
	st.At = next
	return id, at, end, true
}

// readLogical reads one backslash-continued line.
// src[at:end] is the joined text, without the terminating newline.
// next is the byte after that newline, or end at EOF.
// An odd trailing backslash at EOF is not a line.
func readLogical(src []byte, at int) (end, next int, ok bool) {
	if at >= len(src) {
		return at, at, false
	}
	i := at
	for {
		j := i
		for j < len(src) && src[j] != '\n' {
			j++
		}
		odd := trailingBackslashes(string(src[i:j]))%2 == 1
		if j >= len(src) {
			if odd {
				return at, j, false
			}
			return j, j, true
		}
		if !odd {
			return j, j + 1, true
		}
		i = j + 1
		if i == len(src) {
			return i, i, true
		}
	}
}

type lineKind int

const (
	kindBlank lineKind = iota
	kindAssign
	kindDefine
	kindUndefine
	kindIf
	kindElse
	kindEndif
	kindInclude
	kindVPath
	kindExport
	kindRule
	kindUnknown
)

// classOf is the statement decision. Assignment wins over a keyword.
// A single-colon target assignment is not a rule, so a following tab
// is not a recipe.
func classOf(line string) lineKind {
	if line == "" {
		return kindBlank
	}
	if a, ok := assignmentKind(line); ok {
		switch a.kind {
		case assignDefine:
			return kindDefine
		case assignUndefine:
			return kindUndefine
		default:
			return kindAssign
		}
	}
	word, _ := cutWord(line)
	switch word {
	case "ifeq", "ifneq", "ifdef", "ifndef":
		return kindIf
	case "else":
		return kindElse
	case "endif":
		return kindEndif
	case "include", "-include", "sinclude":
		return kindInclude
	case "vpath":
		return kindVPath
	case "export", "unexport":
		return kindExport
	case "undefine":
		return kindUndefine
	}
	if sep, ok := findRuleSep(line); ok {
		if !sep.double && !sep.group {
			right := line[sep.end:]
			if a, ok := assignmentKind(trimLeft(right)); ok && a.kind == assignVar {
				return kindAssign
			}
		}
		return kindRule
	}
	return kindUnknown
}

func termOf(k lineKind) Term {
	switch k {
	case kindAssign:
		return TermAssign
	case kindDefine:
		return TermDefine
	case kindUndefine:
		return TermUndefine
	case kindIf:
		return TermIf
	case kindElse:
		return TermElse
	case kindEndif:
		return TermEndif
	case kindInclude:
		return TermInclude
	case kindVPath:
		return TermVPath
	case kindExport:
		return TermExport
	case kindRule:
		return TermRule
	default:
		return TermUnknown
	}
}

func cookedLine(raw string) string {
	return trimLeft(removeComments(collapse(raw)))
}

func bodyLine(text string) string {
	if text == "\n" {
		return ""
	}
	return collapse(text)
}

func recipeText(raw string) string {
	if strings.HasPrefix(raw, "\t") {
		return raw[1:]
	}
	return raw
}
