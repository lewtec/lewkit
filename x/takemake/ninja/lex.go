package ninja

import (
	"fmt"
	"strings"

	xparse "github.com/lewtec/lewkit/x/parse"
)

// scanSrc ports the ninja 1.12 lexer. Indent and values are modes: a value
// matcher in the same state as an identifier would keep the earlier symbol,
// and the other parse would never see its token. LexState.Data holds that mode.
func scanSrc(src []byte, _ int, _ func(xparse.SymbolID) bool, st *xparse.LexState) (xparse.SymbolID, int, int, bool) {
	l := lexerFrom(st, src)
	if l.failed {
		return 0, l.errAt, l.errAt, false
	}
	tok, err := l.rawNext()
	if err != nil {
		l.failed = true
		if l.errAt < 0 || l.errAt > len(src) {
			l.errAt = 0
		}
		st.At = l.errAt
		return 0, l.errAt, l.errAt, false
	}
	st.At = l.i
	if tok.eof {
		return xparse.SymEOF, tok.a, tok.b, true
	}
	id, ok := st.Symbol(tok.term.String())
	if !ok {
		l.failed = true
		st.At = tok.a
		return 0, tok.a, tok.b, false
	}
	return id, tok.a, tok.b, true
}

func lexerFrom(st *xparse.LexState, src []byte) *lexer {
	if st.Data != nil {
		return st.Data.(*lexer)
	}
	l := &lexer{s: string(src), line: 1}
	st.Data = l
	return l
}

type phase int

const (
	phaseStatement phase = iota
	phaseIdentifier
	phaseEqual
	phaseValue
	phaseNeedNewline
	phaseBlock
	phaseBuildOutput
	phaseBuildRule
	phaseBuildInput
	phaseDefault
	phaseSinglePath
)

type rawTok struct {
	term Term
	a, b int
	eof  bool
}

type lexer struct {
	s           string
	i           int
	line        int
	phase       phase
	substate    int
	saw         bool
	what        string
	next        phase
	newlineNext phase
	tokenStart  int
	tokenLine   int
	errAt       int
	failed      bool
}

func (l *lexer) rawNext() (rawTok, error) {
	switch l.phase {
	case phaseStatement:
		tok, err := l.readToken()
		if err != nil {
			return rawTok{}, err
		}
		if tok.eof {
			return tok, nil
		}
		return l.dispatch(tok)
	case phaseIdentifier:
		return l.nextName()
	case phaseEqual:
		return l.expect(TermEqual, phaseValue)
	case phaseValue:
		return l.nextValue()
	case phaseNeedNewline:
		return l.expect(TermNewline, l.newlineNext)
	case phaseBlock:
		return l.nextBlock()
	case phaseBuildOutput:
		return l.nextBuildOutput()
	case phaseBuildRule:
		l.what = "build command name"
		l.next = phaseBuildInput
		return l.nextName()
	case phaseBuildInput:
		return l.nextBuildInput()
	case phaseDefault:
		return l.nextDefault()
	case phaseSinglePath:
		return l.nextOne()
	default:
		return rawTok{}, l.at(l.line, "lexing error")
	}
}

func (l *lexer) dispatch(tok rawTok) (rawTok, error) {
	switch tok.term {
	case TermNewline:
		l.phase = phaseStatement
		return tok, nil
	case TermIdentifier:
		l.phase = phaseEqual
		l.newlineNext = phaseStatement
		return tok, nil
	case TermRule:
		return l.named(tok, "rule name"), nil
	case TermPool:
		return l.named(tok, "pool name"), nil
	case TermBuild:
		l.phase = phaseBuildOutput
		l.substate = 0
		l.saw = false
		return tok, nil
	case TermDefault:
		l.phase = phaseDefault
		l.substate = 0
		l.saw = false
		return tok, nil
	case TermInclude, TermSubninja:
		l.phase = phaseSinglePath
		l.substate = 0
		return tok, nil
	case TermIndent:
		return rawTok{}, l.tokenErr("unexpected indent")
	default:
		return rawTok{}, l.unexpected(tok)
	}
}

