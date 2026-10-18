package c

import (
	"unicode"

	xparse "github.com/lewtec/lewkit/x/parse"
)

// keywords are the spellings the scanner classifies before an identifier.
// A typedef name is a different token; the scanner records names it has seen.
var keywords = map[string]struct{}{
	"void": {}, "bool": {}, "_Bool": {}, "char": {}, "short": {}, "int": {}, "long": {},
	"float": {}, "double": {}, "signed": {}, "unsigned": {}, "struct": {}, "union": {},
	"enum": {}, "typedef": {}, "const": {}, "volatile": {}, "static": {}, "extern": {},
	"inline": {}, "sizeof": {}, "if": {}, "else": {}, "while": {}, "for": {}, "return": {},
	"break": {}, "continue": {}, "switch": {}, "do": {}, "goto": {}, "restrict": {},
	"auto": {}, "register": {}, "_Noreturn": {}, "_Thread_local": {}, "_Alignas": {},
	"_Atomic": {}, "_Generic": {},
}

var multiPunct = []string{
	"...", "<<=", ">>=", "<<", ">>", "<=", ">=", "==", "!=", "&&", "||",
	"+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "->", "++", "--",
}

func scan(src []byte, at int, _ func(xparse.SymbolID) bool, st *xparse.LexState) (xparse.SymbolID, int, int, bool) {
	lx := lexerFrom(st)
	return lx.next(st, src, at)
}

type scanMode int

const (
	modeNorm scanMode = iota
	modeDefName
	modeDefBody
)

type tdKind int

const (
	tdSpec tdKind = iota
	tdAggro
	tdBody
	tdDecl
	tdGroup
	tdParams
	tdArray
)

type tdFrame struct {
	kind  tdKind
	step  int
	star  bool
	depth int
}

type lexer struct {
	bol     bool
	prev    string
	mode    scanMode
	td      []tdFrame
	pending bool
	tdName  string
	types   map[string]bool
}

func lexerFrom(st *xparse.LexState) *lexer {
	if st.Data != nil {
		return st.Data.(*lexer)
	}
	lx := &lexer{bol: true, types: map[string]bool{}}
	st.Data = lx
	return lx
}

func (lx *lexer) next(st *xparse.LexState, src []byte, at int) (xparse.SymbolID, int, int, bool) {
	i := at
	for {
		if i >= len(src) {
			if lx.mode == modeDefBody {
				lx.mode = modeNorm
				lx.bol = true
				return lx.emit(st, string(TermNewline), "", i, i)
			}
			return xparse.SymEOF, i, i, true
		}
		c := src[i]
		switch c {
		case ' ', '\t', '\r':
			i++
			continue
		case '\n':
			if lx.mode == modeDefBody {
				lx.mode = modeNorm
				lx.bol = true
				return lx.emit(st, string(TermNewline), "", i, i+1)
			}
			i++
			lx.bol = true
			continue
		case '/':
			if i+1 < len(src) && (src[i+1] == '/' || src[i+1] == '*') {
				i = skipComment(src, i)
				continue
			}
		}
		break
	}
	if lx.bol && src[i] == '#' {
		return lx.directive(st, src, i)
	}
	lx.bol = false
	if lx.mode == modeDefName {
		return lx.defName(st, src, i)
	}
	return lx.token(st, src, i)
}

func skipComment(src []byte, i int) int {
	if i+1 < len(src) && src[i+1] == '/' {
		i += 2
		for i < len(src) && src[i] != '\n' {
			i++
		}
		return i
	}
	i += 2
	for i+1 < len(src) && !(src[i] == '*' && src[i+1] == '/') {
		i++
	}
	if i+1 < len(src) {
		i += 2
	} else {
		i = len(src)
	}
	return i
}

func (lx *lexer) directive(st *xparse.LexState, src []byte, i int) (xparse.SymbolID, int, int, bool) {
	start := i
	i++
	i = skipHoriz(src, i)
	word, i := takeIdent(src, i)
	i = skipHoriz(src, i)
	switch word {
	case "include":
		end := lineEnd(src, i)
		lx.bol = true
		return lx.emit(st, string(TermInclude), string(src[start:end]), start, end)
	case "define":
		_, j := takeIdent(src, i)
		if j > i && j < len(src) && src[j] == '(' {
			end := lineEnd(src, start)
			lx.bol = true
			return lx.emit(st, string(TermInvalid), string(src[start:end]), start, end)
		}
		wordEnd := start + 1
		wordEnd = skipHoriz(src, wordEnd)
		_, wordEnd = takeIdent(src, wordEnd)
		lx.mode = modeDefName
		lx.bol = false
		return lx.emit(st, string(TermDefine), "define", start, wordEnd)
	default:
		end := lineEnd(src, start)
		lx.bol = true
		return lx.emit(st, string(TermInvalid), string(src[start:end]), start, end)
	}
}

