package graphviz

import "strings"

type printer struct {
	b        strings.Builder
	level    int
	directed bool
}

func (p *printer) file(f *File) {
	for _, g := range f.Graphs {
		p.directed = g.Directed
		p.graph(g)
	}
}

func (p *printer) graph(g *Graph) {
	if g.Strict {
		p.raw("strict ")
	}
	if g.Directed {
		p.raw("digraph")
	} else {
		p.raw("graph")
	}
	if present(g.Name) {
		p.raw(" ")
		p.atom(g.Name)
	}
	p.raw(" {\n")
	p.level++
	for _, stmt := range g.Stmts {
		p.stmt(stmt)
	}
	p.level--
	p.raw("}\n")
}

func (p *printer) stmt(stmt *Stmt) {
	p.indent()
	switch stmt.Kind {
	case StmtAssign:
		p.atom(stmt.Assign.Name)
		p.raw("=")
		p.atom(stmt.Assign.Value)
	case StmtAttr:
		p.raw(stmt.Attr.Kind.String())
		p.raw(" ")
		p.attrs(stmt.Attr.Attrs)
	case StmtNode:
		p.ids(stmt.Node.IDs)
		if len(stmt.Node.Attrs) > 0 {
			p.raw(" ")
			p.attrs(stmt.Node.Attrs)
		}
	case StmtEdge:
		p.edge(stmt.Edge)
	case StmtSubgraph:
		p.subgraph(stmt.Subgraph)
	}
	p.raw(";\n")
}

func (p *printer) edge(edge *Edge) {
	op := " -- "
	if p.directed {
		op = " -> "
	}
	for i, end := range edge.Ends {
		if i > 0 {
			p.raw(op)
		}
		p.end(end)
	}
	if len(edge.Attrs) > 0 {
		p.raw(" ")
		p.attrs(edge.Attrs)
	}
}

func (p *printer) end(end *End) {
	if end.Sub != nil {
		p.subgraph(end.Sub)
		return
	}
	p.ids(end.IDs)
}

func (p *printer) ids(ids []*NodeID) {
	for i, id := range ids {
		if i > 0 {
			p.raw(", ")
		}
		p.nodeID(id)
	}
}

func (p *printer) nodeID(id *NodeID) {
	p.atom(id.Name)
	if id.HasPort {
		p.raw(":")
		p.atom(id.Port)
	}
	if id.HasCompass {
		p.raw(":")
		p.atom(id.Compass)
	}
}

func (p *printer) subgraph(g *Graph) {
	if g.Bare {
		p.raw("{\n")
	} else {
		p.raw("subgraph")
		if present(g.Name) {
			p.raw(" ")
			p.atom(g.Name)
		}
		p.raw(" {\n")
	}
	p.level++
	for _, stmt := range g.Stmts {
		p.stmt(stmt)
	}
	p.level--
	p.indent()
	p.raw("}")
}

func (p *printer) attrs(attrs []*Attr) {
	p.raw("[")
	for i, attr := range attrs {
		if i > 0 {
			p.raw(", ")
		}
		p.atom(attr.Name)
		p.raw("=")
		p.atom(attr.Value)
	}
	p.raw("]")
}

func (p *printer) atom(atom Atom) {
	if atom.HTML {
		p.raw("<")
		p.raw(atom.Text)
		p.raw(">")
		return
	}
	p.raw(canonText(atom.Text))
}

func (p *printer) indent() {
	for i := 0; i < p.level; i++ {
		p.raw("\t")
	}
}

func (p *printer) raw(s string) {
	p.b.WriteString(s)
}

func present(atom Atom) bool {
	return atom.Text != "" || atom.HTML
}

// canonText prints one decoded string the way Graphviz writes an id.
// A quote is escaped. A backslash stays, so "\n" and "\\" round-trip
// through the Graphviz lexer.
func canonText(arg string) string {
	if arg == "" {
		return `""`
	}
	var b strings.Builder
	b.Grow(len(arg) + 2)
	b.WriteByte('"')
	needsQuotes := false
	partOfEscape := false
	maybeNum := isDigit(arg[0]) || arg[0] == '.' || arg[0] == '-'
	dots := 0
	for i := 0; i < len(arg); i++ {
		uc := arg[i]
		if uc == '"' && !partOfEscape {
			b.WriteByte('\\')
			needsQuotes = true
		} else if !partOfEscape && isEscape(arg[i:]) {
			needsQuotes = true
			partOfEscape = true
		} else if maybeNum {
			switch {
			case uc == '-':
				if i > 0 {
					maybeNum = false
					needsQuotes = true
				}
			case uc == '.':
				dots++
				if dots > 1 {
					maybeNum = false
					needsQuotes = true
				}
			case !isDigit(uc):
				maybeNum = false
				needsQuotes = true
			}
			partOfEscape = false
		} else if !isIDByte(uc) {
			needsQuotes = true
			partOfEscape = false
		} else {
			partOfEscape = false
		}
		b.WriteByte(uc)
	}
	b.WriteByte('"')
	if needsQuotes || (len(arg) == 1 && (arg[0] == '.' || arg[0] == '-')) {
		return b.String()
	}
	if keywordFold(arg) {
		return b.String()
	}
	return arg
}

func isEscape(s string) bool {
	if len(s) < 2 || s[0] != '\\' {
		return false
	}
	switch s[1] {
	case 'E', 'G', 'H', 'L', 'N', 'T', 'l', 'n', 'r', '\\', '"':
		return true
	default:
		return false
	}
}

func isIDByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || isDigit(c) || c == '_' || c >= 0x80
}

func keywordFold(s string) bool {
	switch asciiLower(s) {
	case "node", "edge", "graph", "digraph", "strict", "subgraph":
		return true
	default:
		return false
	}
}

func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}
