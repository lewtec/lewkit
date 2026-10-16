package graphviz

import (
	"fmt"
)

func parseSource(src []byte) (*File, error) {
	p := &parser{}
	p.lx.init(src)
	p.next()
	file := New()
	for p.tok.kind != tokEOF {
		g, err := p.graph()
		if err != nil {
			return nil, err
		}
		file.Graphs = append(file.Graphs, g)
	}
	if p.err != nil {
		return nil, p.err
	}
	return file, nil
}

type tokenKind uint8

const (
	tokEOF tokenKind = iota
	tokIdent
	tokString
	tokKeyword
	tokEdge
	tokPunct
)

type token struct {
	kind tokenKind
	text string
	html bool
	line int
}

type lexer struct {
	src  []byte
	i    int
	line int
}

func (l *lexer) init(src []byte) {
	l.src = src
	l.line = 1
	if len(src) >= 3 && src[0] == 0xEF && src[1] == 0xBB && src[2] == 0xBF {
		l.i = 3
	}
}

func (l *lexer) next() (token, error) {
	if err := l.skip(); err != nil {
		return token{kind: tokEOF, line: l.line}, err
	}
	if l.i >= len(l.src) {
		return token{kind: tokEOF, line: l.line}, nil
	}
	line := l.line
	if l.has("--") || l.has("->") {
		text := string(l.src[l.i : l.i+2])
		l.i += 2
		return token{kind: tokEdge, text: text, line: line}, nil
	}
	c := l.src[l.i]
	switch c {
	case '{', '}', '[', ']', ';', ',', '=', ':', '+':
		l.i++
		return token{kind: tokPunct, text: string(c), line: line}, nil
	case '"':
		return l.string(line)
	case '<':
		return l.html(line)
	}
	if isLetter(c) {
		return l.name(line), nil
	}
	if c == '-' || c == '.' || isDigit(c) {
		return l.number(line), nil
	}
	l.i++
	return token{kind: tokPunct, text: string(c), line: line}, nil
}

func (l *lexer) has(s string) bool {
	return l.i+len(s) <= len(l.src) && string(l.src[l.i:l.i+len(s)]) == s
}

func (l *lexer) skip() error {
	for l.i < len(l.src) {
		c := l.src[l.i]
		switch c {
		case ' ', '\t', '\r':
			l.i++
			continue
		case '\n':
			l.line++
			l.i++
			continue
		case '#':
			l.skipLine()
			continue
		case '/':
			if l.i+1 < len(l.src) && l.src[l.i+1] == '/' {
				l.i += 2
				l.skipLine()
				continue
			}
			if l.i+1 < len(l.src) && l.src[l.i+1] == '*' {
				l.i += 2
				if err := l.blockComment(); err != nil {
					return err
				}
				continue
			}
		}
		break
	}
	return nil
}

func (l *lexer) skipLine() {
	for l.i < len(l.src) && l.src[l.i] != '\n' {
		l.i++
	}
}

func (l *lexer) blockComment() error {
	for l.i < len(l.src) {
		c := l.src[l.i]
		l.i++
		if c == '\n' {
			l.line++
			continue
		}
		if c == '*' && l.i < len(l.src) && l.src[l.i] == '/' {
			l.i++
			return nil
		}
	}
	return fmt.Errorf("%w: line %d: unclosed comment", ErrSyntax, l.line)
}

func (l *lexer) string(line int) (token, error) {
	l.i++ // opening quote
	var b []byte
	for l.i < len(l.src) {
		c := l.src[l.i]
		if c == '"' {
			l.i++
			return token{kind: tokString, text: string(b), line: line}, nil
		}
		if c == '\\' && l.i+1 < len(l.src) {
			next := l.src[l.i+1]
			switch next {
			case '"':
				b = append(b, '"')
				l.i += 2
				continue
			case '\\':
				b = append(b, '\\', '\\')
				l.i += 2
				continue
			case '\n':
				l.line++
				l.i += 2
				continue
			}
		}
		if c == '\n' {
			l.line++
		}
		b = append(b, c)
		l.i++
	}
	return token{kind: tokEOF, line: line}, fmt.Errorf("%w: line %d: unclosed string", ErrSyntax, line)
}

