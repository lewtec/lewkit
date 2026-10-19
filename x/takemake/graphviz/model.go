package graphviz

import (
	"fmt"
	"io"
)

// File is one or more Graphviz graphs.
type File struct {
	Graphs []*Graph
}

// Atom is one identifier, number, or string.
// HTML selects an HTML string. Text is the decoded value.
type Atom struct {
	Text string
	HTML bool
}

// Graph is a graph, a digraph, or a subgraph.
// Strict prints the strict keyword on a top-level graph.
// Directed selects ->. A subgraph leaves Directed unset and uses the
// enclosing graph's operator.
// Sub marks a subgraph. Bare is the `{ ... }` form, with no subgraph keyword.
type Graph struct {
	Name     Atom
	Strict   bool
	Directed bool
	Sub      bool
	Bare     bool
	Stmts    []*Stmt
}

// Stmt is one statement. The field that matches Kind is set.
type Stmt struct {
	Kind     StmtKind
	Node     *Node
	Edge     *Edge
	Attr     *AttrStmt
	Assign   *Assign
	Subgraph *Graph
}

// Node is one node statement.
// IDs is the comma-separated list. Attrs is the trailing attribute list.
type Node struct {
	IDs   []*NodeID
	Attrs []*Attr
}

// NodeID is a node name and an optional port and compass.
// HasPort is set when the source wrote a colon, including an empty port.
type NodeID struct {
	Name       Atom
	HasPort    bool
	Port       Atom
	HasCompass bool
	Compass    Atom
}

// Edge is one edge statement.
// Ends is the sequence of endpoints, with at least two.
// Directed is not stored: the enclosing graph selects -> or --.
type Edge struct {
	Ends  []*End
	Attrs []*Attr
}

// End is one side of an edge operator.
// IDs is a comma-separated node list. Sub is a subgraph endpoint.
type End struct {
	IDs []*NodeID
	Sub *Graph
}

// AttrStmt is a default attribute statement: graph, node, or edge.
type AttrStmt struct {
	Kind  AttrKind
	Attrs []*Attr
}

// Assign is a graph attribute written as name=value.
type Assign struct {
	Name  Atom
	Value Atom
}

// Attr is one name=value pair.
// Value.HTML is an HTML string.
type Attr struct {
	Name  Atom
	Value Atom
}

// New returns an empty file.
func New() *File {
	return &File{}
}

// Take parses Graphviz source from r into a [File].
func Take(r io.Reader) (*File, error) {
	if r == nil {
		return nil, ErrRead
	}
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRead, err)
	}
	return parseSource(buf)
}

// Make prints the file as Graphviz source.
func (f *File) Make(w io.Writer) error {
	if f == nil {
		return ErrFile
	}
	if w == nil {
		return ErrWrite
	}
	if err := f.validate(); err != nil {
		return err
	}
	var p printer
	p.file(f)
	if _, err := io.WriteString(w, p.b.String()); err != nil {
		return fmt.Errorf("%w: %w", ErrWrite, err)
	}
	return nil
}

// Graph appends an undirected graph and returns it.
func (f *File) Graph(name string) *Graph {
	if f == nil {
		return nil
	}
	g := &Graph{Name: Atom{Text: name}}
	f.Graphs = append(f.Graphs, g)
	return g
}

// Digraph appends a directed graph and returns it.
func (f *File) Digraph(name string) *Graph {
	g := f.Graph(name)
	if g == nil {
		return nil
	}
	g.Directed = true
	return g
}

// SetStrict sets the strict keyword and returns the graph.
func (g *Graph) SetStrict(strict bool) *Graph {
	if g == nil {
		return nil
	}
	g.Strict = strict
	return g
}

// SetName sets the graph name and returns the graph.
func (g *Graph) SetName(name string) *Graph {
	if g == nil {
		return nil
	}
	g.Name = Atom{Text: name}
	return g
}