func (lx *lexer) defName(st *xparse.LexState, src []byte, i int) (xparse.SymbolID, int, int, bool) {
	word, j := takeIdent(src, i)
	if word == "" {
		return 0, i, i, false
	}
	lx.mode = modeDefBody
	return lx.emit(st, string(TermIdentifier), word, i, j)
}

func (lx *lexer) token(st *xparse.LexState, src []byte, i int) (xparse.SymbolID, int, int, bool) {
	c := src[i]
	if isIdentStart(c) {
		j := identEnd(src, i)
		word := string(src[i:j])
		name := lx.classify(word)
		return lx.emit(st, name, word, i, j)
	}
	if c >= '0' && c <= '9' {
		j := numberEnd(src, i)
		return lx.emit(st, string(TermNumber), string(src[i:j]), i, j)
	}
	if c == '"' {
		j := quoteEnd(src, i)
		if j < 0 {
			return 0, i, i, false
		}
		return lx.emit(st, string(TermString), string(src[i:j]), i, j)
	}
	if c == '\'' {
		j := quoteEnd(src, i)
		if j < 0 {
			return 0, i, i, false
		}
		return lx.emit(st, string(TermCharacterLiteral), string(src[i:j]), i, j)
	}
	for _, op := range multiPunct {
		if hasPrefix(src, i, op) {
			return lx.emit(st, op, op, i, i+len(op))
		}
	}
	switch c {
	case '(', ')', '[', ']', '{', '}', ',', ';', '*', '=', '<', '>', '!', '~', '+', '-', '&', '|', '^', '/', '%', '.', ':':
		s := string(c)
		return lx.emit(st, s, s, i, i+1)
	default:
		return 0, i, i + 1, false
	}
}

func (lx *lexer) classify(word string) string {
	if _, ok := keywords[word]; ok {
		return word
	}
	switch lx.prev {
	case ".", "->", "struct", "union", "enum":
		return string(TermIdentifier)
	}
	if lx.types[word] {
		return string(TermTypeName)
	}
	return string(TermIdentifier)
}

func (lx *lexer) emit(st *xparse.LexState, name, text string, a, b int) (xparse.SymbolID, int, int, bool) {
	id, ok := st.Symbol(name)
	if !ok {
		return 0, a, b, false
	}
	shown := text
	if shown == "" {
		shown = name
	}
	lx.note(name, shown)
	lx.prev = name
	return id, a, b, true
}

func (lx *lexer) note(kind, text string) {
	for {
		if !lx.feed(kind, text) {
			return
		}
	}
}

func (lx *lexer) feed(kind, text string) bool {
	if len(lx.td) == 0 {
		if kind == string(TermTypedef) {
			lx.td = append(lx.td, tdFrame{kind: tdSpec})
			lx.pending = true
			lx.tdName = ""
			return false
		}
		if lx.pending && (kind == string(TermSemicolon) || kind == string(TermLeftBrace)) {
			if kind == string(TermSemicolon) && lx.tdName != "" {
				lx.types[lx.tdName] = true
			}
			lx.pending = false
			lx.tdName = ""
		}
		return false
	}
	fr := &lx.td[len(lx.td)-1]
	switch fr.kind {
	case tdSpec:
		if isSpecWord(kind) {
			return false
		}
		switch kind {
		case string(TermStruct), string(TermUnion), string(TermEnum):
			lx.td = append(lx.td, tdFrame{kind: tdAggro})
			return false
		default:
			lx.td = lx.td[:len(lx.td)-1]
			lx.td = append(lx.td, tdFrame{kind: tdDecl})
			return true
		}
	case tdAggro:
		if fr.step == 0 {
			fr.step = 1
			if kind == string(TermIdentifier) {
				return false
			}
			return true
		}
		if kind == string(TermLeftBrace) {
			lx.td = lx.td[:len(lx.td)-1]
			lx.td = append(lx.td, tdFrame{kind: tdBody, depth: 1})
			return false
		}
		lx.td = lx.td[:len(lx.td)-1]
		return true
	case tdBody:
		switch kind {
		case string(TermLeftBrace):
			fr.depth++
		case string(TermRightBrace):
			fr.depth--
			if fr.depth == 0 {
				lx.td = lx.td[:len(lx.td)-1]
			}
		}
		return false
	case tdDecl:
		return lx.feedDecl(fr, kind, text)
	case tdGroup:
		if kind == string(TermRightParenthesis) {
			lx.td = lx.td[:len(lx.td)-1]
			return false
		}
		lx.abort()
		return false
	case tdParams:
		switch kind {
		case string(TermLeftParenthesis):
			fr.depth++
		case string(TermRightParenthesis):
			fr.depth--
			if fr.depth == 0 {
				lx.td = lx.td[:len(lx.td)-1]
			}
		}
		return false
	case tdArray:
		switch kind {
		case string(TermLeftBracket):
			fr.depth++
		case string(TermRightBracket):
			fr.depth--
			if fr.depth == 0 {
				lx.td = lx.td[:len(lx.td)-1]
			}
		}
		return false
	default:
		lx.abort()
		return false
	}
}