func (l *lexer) named(tok rawTok, what string) rawTok {
	l.phase = phaseIdentifier
	l.what = what
	l.next = phaseNeedNewline
	l.newlineNext = phaseBlock
	return tok
}

func (l *lexer) nextName() (rawTok, error) {
	a, b, err := l.readIdent(l.what)
	if err != nil {
		return rawTok{}, err
	}
	if l.next == phaseBuildInput {
		l.substate = 0
	}
	l.phase = l.next
	return rawTok{term: TermIdentifier, a: a, b: b}, nil
}

func (l *lexer) nextValue() (rawTok, error) {
	ps, a, b, err := l.readEval(false, false)
	if err != nil {
		return rawTok{}, err
	}
	if len(ps) == 0 {
		return l.expect(TermNewline, l.newlineNext)
	}
	l.phase = phaseNeedNewline
	return rawTok{term: TermValue, a: a, b: b}, nil
}

func (l *lexer) nextBlock() (rawTok, error) {
	tok, err := l.readToken()
	if err != nil {
		return rawTok{}, err
	}
	if tok.eof {
		return tok, nil
	}
	if tok.term == TermIndent {
		l.phase = phaseIdentifier
		l.what = "variable name"
		l.next = phaseEqual
		l.newlineNext = phaseBlock
		return tok, nil
	}
	return l.dispatch(tok)
}

func (l *lexer) nextBuildOutput() (rawTok, error) {
	for {
		ps, a, b, err := l.readEval(true, false)
		if err != nil {
			return rawTok{}, err
		}
		if len(ps) > 0 {
			l.saw = true
			return rawTok{term: TermPath, a: a, b: b}, nil
		}
		if l.substate != 0 {
			if !l.saw {
				return rawTok{}, l.tokenErr("expected path")
			}
			return l.expect(TermColon, phaseBuildRule)
		}
		tok, err := l.readToken()
		if err != nil {
			return rawTok{}, err
		}
		if !tok.eof && tok.term == TermPipe {
			l.substate = 1
			return tok, nil
		}
		if !l.saw {
			return rawTok{}, l.tokenErr("expected path")
		}
		if tok.eof || tok.term != TermColon {
			return rawTok{}, l.wantGot(TermColon, tok)
		}
		l.phase = phaseBuildRule
		return tok, nil
	}
}

func (l *lexer) nextBuildInput() (rawTok, error) {
	for {
		ps, a, b, err := l.readEval(true, false)
		if err != nil {
			return rawTok{}, err
		}
		if len(ps) > 0 {
			return rawTok{term: TermPath, a: a, b: b}, nil
		}
		tok, err := l.readToken()
		if err != nil {
			return rawTok{}, err
		}
		if next, ok := l.inputMark(tok); ok {
			l.substate = next
			return tok, nil
		}
		if tok.eof || tok.term != TermNewline {
			return rawTok{}, l.wantGot(TermNewline, tok)
		}
		l.phase = phaseBlock
		l.substate = 0
		return tok, nil
	}
}

// inputMark is the next dependency list, in ninja's order: | then || then |@.
func (l *lexer) inputMark(tok rawTok) (int, bool) {
	if tok.eof {
		return 0, false
	}
	switch l.substate {
	case 0:
		switch tok.term {
		case TermPipe:
			return 1, true
		case TermDoublePipe:
			return 2, true
		case TermPipeAtSign:
			return 3, true
		}
	case 1:
		switch tok.term {
		case TermDoublePipe:
			return 2, true
		case TermPipeAtSign:
			return 3, true
		}
	case 2:
		if tok.term == TermPipeAtSign {
			return 3, true
		}
	}
	return 0, false
}

