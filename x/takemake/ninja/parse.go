package ninja

import (
	"fmt"
	"strings"
)

// parse reads a ninja manifest the way ninja 1.12's lexer does.
// Comments and blank lines are dropped. Include paths are not loaded.
func parse(s string) (*File, error) {
	l := &lexer{s: s, line: 1}
	f := New()
	for {
		switch l.readToken() {
		case teof:
			return f, nil
		case tnewline:
			continue
		case terror:
			return nil, l.err
		case tpool:
			p, err := l.parsePool()
			if err != nil {
				return nil, err
			}
			f.Stmts = append(f.Stmts, p)
		case tbuild:
			e, err := l.parseEdge()
			if err != nil {
				return nil, err
			}
			f.Stmts = append(f.Stmts, e)
		case trule:
			r, err := l.parseRule()
			if err != nil {
				return nil, err
			}
			f.Stmts = append(f.Stmts, r)
		case tdefault:
			d, err := l.parseDefault()
			if err != nil {
				return nil, err
			}
			f.Stmts = append(f.Stmts, d)
		case tinclude:
			inc, err := l.parseInclude(false)
			if err != nil {
				return nil, err
			}
			f.Stmts = append(f.Stmts, inc)
		case tsubninja:
			inc, err := l.parseInclude(true)
			if err != nil {
				return nil, err
			}
			f.Stmts = append(f.Stmts, inc)
		case tident:
			v, err := l.parseVariable(l.text)
			if err != nil {
				return nil, err
			}
			f.Stmts = append(f.Stmts, v)
		default:
			return nil, l.tokenErr("unexpected " + l.tok.String())
		}
	}
}

type token int

const (
	terror token = iota
	teof
	tnewline
	tindent
	tident
	tequals
	tcolon
	tpipe
	tpipe2
	tpipeat
	tbuild
	tpool
	trule
	tdefault
	tinclude
	tsubninja
)

func (t token) String() string {
	switch t {
	case terror:
		return "lexing error"
	case teof:
		return "eof"
	case tnewline:
		return "newline"
	case tindent:
		return "indent"
	case tident:
		return "identifier"
	case tequals:
		return "'='"
	case tcolon:
		return "':'"
	case tpipe:
		return "'|'"
	case tpipe2:
		return "'||'"
	case tpipeat:
		return "'|@'"
	case tbuild:
		return "'build'"
	case tpool:
		return "'pool'"
	case trule:
		return "'rule'"
	case tdefault:
		return "'default'"
	case tinclude:
		return "'include'"
	case tsubninja:
		return "'subninja'"
	default:
		return "token"
	}
}

type lexer struct {
	s        string
	i        int
	line     int
	tok      token
	tokStart int
	tokLine  int
	text     string
	err      error
}

func (l *lexer) at(line int, msg string) error {
	err := fmt.Errorf("%w: line %d: %s", errSyntax, line, msg)
	l.err = err
	return err
}

func (l *lexer) tokenErr(msg string) error {
	line := l.tokLine
	if line == 0 {
		line = l.line
	}
	return l.at(line, msg)
}

func (l *lexer) unread() {
	l.i = l.tokStart
	l.line = l.tokLine
}

func (l *lexer) readToken() token {
	if l.err != nil {
		return terror
	}
	for {
		l.tokStart = l.i
		l.tokLine = l.line
		if l.i >= len(l.s) {
			l.tok = teof
			return teof
		}
		if l.tryComment() {
			continue
		}
		if l.tryNewline() {
			l.tok = tnewline
			return tnewline
		}
		if l.s[l.i] == ' ' {
			for l.i < len(l.s) && l.s[l.i] == ' ' {
				l.i++
			}
			l.tok = tindent
			l.eatWS()
			return tindent
		}
		if l.s[l.i] == '\t' {
			l.tok = terror
			l.tokenErr("tabs are not allowed, use spaces")
			return terror
		}
		switch l.s[l.i] {
		case '=':
			l.i++
			l.tok = tequals
			l.eatWS()
			return tequals
		case ':':
			l.i++
			l.tok = tcolon
			l.eatWS()
			return tcolon
		case '|':
			if strings.HasPrefix(l.s[l.i:], "|@") {
				l.i += 2
				l.tok = tpipeat
			} else if strings.HasPrefix(l.s[l.i:], "||") {
				l.i += 2
				l.tok = tpipe2
			} else {
				l.i++
				l.tok = tpipe
			}
			l.eatWS()
			return l.tok
		}
		if isVarChar(l.s[l.i]) {
			start := l.i
			for l.i < len(l.s) && isVarChar(l.s[l.i]) {
				l.i++
			}
			l.text = l.s[start:l.i]
			l.tok = keyword(l.text)
			l.eatWS()
			return l.tok
		}
		l.tok = terror
		l.tokenErr("lexing error")
		return terror
	}
}

