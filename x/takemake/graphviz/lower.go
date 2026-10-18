package graphviz

import (
	"fmt"

	xparse "github.com/lewtec/lewkit/x/parse"
)

func lowerFile(src []byte, root xparse.SyntaxNode[Term, Rule]) (*File, error) {
	file := New()
	for i := 0; i < root.ChildCount(); i++ {
		ch := root.Child(i)
		rule, ok := ch.Rule()
		if !ok || rule != RuleTop {
			continue
		}
		g, err := lowerTop(src, ch)
		if err != nil {
			return nil, err
		}
		file.Graphs = append(file.Graphs, g)
	}
	return file, nil
}

func lowerTop(src []byte, n xparse.SyntaxNode[Term, Rule]) (*Graph, error) {
	g := &Graph{}
	directed := false
	saw := false
	for i := 0; i < n.ChildCount(); i++ {
		ch := n.Child(i)
		if term, ok := ch.Term(); ok {
			switch term {
			case TermStrict:
				g.Strict = true
			case TermGraph:
				saw = true
			case TermDigraph:
				saw = true
				directed = true
				g.Directed = true
			}
			continue
		}
		rule, ok := ch.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleAtom:
			atom, err := lowerAtom(src, ch)
			if err != nil {
				return nil, err
			}
			g.Name = atom
		case RuleStatement:
			if err := lowerStmt(g, src, ch, directed); err != nil {
				return nil, err
			}
		}
	}
	if !saw {
		return nil, syntaxAt(n)
	}
	return g, nil
}

func lowerStmt(g *Graph, src []byte, n xparse.SyntaxNode[Term, Rule], directed bool) error {
	for i := 0; i < n.ChildCount(); i++ {
		ch := n.Child(i)
		rule, ok := ch.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleAttributeStatement:
			return lowerAttr(g, src, ch)
		case RuleLead:
			return lowerLead(g, src, ch, directed)
		case RuleSubgraphStatement:
			return lowerSub(g, src, ch, directed)
		}
	}
	return syntaxAt(n)
}

func lowerAttr(g *Graph, src []byte, n xparse.SyntaxNode[Term, Rule]) error {
	var kind AttrKind
	for i := 0; i < n.ChildCount(); i++ {
		ch := n.Child(i)
		term, ok := ch.Term()
		if !ok {
			continue
		}
		switch term {
		case TermGraph:
			kind = AttrGraph
		case TermNode:
			kind = AttrNode
		case TermEdge:
			kind = AttrEdge
		}
	}
	if !kind.IsValid() {
		return syntaxAt(n)
	}
	attrs, err := pairsIn(src, n)
	if err != nil {
		return err
	}
	g.Stmts = append(g.Stmts, &Stmt{
		Kind: StmtAttr,
		Attr: &AttrStmt{Kind: kind, Attrs: attrs},
	})
	return nil
}

func lowerLead(g *Graph, src []byte, n xparse.SyntaxNode[Term, Rule], directed bool) error {
	atomN := findRule(n, RuleAtom)
	tailN := findRule(n, RuleTail)
	if !atomN.Valid() || !tailN.Valid() {
		return syntaxAt(n)
	}
	name, err := lowerAtom(src, atomN)
	if err != nil {
		return err
	}
	if isAssign(tailN) {
		value, err := firstAtom(src, tailN)
		if err != nil {
			return err
		}
		g.Stmts = append(g.Stmts, &Stmt{
			Kind:   StmtAssign,
			Assign: &Assign{Name: name, Value: value},
		})
		return nil
	}
	id := &NodeID{Name: name}
	if portN := findRule(tailN, RulePort); portN.Valid() {
		if err := fillPort(src, id, portN); err != nil {
			return err
		}
	}
	listN := findRule(tailN, RuleList)
	ids := []*NodeID{id}
	var ends []*End
	saw := false
	if err := walkList(src, listN, &ids, &ends, &saw, directed); err != nil {
		return err
	}
	attrs, err := pairsIn(src, listN)
	if err != nil {
		return err
	}
	if saw {
		ends = append([]*End{{IDs: ids}}, ends...)
		if len(ends) < 2 {
			return syntaxAt(n)
		}
		g.Stmts = append(g.Stmts, &Stmt{
			Kind: StmtEdge,
			Edge: &Edge{Ends: ends, Attrs: attrs},
		})
		return nil
	}
	g.Stmts = append(g.Stmts, &Stmt{
		Kind: StmtNode,
		Node: &Node{IDs: ids, Attrs: attrs},
	})
	return nil
}

func lowerSub(g *Graph, src []byte, n xparse.SyntaxNode[Term, Rule], directed bool) error {
	blockN := findRule(n, RuleBlock)
	tailN := findRule(n, RuleStatementTail)
	if !blockN.Valid() {
		return syntaxAt(n)
	}
	sub, err := lowerBlock(src, blockN, directed)
	if err != nil {
		return err
	}
	if !tailN.Valid() || tailN.ChildCount() == 0 {
		g.Stmts = append(g.Stmts, &Stmt{Kind: StmtSubgraph, Subgraph: sub})
		return nil
	}
	ends := []*End{{Sub: sub}}
	if err := walkEdge(src, tailN, &ends, directed); err != nil {
		return err
	}
	if len(ends) < 2 {
		return syntaxAt(n)
	}
	attrs, err := pairsIn(src, tailN)
	if err != nil {
		return err
	}
	g.Stmts = append(g.Stmts, &Stmt{
		Kind: StmtEdge,
		Edge: &Edge{Ends: ends, Attrs: attrs},
	})
	return nil
}