// Assign stores a graph attribute as name=value and returns the graph.
func (g *Graph) Assign(name, value string) *Graph {
	return g.assign(Atom{Text: name}, Atom{Text: value})
}

// AssignHTML stores a graph attribute whose value is an HTML string.
func (g *Graph) AssignHTML(name, html string) *Graph {
	return g.assign(Atom{Text: name}, Atom{Text: html, HTML: true})
}

func (g *Graph) assign(name, value Atom) *Graph {
	if g == nil {
		return nil
	}
	g.Stmts = append(g.Stmts, &Stmt{
		Kind:   StmtAssign,
		Assign: &Assign{Name: name, Value: value},
	})
	return g
}

// GraphAttr appends a graph default-attribute statement and returns it.
func (g *Graph) GraphAttr(name, value string) *AttrStmt {
	return g.defaults(AttrGraph, name, value, false)
}

// NodeAttr appends a node default-attribute statement and returns it.
func (g *Graph) NodeAttr(name, value string) *AttrStmt {
	return g.defaults(AttrNode, name, value, false)
}

// EdgeAttr appends an edge default-attribute statement and returns it.
func (g *Graph) EdgeAttr(name, value string) *AttrStmt {
	return g.defaults(AttrEdge, name, value, false)
}

func (g *Graph) defaults(kind AttrKind, name, value string, html bool) *AttrStmt {
	if g == nil {
		return nil
	}
	stmt := &AttrStmt{Kind: kind}
	stmt.attr(name, value, html)
	g.Stmts = append(g.Stmts, &Stmt{Kind: StmtAttr, Attr: stmt})
	return stmt
}

// Attr appends a name=value pair and returns the statement.
func (s *AttrStmt) Attr(name, value string) *AttrStmt {
	return s.attr(name, value, false)
}

// HTML appends a name=value pair whose value is an HTML string.
func (s *AttrStmt) HTML(name, html string) *AttrStmt {
	return s.attr(name, html, true)
}

func (s *AttrStmt) attr(name, value string, html bool) *AttrStmt {
	if s == nil {
		return nil
	}
	s.Attrs = append(s.Attrs, &Attr{
		Name:  Atom{Text: name},
		Value: Atom{Text: value, HTML: html},
	})
	return s
}

// Node appends a node statement and returns it.
func (g *Graph) Node(name string) *Node {
	if g == nil {
		return nil
	}
	node := &Node{IDs: []*NodeID{{Name: Atom{Text: name}}}}
	g.Stmts = append(g.Stmts, &Stmt{Kind: StmtNode, Node: node})
	return node
}

// Also appends another node id to this statement and returns the node.
func (n *Node) Also(name string) *Node {
	if n == nil {
		return nil
	}
	n.IDs = append(n.IDs, &NodeID{Name: Atom{Text: name}})
	return n
}

// Attr appends a name=value pair and returns the node.
func (n *Node) Attr(name, value string) *Node {
	return n.attr(name, value, false)
}

// HTML appends a name=value pair whose value is an HTML string.
func (n *Node) HTML(name, html string) *Node {
	return n.attr(name, html, true)
}

func (n *Node) attr(name, value string, html bool) *Node {
	if n == nil {
		return nil
	}
	n.Attrs = append(n.Attrs, &Attr{
		Name:  Atom{Text: name},
		Value: Atom{Text: value, HTML: html},
	})
	return n
}

// Port sets the port on the last id and returns the node.
func (n *Node) Port(port string) *Node {
	if n == nil || len(n.IDs) == 0 {
		return n
	}
	id := n.IDs[len(n.IDs)-1]
	id.HasPort = true
	id.Port = Atom{Text: port}
	return n
}

// Compass sets the compass on the last id and returns the node.
func (n *Node) Compass(compass string) *Node {
	if n == nil || len(n.IDs) == 0 {
		return n
	}
	id := n.IDs[len(n.IDs)-1]
	id.HasCompass = true
	id.Compass = Atom{Text: compass}
	return n
}