func (l *lexer) nextDefault() (rawTok, error) {
	for {
		if l.substate != 0 {
			return l.expect(TermNewline, phaseStatement)
		}
		ps, a, b, err := l.readEval(true, false)
		if err != nil {
			return rawTok{}, err
		}
		if len(ps) > 0 {
			l.saw = true
			return rawTok{term: TermPath, a: a, b: b}, nil
		}
		if !l.saw {
			return rawTok{}, l.tokenErr("expected target name")
		}
		l.substate = 1
	}
}

func (l *lexer) nextOne() (rawTok, error) {
	if l.substate == 0 {
		ps, a, b, err := l.readEval(true, false)
		if err != nil {
			return rawTok{}, err
		}
		if len(ps) == 0 {
			return rawTok{}, l.tokenErr("expected path")
		}
		l.substate = 1
		return rawTok{term: TermPath, a: a, b: b}, nil
	}
	l.substate = 0
	return l.expect(TermNewline, phaseStatement)
}

func (l *lexer) expect(want Term, next phase) (rawTok, error) {
	tok, err := l.readToken()
	if err != nil {
		return rawTok{}, err
	}
	if tok.eof || tok.term != want {
		return rawTok{}, l.wantGot(want, tok)
	}
	l.phase = next
	return tok, nil
}

func (l *lexer) readToken() (rawTok, error) {
	for {
		l.tokenStart = l.i
		l.tokenLine = l.line
		if l.i >= len(l.s) {
			return rawTok{eof: true, a: l.i, b: l.i}, nil
		}
		if l.tryComment() {
			continue
		}
		if l.tryNewline() {
			return rawTok{term: TermNewline, a: l.tokenStart, b: l.i}, nil
		}
		if l.s[l.i] == ' ' {
			start := l.i
			for l.i < len(l.s) && l.s[l.i] == ' ' {
				l.i++
			}
			b := l.i
			l.eatWS()
			return rawTok{term: TermIndent, a: start, b: b}, nil
		}
		if l.s[l.i] == '\t' {
			return rawTok{}, l.tokenErr("tabs are not allowed, use spaces")
		}
		start := l.i
		switch l.s[l.i] {
		case '=':
			l.i++
			l.eatWS()
			return rawTok{term: TermEqual, a: start, b: start + 1}, nil
		case ':':
			l.i++
			l.eatWS()
			return rawTok{term: TermColon, a: start, b: start + 1}, nil
		case '|':
			term := TermPipe
			n := 1
			if strings.HasPrefix(l.s[l.i:], "|@") {
				term = TermPipeAtSign
				n = 2
			} else if strings.HasPrefix(l.s[l.i:], "||") {
				term = TermDoublePipe
				n = 2
			}
			l.i += n
			l.eatWS()
			return rawTok{term: term, a: start, b: start + n}, nil
		}
		if isVarChar(l.s[l.i]) {
			for l.i < len(l.s) && isVarChar(l.s[l.i]) {
				l.i++
			}
			term := keywordTerm(l.s[start:l.i])
			b := l.i
			l.eatWS()
			return rawTok{term: term, a: start, b: b}, nil
		}
		return rawTok{}, l.tokenErr("lexing error")
	}
}

func (l *lexer) readIdent(what string) (int, int, error) {
	l.tokenStart = l.i
	l.tokenLine = l.line
	if l.i >= len(l.s) || !isVarChar(l.s[l.i]) {
		return 0, 0, l.tokenErr("expected " + what)
	}
	a := l.i
	for l.i < len(l.s) && isVarChar(l.s[l.i]) {
		l.i++
	}
	b := l.i
	l.eatWS()
	return a, b, nil
}

// tryComment skips [ ]*#[^\n]*\n. A hash without a newline is not a comment.
func (l *lexer) tryComment() bool {
	j := l.i
	for j < len(l.s) && l.s[j] == ' ' {
		j++
	}
	if j >= len(l.s) || l.s[j] != '#' {
		return false
	}
	j++
	for j < len(l.s) && l.s[j] != '\n' {
		j++
	}
	if j >= len(l.s) {
		return false
	}
	j++
	l.line += strings.Count(l.s[l.i:j], "\n")
	l.i = j
	return true
}

