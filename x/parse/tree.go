package parse

import (
	"sort"
	"strings"
)

const (
	flagNamed uint8 = 1 << iota
	flagHidden
)

// node is one record in a tree's arena. Children are a range of kid indexes.
// Text is src[start:end]. Node 0 is a sentinel and is never a real node.
type node struct {
	kind   SymbolID
	flags  uint8
	start  uint32
	end    uint32
	child  uint32
	nchild uint32
	dyn    int32
	prod   uint16
}

// Tree is one concrete syntax tree. Nodes live in one array. Token text is
// a slice of Src; the tree does not copy it.
type Tree struct {
	lang  *Language
	Src   []byte
	nodes []node
	kids  []uint32
	lines []int
	root  uint32
}

// Leaf is one terminal in source order.
type Leaf struct {
	Name       string
	Text       string
	Start, End int
}

// Node is a handle into a tree.
type Node struct {
	tree *Tree
	id   uint32
}

// Root is the start symbol, or the zero node when the tree is empty.
func (t *Tree) Root() Node {
	if t == nil {
		return Node{}
	}
	return Node{tree: t, id: t.root}
}

// Valid reports whether the handle points at a node.
func (n Node) Valid() bool {
	return n.tree != nil && n.id != 0 && int(n.id) < len(n.tree.nodes)
}

// Name is the grammar name of the node.
func (n Node) Name() string {
	if !n.Valid() {
		return ""
	}
	return n.tree.lang.syms[n.tree.nodes[n.id].kind].name
}

// Start is the first source byte of the node.
func (n Node) Start() int {
	if !n.Valid() {
		return 0
	}
	return int(n.tree.nodes[n.id].start)
}

// Line is the 1-based line of Start. It uses the line index built with the tree.
// A node that is not in a tree is line 1.
func (n Node) Line() int {
	if !n.Valid() {
		return 1
	}
	line, _ := lineCol(n.tree.lines, n.Start())
	return line
}

// End is the byte after the node.
func (n Node) End() int {
	if !n.Valid() {
		return 0
	}
	return int(n.tree.nodes[n.id].end)
}

// Text is the source slice. A zero-width semicolon is the inserted "\n".
func (n Node) Text() string {
	if !n.Valid() {
		return ""
	}
	return n.tree.lit(n.tree.nodes[n.id])
}

// Named reports whether the grammar named this node.
func (n Node) Named() bool {
	if !n.Valid() {
		return false
	}
	return n.tree.nodes[n.id].flags&flagNamed != 0
}

// ChildCount is the number of children after hidden rules are flattened.
func (n Node) ChildCount() int {
	if !n.Valid() {
		return 0
	}
	return int(n.tree.nodes[n.id].nchild)
}

// Child returns one child. An out-of-range index is the zero node.
func (n Node) Child(i int) Node {
	if !n.Valid() || i < 0 || i >= n.ChildCount() {
		return Node{}
	}
	nd := n.tree.nodes[n.id]
	return Node{tree: n.tree, id: n.tree.kids[nd.child+uint32(i)]}
}

// SExpr prints named nodes as (name ...). An anonymous leaf is its text.
// An inserted semicolon prints as ";".
func (t *Tree) SExpr() string {
	if t == nil || t.root == 0 {
		return ""
	}
	var b strings.Builder
	t.write(&b, t.root)
	return b.String()
}

func (t *Tree) write(b *strings.Builder, id uint32) {
	n := t.nodes[id]
	info := t.lang.syms[n.kind]
	if info.terminal {
		text := t.shown(n)
		if n.flags&flagNamed == 0 {
			b.WriteString(text)
			return
		}
		b.WriteByte('(')
		b.WriteString(info.name)
		if text != "" && text != info.name {
			b.WriteByte(' ')
			b.WriteString(text)
		}
		b.WriteByte(')')
		return
	}
	b.WriteByte('(')
	b.WriteString(info.name)
	for i := uint32(0); i < n.nchild; i++ {
		b.WriteByte(' ')
		t.write(b, t.kids[n.child+i])
	}
	b.WriteByte(')')
}

// Leaves walks terminals in order. Hidden rules are already flattened.
func (t *Tree) Leaves() []Leaf {
	if t == nil || t.root == 0 {
		return nil
	}
	var out []Leaf
	var walk func(uint32)
	walk = func(id uint32) {
		if id == 0 {
			return
		}
		n := t.nodes[id]
		if t.lang.syms[n.kind].terminal {
			out = append(out, Leaf{
				Name:  t.lang.syms[n.kind].name,
				Text:  t.lit(n),
				Start: int(n.start),
				End:   int(n.end),
			})
			return
		}
		for i := uint32(0); i < n.nchild; i++ {
			walk(t.kids[n.child+i])
		}
	}
	walk(t.root)
	return out
}

func (t *Tree) lit(n node) string {
	if t.lang.syms[n.kind].name == ";" && n.start == n.end {
		return "\n"
	}
	if int(n.end) > len(t.Src) || int(n.start) > int(n.end) {
		return ""
	}
	return string(t.Src[n.start:n.end])
}

func (t *Tree) shown(n node) string {
	text := t.lit(n)
	if t.lang.syms[n.kind].name == ";" && (text == "\n" || text == "") {
		return ";"
	}
	return text
}

func lineStarts(src []byte) []int {
	lines := []int{0}
	for i, c := range src {
		if c == '\n' {
			lines = append(lines, i+1)
		}
	}
	return lines
}

func lineCol(lines []int, off int) (int, int) {
	if len(lines) == 0 {
		return 1, 1
	}
	if off < 0 {
		off = 0
	}
	i := sort.Search(len(lines), func(i int) bool { return lines[i] > off }) - 1
	if i < 0 {
		i = 0
	}
	return i + 1, off - lines[i] + 1
}