// Edge appends an edge from from to to and returns it.
func (g *Graph) Edge(from, to string) *Edge {
	if g == nil {
		return nil
	}
	edge := &Edge{Ends: []*End{
		{IDs: []*NodeID{{Name: Atom{Text: from}}}},
		{IDs: []*NodeID{{Name: Atom{Text: to}}}},
	}}
	g.Stmts = append(g.Stmts, &Stmt{Kind: StmtEdge, Edge: edge})
	return edge
}

// To appends an endpoint and returns the edge.
func (e *Edge) To(name string) *Edge {
	if e == nil {
		return nil
	}
	e.Ends = append(e.Ends, &End{IDs: []*NodeID{{Name: Atom{Text: name}}}})
	return e
}

// Attr appends a name=value pair and returns the edge.
func (e *Edge) Attr(name, value string) *Edge {
	return e.attr(name, value, false)
}

// HTML appends a name=value pair whose value is an HTML string.
func (e *Edge) HTML(name, html string) *Edge {
	return e.attr(name, html, true)
}

func (e *Edge) attr(name, value string, html bool) *Edge {
	if e == nil {
		return nil
	}
	e.Attrs = append(e.Attrs, &Attr{
		Name:  Atom{Text: name},
		Value: Atom{Text: value, HTML: html},
	})
	return e
}

// FromPort sets the port on the first endpoint and returns the edge.
func (e *Edge) FromPort(port string) *Edge {
	return e.port(true, port)
}

// ToPort sets the port on the last endpoint and returns the edge.
func (e *Edge) ToPort(port string) *Edge {
	return e.port(false, port)
}

func (e *Edge) port(from bool, port string) *Edge {
	if e == nil || len(e.Ends) == 0 {
		return e
	}
	end := e.Ends[0]
	if !from {
		end = e.Ends[len(e.Ends)-1]
	}
	if len(end.IDs) == 0 {
		return e
	}
	id := end.IDs[0]
	if !from {
		id = end.IDs[len(end.IDs)-1]
	}
	id.HasPort = true
	id.Port = Atom{Text: port}
	return e
}

// FromCompass sets the compass on the first endpoint and returns the edge.
func (e *Edge) FromCompass(compass string) *Edge {
	return e.compass(true, compass)
}

// ToCompass sets the compass on the last endpoint and returns the edge.
func (e *Edge) ToCompass(compass string) *Edge {
	return e.compass(false, compass)
}

func (e *Edge) compass(from bool, compass string) *Edge {
	if e == nil || len(e.Ends) == 0 {
		return e
	}
	end := e.Ends[0]
	if !from {
		end = e.Ends[len(e.Ends)-1]
	}
	if len(end.IDs) == 0 {
		return e
	}
	id := end.IDs[0]
	if !from {
		id = end.IDs[len(end.IDs)-1]
	}
	id.HasCompass = true
	id.Compass = Atom{Text: compass}
	return e
}

// Subgraph appends a subgraph statement and returns it.
// An empty name prints `subgraph { ... }`.
func (g *Graph) Subgraph(name string) *Graph {
	if g == nil {
		return nil
	}
	sub := &Graph{Sub: true, Name: Atom{Text: name}}
	g.Stmts = append(g.Stmts, &Stmt{Kind: StmtSubgraph, Subgraph: sub})
	return sub
}

// Group appends an anonymous `{ ... }` subgraph and returns it.
func (g *Graph) Group() *Graph {
	if g == nil {
		return nil
	}
	sub := &Graph{Sub: true, Bare: true}
	g.Stmts = append(g.Stmts, &Stmt{Kind: StmtSubgraph, Subgraph: sub})
	return sub
}

func (f *File) validate() error {
	if f == nil {
		return ErrFile
	}
	for _, g := range f.Graphs {
		if err := validateGraph(g, false); err != nil {
			return err
		}
	}
	return nil
}