func (l *lexer) html(line int) (token, error) {
	l.i++ // opening <
	nest := 1
	var b []byte
	for l.i < len(l.src) {
		c := l.src[l.i]
		if c == '>' {
			nest--
			l.i++
			if nest == 0 {
				return token{kind: tokString, text: string(b), html: true, line: line}, nil
			}
			b = append(b, c)
			continue
		}
		if c == '<' {
			nest++
		}
		if c == '\n' {
			l.line++
		}
		b = append(b, c)
		l.i++
	}
	return token{kind: tokEOF, line: line}, fmt.Errorf("%w: line %d: unclosed HTML string", ErrSyntax, line)
}

func (l *lexer) name(line int) token {
	start := l.i
	for l.i < len(l.src) && (isLetter(l.src[l.i]) || isDigit(l.src[l.i])) {
		l.i++
	}
	text := string(l.src[start:l.i])
	if isKeyword(text) {
		return token{kind: tokKeyword, text: text, line: line}
	}
	return token{kind: tokIdent, text: text, line: line}
}

func (l *lexer) number(line int) token {
	start := l.i
	rest := l.src[l.i:]
	if len(rest) == 0 {
		return token{kind: tokEOF, line: line}
	}
	if rest[0] == '-' {
		if len(rest) > 1 && (isDigit(rest[1]) || (rest[1] == '.' && len(rest) > 2 && isDigit(rest[2]))) {
			l.i++
		} else {
			l.i++
			return token{kind: tokPunct, text: "-", line: line}
		}
	} else if rest[0] == '.' {
		if len(rest) > 1 && isDigit(rest[1]) {
			l.i++
		} else {
			l.i++
			return token{kind: tokPunct, text: ".", line: line}
		}
	}
	if l.i < len(l.src) && isDigit(l.src[l.i]) {
		for l.i < len(l.src) && isDigit(l.src[l.i]) {
			l.i++
		}
		if l.i < len(l.src) && l.src[l.i] == '.' {
			l.i++
			for l.i < len(l.src) && isDigit(l.src[l.i]) {
				l.i++
			}
		}
	} else if l.i < len(l.src) && l.src[l.i] == '.' {
		l.i++
		for l.i < len(l.src) && isDigit(l.src[l.i]) {
			l.i++
		}
	}
	if l.i == start {
		c := l.src[l.i]
		l.i++
		return token{kind: tokPunct, text: string(c), line: line}
	}
	return token{kind: tokIdent, text: string(l.src[start:l.i]), line: line}
}

func isKeyword(s string) bool {
	switch s {
	case "node", "edge", "graph", "digraph", "strict", "subgraph":
		return true
	default:
		return false
	}
}