func (lx *lexer) feedDecl(fr *tdFrame, kind, text string) bool {
	switch fr.step {
	case 0:
		if kind == string(TermStar) {
			fr.star = true
			return false
		}
		if fr.star && (kind == string(TermConst) || kind == string(TermVolatile) || kind == string(TermRestrict)) {
			return false
		}
		fr.step = 1
		return true
	case 1:
		switch kind {
		case string(TermIdentifier):
			lx.tdName = text
			fr.step = 2
			return false
		case string(TermLeftParenthesis):
			fr.step = 2
			lx.td = append(lx.td, tdFrame{kind: tdGroup}, tdFrame{kind: tdDecl})
			return false
		default:
			fr.step = 2
			return true
		}
	default:
		switch kind {
		case string(TermLeftBracket):
			lx.td = append(lx.td, tdFrame{kind: tdArray, depth: 1})
			return false
		case string(TermLeftParenthesis):
			lx.td = append(lx.td, tdFrame{kind: tdParams, depth: 1})
			return false
		default:
			lx.td = lx.td[:len(lx.td)-1]
			return true
		}
	}
}

func (lx *lexer) abort() {
	lx.td = nil
	lx.pending = false
	lx.tdName = ""
}

func isSpecWord(kind string) bool {
	switch kind {
	case "const", "volatile", "inline", "static", "extern", "typedef",
		"void", "bool", "_Bool", "char", "short", "int", "long", "float", "double",
		"signed", "unsigned", "restrict", "auto", "register", "_Noreturn", "_Thread_local",
		"_Alignas", "_Atomic", "_Generic", "type_name":
		return true
	default:
		return false
	}
}

func skipHoriz(src []byte, i int) int {
	for i < len(src) && (src[i] == ' ' || src[i] == '\t') {
		i++
	}
	return i
}

func lineEnd(src []byte, i int) int {
	for i < len(src) && src[i] != '\n' {
		i++
	}
	return i
}

func takeIdent(src []byte, i int) (string, int) {
	j := identEnd(src, i)
	if j == i {
		return "", i
	}
	return string(src[i:j]), j
}

func identEnd(src []byte, i int) int {
	if i >= len(src) || !isIdentStart(src[i]) {
		return i
	}
	i++
	for i < len(src) && isIdentCont(src[i]) {
		i++
	}
	return i
}

func numberEnd(src []byte, i int) int {
	if i < len(src) && src[i] == '0' && i+1 < len(src) && (src[i+1] == 'x' || src[i+1] == 'X') {
		i += 2
		for i < len(src) && isHex(src[i]) {
			i++
		}
	} else {
		for i < len(src) && src[i] >= '0' && src[i] <= '9' {
			i++
		}
		if i < len(src) && src[i] == '.' {
			i++
			for i < len(src) && src[i] >= '0' && src[i] <= '9' {
				i++
			}
		}
		if i < len(src) && (src[i] == 'e' || src[i] == 'E') {
			i++
			if i < len(src) && (src[i] == '+' || src[i] == '-') {
				i++
			}
			for i < len(src) && src[i] >= '0' && src[i] <= '9' {
				i++
			}
		}
	}
	for i < len(src) {
		s := src[i]
		if s == 'u' || s == 'U' || s == 'l' || s == 'L' || s == 'f' || s == 'F' {
			i++
			continue
		}
		break
	}
	return i
}

func quoteEnd(src []byte, i int) int {
	if i >= len(src) {
		return -1
	}
	q := src[i]
	i++
	for i < len(src) {
		if src[i] == '\n' {
			return -1
		}
		if src[i] == '\\' {
			if i+1 >= len(src) || src[i+1] == '\n' {
				return -1
			}
			i += 2
			continue
		}
		if src[i] == q {
			return i + 1
		}
		i++
	}
	return -1
}

func hasPrefix(src []byte, at int, word string) bool {
	if at < 0 || at+len(word) > len(src) {
		return false
	}
	for i := 0; i < len(word); i++ {
		if src[at+i] != word[i] {
			return false
		}
	}
	return true
}

func isIdentStart(c byte) bool {
	return c == '_' || unicode.IsLetter(rune(c))
}

func isIdentCont(c byte) bool {
	return c == '_' || unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c))
}

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