func validateGraph(g *Graph, sub bool) error {
	if g == nil {
		return fmt.Errorf("%w: nil graph", ErrForm)
	}
	if g.Sub != sub {
		return fmt.Errorf("%w: subgraph placement", ErrForm)
	}
	if sub && (g.Strict || g.Directed) {
		return fmt.Errorf("%w: subgraph keyword", ErrForm)
	}
	if !sub && g.Bare {
		return fmt.Errorf("%w: bare graph", ErrForm)
	}
	if g.Bare && (g.Name.Text != "" || g.Name.HTML) {
		return fmt.Errorf("%w: bare subgraph name", ErrForm)
	}
	if err := validateAtom(g.Name); err != nil {
		return err
	}
	for _, stmt := range g.Stmts {
		if err := validateStmt(stmt); err != nil {
			return err
		}
	}
	return nil
}

func validateStmt(stmt *Stmt) error {
	if stmt == nil || !stmt.Kind.IsValid() {
		return fmt.Errorf("%w: statement", ErrForm)
	}
	switch stmt.Kind {
	case StmtNode:
		if stmt.Node == nil || len(stmt.Node.IDs) == 0 {
			return fmt.Errorf("%w: node", ErrForm)
		}
		for _, id := range stmt.Node.IDs {
			if err := validateID(id); err != nil {
				return err
			}
		}
		return validateAttrs(stmt.Node.Attrs)
	case StmtEdge:
		if stmt.Edge == nil || len(stmt.Edge.Ends) < 2 {
			return fmt.Errorf("%w: edge", ErrForm)
		}
		for _, end := range stmt.Edge.Ends {
			if err := validateEnd(end); err != nil {
				return err
			}
		}
		return validateAttrs(stmt.Edge.Attrs)
	case StmtAttr:
		if stmt.Attr == nil || !stmt.Attr.Kind.IsValid() {
			return fmt.Errorf("%w: attribute statement", ErrForm)
		}
		return validateAttrs(stmt.Attr.Attrs)
	case StmtAssign:
		if stmt.Assign == nil {
			return fmt.Errorf("%w: assignment", ErrForm)
		}
		if err := validateAtom(stmt.Assign.Name); err != nil {
			return err
		}
		return validateAtom(stmt.Assign.Value)
	case StmtSubgraph:
		return validateGraph(stmt.Subgraph, true)
	default:
		return fmt.Errorf("%w: statement", ErrForm)
	}
}

func validateEnd(end *End) error {
	if end == nil {
		return fmt.Errorf("%w: edge end", ErrForm)
	}
	if end.Sub != nil {
		if len(end.IDs) != 0 {
			return fmt.Errorf("%w: edge end", ErrForm)
		}
		return validateGraph(end.Sub, true)
	}
	if len(end.IDs) == 0 {
		return fmt.Errorf("%w: edge end", ErrForm)
	}
	for _, id := range end.IDs {
		if err := validateID(id); err != nil {
			return err
		}
	}
	return nil
}

func validateID(id *NodeID) error {
	if id == nil {
		return fmt.Errorf("%w: node id", ErrForm)
	}
	if err := validateAtom(id.Name); err != nil {
		return err
	}
	if id.HasPort {
		if err := validateAtom(id.Port); err != nil {
			return err
		}
	}
	if id.HasCompass {
		if !id.HasPort {
			return fmt.Errorf("%w: compass without port", ErrForm)
		}
		if err := validateAtom(id.Compass); err != nil {
			return err
		}
	}
	return nil
}

func validateAttrs(attrs []*Attr) error {
	for _, attr := range attrs {
		if attr == nil {
			return fmt.Errorf("%w: attribute", ErrForm)
		}
		if err := validateAtom(attr.Name); err != nil {
			return err
		}
		if err := validateAtom(attr.Value); err != nil {
			return err
		}
	}
	return nil
}

func validateAtom(atom Atom) error {
	if atom.HTML && !balancedHTML(atom.Text) {
		return fmt.Errorf("%w: html", ErrForm)
	}
	return nil
}

func balancedHTML(s string) bool {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '<':
			depth++
		case '>':
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0
}