func isLetter(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || c >= 0x80
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

type parser struct {
	lx  lexer
	tok token
	err error
}

func (p *parser) next() {
	if p.err != nil {
		p.tok = token{kind: tokEOF, line: p.lx.line}
		return
	}
	tok, err := p.lx.next()
	p.tok = tok
	p.err = err
}

func (p *parser) syntax(msg string) error {
	if p.err != nil {
		return p.err
	}
	found := p.tok.text
	if p.tok.kind == tokEOF {
		found = "end of file"
	}
	return fmt.Errorf("%w: line %d: %s, found %q", ErrSyntax, p.tok.line, msg, found)
}

func (p *parser) expect(text string) error {
	if p.err != nil {
		return p.err
	}
	if p.tok.text != text {
		return p.syntax("expected " + text)
	}
	p.next()
	return p.err
}

func (p *parser) isAtom() bool {
	return p.tok.kind == tokIdent || p.tok.kind == tokString
}

func (p *parser) graph() (*Graph, error) {
	if p.err != nil {
		return nil, p.err
	}
	g := &Graph{}
	if p.tok.kind == tokKeyword && p.tok.text == "strict" {
		g.Strict = true
		p.next()
	}
	if p.err != nil {
		return nil, p.err
	}
	if p.tok.kind != tokKeyword {
		return nil, p.syntax("expected graph or digraph")
	}
	switch p.tok.text {
	case "graph":
	case "digraph":
		g.Directed = true
	default:
		return nil, p.syntax("expected graph or digraph")
	}
	p.next()
	if p.isAtom() {
		name, err := p.atom()
		if err != nil {
			return nil, err
		}
		g.Name = name
	}
	if err := p.expect("{"); err != nil {
		return nil, err
	}
	if err := p.stmtList(g, g.Directed); err != nil {
		return nil, err
	}
	if err := p.expect("}"); err != nil {
		return nil, err
	}
	return g, nil
}

func (p *parser) stmtList(g *Graph, directed bool) error {
	for p.tok.text != "}" && p.tok.kind != tokEOF {
		if err := p.stmt(g, directed); err != nil {
			return err
		}
		if p.tok.text == ";" {
			p.next()
			if p.err != nil {
				return p.err
			}
		}
	}
	if p.tok.kind == tokEOF {
		return p.syntax("unclosed '{'")
	}
	return nil
}

func (p *parser) stmt(g *Graph, directed bool) error {
	if p.err != nil {
		return p.err
	}
	if p.tok.kind == tokKeyword {
		switch p.tok.text {
		case "graph", "node", "edge":
			return p.attrStmt(g)
		case "subgraph":
			sub, err := p.subgraph(directed)
			if err != nil {
				return err
			}
			return p.finish(g, directed, &End{Sub: sub})
		default:
			return p.syntax("expected a statement")
		}
	}
	if p.tok.text == "{" {
		sub, err := p.subgraph(directed)
		if err != nil {
			return err
		}
		return p.finish(g, directed, &End{Sub: sub})
	}
	if !p.isAtom() {
		return p.syntax("expected a statement")
	}
	name, err := p.atom()
	if err != nil {
		return err
	}
	if p.tok.text == "=" {
		p.next()
		value, err := p.atom()
		if err != nil {
			return err
		}
		g.Stmts = append(g.Stmts, &Stmt{
			Kind:   StmtAssign,
			Assign: &Assign{Name: name, Value: value},
		})
		return nil
	}
	id, err := p.nodeID(name)
	if err != nil {
		return err
	}
	ids, err := p.nodeList(id)
	if err != nil {
		return err
	}
	return p.finish(g, directed, &End{IDs: ids})
}

func (p *parser) finish(g *Graph, directed bool, first *End) error {
	ends := []*End{first}
	edge := false
	for p.tok.kind == tokEdge {
		edge = true
		if directed && p.tok.text != "->" {
			return p.syntax("undirected edge in a digraph")
		}
		if !directed && p.tok.text != "--" {
			return p.syntax("directed edge in a graph")
		}
		p.next()
		end, err := p.simple(directed)
		if err != nil {
			return err
		}
		ends = append(ends, end)
	}
	attrs, err := p.attrLists(false)
	if err != nil {
		return err
	}
	if edge {
		g.Stmts = append(g.Stmts, &Stmt{
			Kind: StmtEdge,
			Edge: &Edge{Ends: ends, Attrs: attrs},
		})
		return nil
	}
	if first.Sub != nil {
		if len(attrs) > 0 {
			return p.syntax("attributes on a subgraph")
		}
		g.Stmts = append(g.Stmts, &Stmt{Kind: StmtSubgraph, Subgraph: first.Sub})
		return nil
	}
	g.Stmts = append(g.Stmts, &Stmt{
		Kind: StmtNode,
		Node: &Node{IDs: first.IDs, Attrs: attrs},
	})
	return nil
}

func (p *parser) simple(directed bool) (*End, error) {
	if p.err != nil {
		return nil, p.err
	}
	if p.tok.text == "{" || (p.tok.kind == tokKeyword && p.tok.text == "subgraph") {
		sub, err := p.subgraph(directed)
		if err != nil {
			return nil, err
		}
		return &End{Sub: sub}, nil
	}
	if !p.isAtom() {
		return nil, p.syntax("expected a node")
	}
	name, err := p.atom()
	if err != nil {
		return nil, err
	}
	id, err := p.nodeID(name)
	if err != nil {
		return nil, err
	}
	ids, err := p.nodeList(id)
	if err != nil {
		return nil, err
	}
	return &End{IDs: ids}, nil
}

func (p *parser) nodeList(first *NodeID) ([]*NodeID, error) {
	ids := []*NodeID{first}
	for p.tok.text == "," {
		p.next()
		if p.err != nil {
			return nil, p.err
		}
		if !p.isAtom() {
			return nil, p.syntax("expected a node")
		}
		name, err := p.atom()
		if err != nil {
			return nil, err
		}
		id, err := p.nodeID(name)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (p *parser) nodeID(name Atom) (*NodeID, error) {
	id := &NodeID{Name: name}
	if p.tok.text != ":" {
		return id, p.err
	}
	p.next()
	port, err := p.atom()
	if err != nil {
		return nil, err
	}
	id.HasPort = true
	id.Port = port
	if p.tok.text != ":" {
		return id, p.err
	}
	p.next()
	compass, err := p.atom()
	if err != nil {
		return nil, err
	}
	id.HasCompass = true
	id.Compass = compass
	return id, nil
}

func (p *parser) subgraph(directed bool) (*Graph, error) {
	if p.err != nil {
		return nil, p.err
	}
	g := &Graph{Sub: true}
	if p.tok.kind == tokKeyword && p.tok.text == "subgraph" {
		p.next()
		if p.isAtom() {
			name, err := p.atom()
			if err != nil {
				return nil, err
			}
			g.Name = name
		}
	} else if p.tok.text == "{" {
		g.Bare = true
	} else {
		return nil, p.syntax("expected a subgraph")
	}
	if err := p.expect("{"); err != nil {
		return nil, err
	}
	if err := p.stmtList(g, directed); err != nil {
		return nil, err
	}
	if err := p.expect("}"); err != nil {
		return nil, err
	}
	return g, nil
}

func (p *parser) attrStmt(g *Graph) error {
	var kind AttrKind
	switch p.tok.text {
	case "graph":
		kind = AttrGraph
	case "node":
		kind = AttrNode
	case "edge":
		kind = AttrEdge
	default:
		return p.syntax("expected an attribute statement")
	}
	p.next()
	if p.isAtom() {
		if _, err := p.atom(); err != nil {
			return err
		}
		if err := p.expect("="); err != nil {
			return err
		}
	}
	attrs, err := p.attrLists(true)
	if err != nil {
		return err
	}
	g.Stmts = append(g.Stmts, &Stmt{
		Kind: StmtAttr,
		Attr: &AttrStmt{Kind: kind, Attrs: attrs},
	})
	return nil
}

func (p *parser) attrLists(required bool) ([]*Attr, error) {
	if p.err != nil {
		return nil, p.err
	}
	if p.tok.text != "[" {
		if required {
			return nil, p.syntax("expected '['")
		}
		return nil, nil
	}
	var attrs []*Attr
	for p.tok.text == "[" {
		p.next()
		for p.tok.text != "]" && p.tok.kind != tokEOF {
			if !p.isAtom() {
				return nil, p.syntax("expected an attribute")
			}
			name, err := p.atom()
			if err != nil {
				return nil, err
			}
			if err := p.expect("="); err != nil {
				return nil, err
			}
			value, err := p.atom()
			if err != nil {
				return nil, err
			}
			attrs = append(attrs, &Attr{Name: name, Value: value})
			if p.tok.text == "," || p.tok.text == ";" {
				p.next()
				if p.err != nil {
					return nil, p.err
				}
			}
		}
		if err := p.expect("]"); err != nil {
			return nil, err
		}
	}
	return attrs, nil
}

func (p *parser) atom() (Atom, error) {
	if p.err != nil {
		return Atom{}, p.err
	}
	if !p.isAtom() {
		return Atom{}, p.syntax("expected an id")
	}
	atom := Atom{Text: p.tok.text, HTML: p.tok.html}
	quoted := p.tok.kind == tokString
	p.next()
	if p.err != nil {
		return Atom{}, p.err
	}
	if !quoted {
		return atom, nil
	}
	for p.tok.text == "+" {
		p.next()
		if p.err != nil {
			return Atom{}, p.err
		}
		if p.tok.kind != tokString {
			return Atom{}, p.syntax("expected a quoted string")
		}
		atom.Text += p.tok.text
		atom.HTML = false
		p.next()
		if p.err != nil {
			return Atom{}, p.err
		}
	}
	return atom, nil
}