func (l *lexer) tryNewline() bool {
	j := l.i
	for j < len(l.s) && l.s[j] == ' ' {
		j++
	}
	if j+1 < len(l.s) && l.s[j] == '\r' && l.s[j+1] == '\n' {
		l.line++
		l.i = j + 2
		return true
	}
	if j < len(l.s) && l.s[j] == '\n' {
		l.line++
		l.i = j + 1
		return true
	}
	return false
}

func (l *lexer) eatWS() {
	for {
		if l.i < len(l.s) && l.s[l.i] == ' ' {
			for l.i < len(l.s) && l.s[l.i] == ' ' {
				l.i++
			}
			continue
		}
		if l.i+1 < len(l.s) && l.s[l.i] == '$' && l.s[l.i+1] == '\n' {
			l.i += 2
			l.line++
			continue
		}
		if l.i+2 < len(l.s) && l.s[l.i] == '$' && l.s[l.i+1] == '\r' && l.s[l.i+2] == '\n' {
			l.i += 3
			l.line++
			continue
		}
		return
	}
}

// readEval reads a value (path false) or one path (path true).
// A path stops at an unescaped space, colon, pipe, or newline.
// The stopping newline is left for the next token. End of input is an error
// when eofOK is false, matching ninja's unexpected EOF on a missing final newline.
// eofOK is the lowerer rereading one token: the stopper is already outside the span.
func (l *lexer) readEval(path, eofOK bool) ([]Piece, int, int, error) {
	a := l.i
	var ps []Piece
	for {
		if l.i >= len(l.s) {
			if eofOK {
				break
			}
			return nil, 0, 0, l.at(l.line, "unexpected EOF")
		}
		c := l.s[l.i]
		if c == 0 {
			return nil, 0, 0, l.at(l.line, "unexpected EOF")
		}
		if c == '\n' || (c == '\r' && l.i+1 < len(l.s) && l.s[l.i+1] == '\n') {
			break
		}
		if c == '\r' {
			return nil, 0, 0, l.at(l.line, "lexing error")
		}
		if path && (c == ' ' || c == ':' || c == '|') {
			break
		}
		if c == '$' {
			if err := l.readDollar(&ps); err != nil {
				return nil, 0, 0, err
			}
			continue
		}
		start := l.i
		for l.i < len(l.s) {
			c = l.s[l.i]
			if c == '$' || c == 0 || c == '\n' || c == '\r' {
				break
			}
			if path && (c == ' ' || c == ':' || c == '|') {
				break
			}
			l.i++
		}
		addText(&ps, l.s[start:l.i])
	}
	b := l.i
	if path {
		l.eatWS()
	}
	return ps, a, b, nil
}

func (l *lexer) readDollar(ps *[]Piece) error {
	if l.i+1 >= len(l.s) || l.s[l.i+1] == 0 {
		return l.at(l.line, "bad $-escape (literal $ must be written as $$)")
	}
	switch l.s[l.i+1] {
	case '$':
		addText(ps, "$")
		l.i += 2
	case ' ':
		addText(ps, " ")
		l.i += 2
	case ':':
		addText(ps, ":")
		l.i += 2
	case '\n':
		l.i += 2
		l.line++
		l.skipSpaces()
	case '\r':
		if l.i+2 < len(l.s) && l.s[l.i+2] == '\n' {
			l.i += 3
			l.line++
			l.skipSpaces()
			break
		}
		return l.at(l.line, "bad $-escape (literal $ must be written as $$)")
	case '{':
		j := l.i + 2
		start := j
		for j < len(l.s) && isVarChar(l.s[j]) {
			j++
		}
		if j == start || j >= len(l.s) || l.s[j] != '}' {
			return l.at(l.line, "bad $-escape (literal $ must be written as $$)")
		}
		addVar(ps, l.s[start:j])
		l.i = j + 1
	default:
		if !isSimpleChar(l.s[l.i+1]) {
			return l.at(l.line, "bad $-escape (literal $ must be written as $$)")
		}
		j := l.i + 1
		for j < len(l.s) && isSimpleChar(l.s[j]) {
			j++
		}
		addVar(ps, l.s[l.i+1:j])
		l.i = j
	}
	return nil
}

