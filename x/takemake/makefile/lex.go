package makefile

import (
	"strings"

	xparse "github.com/lewtec/lewkit/x/parse"
)

const (
	modeRecipe xparse.Mode = iota + 1
	modeCondition
	modeDefine
)

// makeModes is the scanner memory. A tab is a recipe when a rule is open or
// a condition frame is open, and define nesting counts. A target-specific
// assignment is TermAssign, so it does not arm recipes.
var makeModes = xparse.Modes[Term]{
	{On: TermIf, Enter: modeCondition, Depth: true},
	{On: TermIf, Leave: modeRecipe},
	{On: TermElse, Leave: modeRecipe},
	{On: TermEndif, Leave: modeCondition, Depth: true},
	{On: TermEndif, Leave: modeRecipe},
	{On: TermRule, Enter: modeRecipe},
	{On: TermDefine, Enter: modeDefine, Depth: true},
	{On: TermDefine, Leave: modeRecipe},
	{On: TermEndef, Leave: modeDefine, Depth: true},
	{On: TermEndef, Leave: modeRecipe},
	{On: TermAssign, Leave: modeRecipe},
	{On: TermUndefine, Leave: modeRecipe},
	{On: TermInclude, Leave: modeRecipe},
	{On: TermVPath, Leave: modeRecipe},
	{On: TermExport, Leave: modeRecipe},
	{On: TermUnknown, Leave: modeRecipe},
	{In: modeRecipe, Not: modeDefine, When: xparse.Byte('\t'), Emit: TermRecipe, Span: recipeSpan},
	{In: modeCondition, Not: modeDefine, When: xparse.Byte('\t'), Emit: TermRecipe, Span: recipeSpan},
}

func scan(src []byte, at int, valid func(xparse.SymbolID) bool, st *xparse.LexState) (xparse.SymbolID, int, int, bool) {
	return xparse.Drive(makeModes, readStatement)(src, at, valid, st)
}

func readStatement(src []byte, at int, st *xparse.LexState) (Term, int, int, bool) {
	if at >= len(src) {
		return "", at, at, true
	}
	mem := xparse.Remember(st, makeModes)
	if mem.Active(modeDefine) {
		return readBody(src, at, st, mem)
	}
	for {
		if at >= len(src) {
			return "", at, at, true
		}
		end, next, ok := readLogical(src, at)
		if !ok || next <= at {
			st.At = at
			return "", at, at, false
		}
		raw := string(src[at:end])
		if cookedLine(raw) == "" {
			at = next
			continue
		}
		st.At = next
		return termOf(classOf(cookedLine(raw))), at, end, true
	}
}

func recipeSpan(src []byte, at int, st *xparse.LexState) (int, bool) {
	j := at
	for j < len(src) && src[j] != '\n' {
		j++
	}
	next := j
	if j < len(src) {
		next = j + 1
	}
	st.At = next
	return j, true
}

func readBody(src []byte, at int, st *xparse.LexState, mem *xparse.ModeMemory[Term]) (Term, int, int, bool) {
	end, next, ok := readLogical(src, at)
	if !ok || next <= at {
		st.At = at
		return "", at, at, false
	}
	kind := TermBody
	if end == at {
		st.At = next
		return kind, at, next, true
	}
	raw := string(src[at:end])
	line := collapse(raw)
	if !strings.HasPrefix(line, "\t") {
		word, rest := cutWord(trimLeft(line))
		switch word {
		case "define":
			mem.Enter(modeDefine, true)
		case "endef":
			extra := trimSpace(removeComments(rest)) != ""
			if mem.Depth(modeDefine) == 1 {
				kind = TermEndef
				break
			}
			if extra {
				st.At = at
				return "", at, at, false
			}
			mem.Leave(modeDefine, true)
		}
	}
	st.At = next
	return kind, at, end, true
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