func keyword(s string) token {
	switch s {
	case "build":
		return tbuild
	case "pool":
		return tpool
	case "rule":
		return trule
	case "default":
		return tdefault
	case "include":
		return tinclude
	case "subninja":
		return tsubninja
	default:
		return tident
	}
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

func (l *lexer) peek(want token) (bool, error) {
	t := l.readToken()
	if t == terror {
		return false, l.err
	}
	if t == want {
		return true, nil
	}
	l.unread()
	return false, nil
}

func (l *lexer) expect(want token) error {
	t := l.readToken()
	if t == terror {
		return l.err
	}
	if t != want {
		return l.tokenErr("expected " + want.String() + ", got " + t.String())
	}
	return nil
}

func (l *lexer) readIdent(what string) (string, error) {
	l.tokStart = l.i
	l.tokLine = l.line
	if l.i >= len(l.s) || !isVarChar(l.s[l.i]) {
		return "", l.tokenErr("expected " + what)
	}
	start := l.i
	for l.i < len(l.s) && isVarChar(l.s[l.i]) {
		l.i++
	}
	name := l.s[start:l.i]
	l.eatWS()
	return name, nil
}

func (l *lexer) parseVariable(name string) (*Variable, error) {
	if err := l.expect(tequals); err != nil {
		return nil, err
	}
	val, err := l.readEval(false)
	if err != nil {
		return nil, err
	}
	return &Variable{Name: name, Value: val}, nil
}

func (l *lexer) parseBinding() (Binding, error) {
	name, err := l.readIdent("variable name")
	if err != nil {
		return Binding{}, err
	}
	if err := l.expect(tequals); err != nil {
		return Binding{}, err
	}
	val, err := l.readEval(false)
	if err != nil {
		return Binding{}, err
	}
	if err := l.expect(tnewline); err != nil {
		return Binding{}, err
	}
	return Binding{Name: name, Value: val}, nil
}

func (l *lexer) parseBindings() ([]Binding, error) {
	var out []Binding
	for {
		ok, err := l.peek(tindent)
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		b, err := l.parseBinding()
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

func (l *lexer) parseRule() (*Rule, error) {
	name, err := l.readIdent("rule name")
	if err != nil {
		return nil, err
	}
	if err := l.expect(tnewline); err != nil {
		return nil, err
	}
	bs, err := l.parseBindings()
	if err != nil {
		return nil, err
	}
	return &Rule{Name: name, Bindings: bs}, nil
}

func (l *lexer) parsePool() (*Pool, error) {
	name, err := l.readIdent("pool name")
	if err != nil {
		return nil, err
	}
	if err := l.expect(tnewline); err != nil {
		return nil, err
	}
	bs, err := l.parseBindings()
	if err != nil {
		return nil, err
	}
	return &Pool{Name: name, Bindings: bs}, nil
}

func (l *lexer) parseEdge() (*Edge, error) {
	outs, err := l.readPaths()
	if err != nil {
		return nil, err
	}
	var implOut [][]Piece
	ok, err := l.peek(tpipe)
	if err != nil {
		return nil, err
	}
	if ok {
		implOut, err = l.readPaths()
		if err != nil {
			return nil, err
		}
	}
	if len(outs) == 0 && len(implOut) == 0 {
		return nil, l.tokenErr("expected path")
	}
	if err := l.expect(tcolon); err != nil {
		return nil, err
	}
	rule, err := l.readIdent("build command name")
	if err != nil {
		return nil, err
	}
	ins, err := l.readPaths()
	if err != nil {
		return nil, err
	}
	var impl, order, valid [][]Piece
	ok, err = l.peek(tpipe)
	if err != nil {
		return nil, err
	}
	if ok {
		impl, err = l.readPaths()
		if err != nil {
			return nil, err
		}
	}
	ok, err = l.peek(tpipe2)
	if err != nil {
		return nil, err
	}
	if ok {
		order, err = l.readPaths()
		if err != nil {
			return nil, err
		}
	}
	ok, err = l.peek(tpipeat)
	if err != nil {
		return nil, err
	}
	if ok {
		valid, err = l.readPaths()
		if err != nil {
			return nil, err
		}
	}
	if err := l.expect(tnewline); err != nil {
		return nil, err
	}
	bs, err := l.parseBindings()
	if err != nil {
		return nil, err
	}
	return &Edge{
		Outputs:         outs,
		ImplicitOutputs: implOut,
		Rule:            rule,
		Inputs:          ins,
		ImplicitDeps:    impl,
		OrderOnlyDeps:   order,
		Validations:     valid,
		Bindings:        bs,
	}, nil
}

func (l *lexer) parseDefault() (*Default, error) {
	targets, err := l.readPaths()
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return nil, l.tokenErr("expected target name")
	}
	if err := l.expect(tnewline); err != nil {
		return nil, err
	}
	return &Default{Targets: targets}, nil
}

func (l *lexer) parseInclude(sub bool) (*Include, error) {
	path, err := l.readEval(true)
	if err != nil {
		return nil, err
	}
	if len(path) == 0 {
		return nil, l.tokenErr("expected path")
	}
	if err := l.expect(tnewline); err != nil {
		return nil, err
	}
	return &Include{Path: path, Subninja: sub}, nil
}

func (l *lexer) readPaths() ([][]Piece, error) {
	var out [][]Piece
	for {
		p, err := l.readEval(true)
		if err != nil {
			return nil, err
		}
		if len(p) == 0 {
			break
		}
		out = append(out, p)
	}
	return out, nil
}

// readEval reads a value (path false) or one path (path true).
// A path stops at an unescaped space, colon, pipe, or newline.
// The stopping newline is left for the next token. End of input is an error,
// matching ninja's unexpected EOF on a missing final newline.
func (l *lexer) readEval(path bool) ([]Piece, error) {
	var ps []Piece
	for {
		if l.i >= len(l.s) {
			return nil, l.at(l.line, "unexpected EOF")
		}
		c := l.s[l.i]
		if c == 0 {
			return nil, l.at(l.line, "unexpected EOF")
		}
		if c == '\n' || (c == '\r' && l.i+1 < len(l.s) && l.s[l.i+1] == '\n') {
			break
		}
		if c == '\r' {
			return nil, l.at(l.line, "lexing error")
		}
		if path && (c == ' ' || c == ':' || c == '|') {
			break
		}
		if c == '$' {
			if err := l.readDollar(&ps); err != nil {
				return nil, err
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
	if path {
		l.eatWS()
	}
	if len(ps) == 0 {
		return nil, nil
	}
	return ps, nil
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
	return keyword(s) != tident
}