func (l *lexer) skipSpaces() {
	for l.i < len(l.s) && l.s[l.i] == ' ' {
		l.i++
	}
}

func (l *lexer) at(line int, msg string) error {
	l.errAt = l.i
	return fmt.Errorf("%w: line %d: %s", errSyntax, line, msg)
}

func (l *lexer) tokenErr(msg string) error {
	line := l.tokenLine
	if line == 0 {
		line = l.line
	}
	l.errAt = l.tokenStart
	return fmt.Errorf("%w: line %d: %s", errSyntax, line, msg)
}

func (l *lexer) wantGot(want Term, tok rawTok) error {
	return l.tokenErr("expected " + termLabel(want) + ", got " + l.got(tok))
}

func (l *lexer) unexpected(tok rawTok) error {
	return l.tokenErr("unexpected " + l.got(tok))
}

func (l *lexer) got(tok rawTok) string {
	if tok.eof {
		return "eof"
	}
	return termLabel(tok.term)
}

func keywordTerm(s string) Term {
	switch s {
	case "build":
		return TermBuild
	case "pool":
		return TermPool
	case "rule":
		return TermRule
	case "default":
		return TermDefault
	case "include":
		return TermInclude
	case "subninja":
		return TermSubninja
	default:
		return TermIdentifier
	}
}

func termLabel(t Term) string {
	switch t {
	case TermIdentifier:
		return "identifier"
	case TermNewline:
		return "newline"
	case TermIndent:
		return "indent"
	case TermEqual:
		return "'='"
	case TermColon:
		return "':'"
	case TermPipe:
		return "'|'"
	case TermDoublePipe:
		return "'||'"
	case TermPipeAtSign:
		return "'|@'"
	case TermBuild:
		return "'build'"
	case TermPool:
		return "'pool'"
	case TermRule:
		return "'rule'"
	case TermDefault:
		return "'default'"
	case TermInclude:
		return "'include'"
	case TermSubninja:
		return "'subninja'"
	default:
		return "token"
	}
}

func evalText(s string, path bool) ([]Piece, error) {
	l := &lexer{s: s, line: 1}
	ps, _, end, err := l.readEval(path, true)
	if err != nil {
		return nil, err
	}
	if end != len(s) {
		return nil, errSyntax
	}
	if len(ps) == 0 {
		return nil, nil
	}
	return ps, nil
}

func firstLexErr(s string) error {
	l := &lexer{s: s, line: 1}
	limit := len(s) + 8
	if limit < 8 {
		limit = 8
	}
	for n := 0; n < limit; n++ {
		tok, err := l.rawNext()
		if err != nil {
			return err
		}
		if tok.eof {
			return nil
		}
	}
	return fmt.Errorf("%w: line %d: lexing error", errSyntax, l.line)
}

func isVarChar(c byte) bool {
	return isSimpleChar(c) || c == '.'
}

func isSimpleChar(c byte) bool {
	return c == '_' || c == '-' ||
		(c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9')
}

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isVarChar(s[i]) {
			return false
		}
	}
	return true
}

func isSimple(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isSimpleChar(s[i]) {
			return false
		}
	}
	return true
}

func isKeyword(s string) bool {
	return keywordTerm(s) != TermIdentifier
}

func wordRune(r rune) bool {
	return r < 128 && isVarChar(byte(r))
}