func lowerBlock(src []byte, n xparse.SyntaxNode[Term, Rule], directed bool) (*Graph, error) {
	g := &Graph{Sub: true, Bare: true}
	for i := 0; i < n.ChildCount(); i++ {
		ch := n.Child(i)
		if term, ok := ch.Term(); ok && term == TermSubgraph {
			g.Bare = false
			continue
		}
		rule, ok := ch.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleAtom:
			atom, err := lowerAtom(src, ch)
			if err != nil {
				return nil, err
			}
			g.Name = atom
		case RuleStatement:
			if err := lowerStmt(g, src, ch, directed); err != nil {
				return nil, err
			}
		}
	}
	return g, nil
}

func walkList(src []byte, n xparse.SyntaxNode[Term, Rule], ids *[]*NodeID, ends *[]*End, saw *bool, directed bool) error {
	if !n.Valid() || n.ChildCount() == 0 {
		return nil
	}
	if isLit(n.Child(0), ",") {
		idN := findRule(n, RuleNodeIdentifier)
		rest := findRule(n, RuleList)
		if !idN.Valid() {
			return syntaxAt(n)
		}
		id, err := lowerNodeID(src, idN)
		if err != nil {
			return err
		}
		*ids = append(*ids, id)
		return walkList(src, rest, ids, ends, saw, directed)
	}
	if isEdgeOp(n.Child(0)) {
		*saw = true
		return walkEdge(src, n, ends, directed)
	}
	return nil
}

func walkEdge(src []byte, n xparse.SyntaxNode[Term, Rule], ends *[]*End, directed bool) error {
	if !n.Valid() || n.ChildCount() == 0 || !isEdgeOp(n.Child(0)) {
		return nil
	}
	if err := checkOp(src, n.Child(0), directed); err != nil {
		return err
	}
	endN := findRule(n, RuleEnd)
	if !endN.Valid() {
		return syntaxAt(n)
	}
	end, err := lowerEnd(src, endN, directed)
	if err != nil {
		return err
	}
	*ends = append(*ends, end)
	return walkEdge(src, findRule(n, RuleEdgeTail), ends, directed)
}

func lowerEnd(src []byte, n xparse.SyntaxNode[Term, Rule], directed bool) (*End, error) {
	if idsN := findRule(n, RuleIdentifiers); idsN.Valid() {
		ids, err := lowerIDs(src, idsN)
		if err != nil {
			return nil, err
		}
		return &End{IDs: ids}, nil
	}
	if blockN := findRule(n, RuleBlock); blockN.Valid() {
		sub, err := lowerBlock(src, blockN, directed)
		if err != nil {
			return nil, err
		}
		return &End{Sub: sub}, nil
	}
	return nil, syntaxAt(n)
}

func lowerIDs(src []byte, n xparse.SyntaxNode[Term, Rule]) ([]*NodeID, error) {
	var ids []*NodeID
	for i := 0; i < n.ChildCount(); i++ {
		ch := n.Child(i)
		rule, ok := ch.Rule()
		if !ok || rule != RuleNodeIdentifier {
			continue
		}
		id, err := lowerNodeID(src, ch)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, syntaxAt(n)
	}
	return ids, nil
}

func lowerNodeID(src []byte, n xparse.SyntaxNode[Term, Rule]) (*NodeID, error) {
	atomN := findRule(n, RuleAtom)
	if !atomN.Valid() {
		return nil, syntaxAt(n)
	}
	name, err := lowerAtom(src, atomN)
	if err != nil {
		return nil, err
	}
	id := &NodeID{Name: name}
	if portN := findRule(n, RulePort); portN.Valid() {
		if err := fillPort(src, id, portN); err != nil {
			return nil, err
		}
	}
	return id, nil
}

func fillPort(src []byte, id *NodeID, n xparse.SyntaxNode[Term, Rule]) error {
	var atoms []Atom
	for i := 0; i < n.ChildCount(); i++ {
		ch := n.Child(i)
		rule, ok := ch.Rule()
		if !ok || rule != RuleAtom {
			continue
		}
		atom, err := lowerAtom(src, ch)
		if err != nil {
			return err
		}
		atoms = append(atoms, atom)
	}
	if len(atoms) == 0 {
		return syntaxAt(n)
	}
	id.HasPort = true
	id.Port = atoms[0]
	if len(atoms) > 1 {
		id.HasCompass = true
		id.Compass = atoms[1]
	}
	return nil
}

func lowerAtom(src []byte, n xparse.SyntaxNode[Term, Rule]) (Atom, error) {
	if strs := findRule(n, RuleStrings); strs.Valid() {
		return lowerStrs(src, strs)
	}
	for i := 0; i < n.ChildCount(); i++ {
		ch := n.Child(i)
		term, ok := ch.Term()
		if !ok {
			continue
		}
		if term == TermIdentifier || term == TermNumber {
			return Atom{Text: ch.Text()}, nil
		}
	}
	return Atom{}, syntaxAt(n)
}

func firstAtom(src []byte, n xparse.SyntaxNode[Term, Rule]) (Atom, error) {
	if atomN := findRule(n, RuleAtom); atomN.Valid() {
		return lowerAtom(src, atomN)
	}
	return Atom{}, syntaxAt(n)
}

func lowerStrs(src []byte, n xparse.SyntaxNode[Term, Rule]) (Atom, error) {
	var parts []Atom
	for i := 0; i < n.ChildCount(); i++ {
		ch := n.Child(i)
		term, ok := ch.Term()
		if !ok {
			continue
		}
		switch term {
		case TermString:
			text, err := decodeStr(ch.Text())
			if err != nil {
				return Atom{}, syntaxAt(ch)
			}
			parts = append(parts, Atom{Text: text})
		case TermHTMLLabel:
			text, ok := decodeHTML(ch.Text())
			if !ok {
				return Atom{}, syntaxAt(ch)
			}
			parts = append(parts, Atom{Text: text, HTML: true})
		}
	}
	if len(parts) == 0 {
		return Atom{}, syntaxAt(n)
	}
	if len(parts) == 1 {
		return parts[0], nil
	}
	text := ""
	for _, part := range parts {
		text += part.Text
	}
	return Atom{Text: text}, nil
}

func decodeStr(raw string) (string, error) {
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return "", ErrSyntax
	}
	s := raw[1 : len(raw)-1]
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case '"':
				b = append(b, '"')
				i++
				continue
			case '\\':
				b = append(b, '\\', '\\')
				i++
				continue
			case '\n':
				i++
				continue
			}
		}
		b = append(b, c)
	}
	return string(b), nil
}

func decodeHTML(raw string) (string, bool) {
	if len(raw) < 2 || raw[0] != '<' || raw[len(raw)-1] != '>' {
		return "", false
	}
	return raw[1 : len(raw)-1], true
}

func pairsIn(src []byte, n xparse.SyntaxNode[Term, Rule]) ([]*Attr, error) {
	if !n.Valid() {
		return nil, nil
	}
	var out []*Attr
	var walk func(xparse.SyntaxNode[Term, Rule]) error
	walk = func(n xparse.SyntaxNode[Term, Rule]) error {
		if rule, ok := n.Rule(); ok {
			switch rule {
			case RulePair:
				attr, err := lowerPair(src, n)
				if err != nil {
					return err
				}
				out = append(out, attr)
				return nil
			case RuleBlock, RuleEnd, RuleNodeIdentifier, RuleAtom, RulePort:
				return nil
			}
		}
		for i := 0; i < n.ChildCount(); i++ {
			if err := walk(n.Child(i)); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(n); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func lowerPair(src []byte, n xparse.SyntaxNode[Term, Rule]) (*Attr, error) {
	var atoms []Atom
	for i := 0; i < n.ChildCount(); i++ {
		ch := n.Child(i)
		rule, ok := ch.Rule()
		if !ok || rule != RuleAtom {
			continue
		}
		atom, err := lowerAtom(src, ch)
		if err != nil {
			return nil, err
		}
		atoms = append(atoms, atom)
	}
	if len(atoms) != 2 {
		return nil, syntaxAt(n)
	}
	return &Attr{Name: atoms[0], Value: atoms[1]}, nil
}

func isAssign(n xparse.SyntaxNode[Term, Rule]) bool {
	for i := 0; i < n.ChildCount(); i++ {
		if isLit(n.Child(i), "=") {
			return true
		}
	}
	return false
}

func findRule(n xparse.SyntaxNode[Term, Rule], want Rule) xparse.SyntaxNode[Term, Rule] {
	for i := 0; i < n.ChildCount(); i++ {
		ch := n.Child(i)
		if rule, ok := ch.Rule(); ok && rule == want {
			return ch
		}
	}
	return xparse.SyntaxNode[Term, Rule]{}
}

func isLit(n xparse.SyntaxNode[Term, Rule], text string) bool {
	if !n.Valid() {
		return false
	}
	if _, ok := n.Rule(); ok {
		return false
	}
	if _, ok := n.Term(); ok {
		return false
	}
	return n.Text() == text
}

func isEdgeOp(n xparse.SyntaxNode[Term, Rule]) bool {
	return isLit(n, "->") || isLit(n, "--")
}

func checkOp(src []byte, op xparse.SyntaxNode[Term, Rule], directed bool) error {
	text := op.Text()
	if directed && text != "->" {
		return fmt.Errorf("%w: line %d: undirected edge in a digraph", ErrSyntax, op.Line())
	}
	if !directed && text != "--" {
		return fmt.Errorf("%w: line %d: directed edge in a graph", ErrSyntax, op.Line())
	}
	return nil
}
