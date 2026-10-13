package treesitter

// GrammarNode is the node pointer the pure-Go engines share.
// N is the node struct. Methods live on the pointer.
type GrammarNode[N any] interface {
	*N
	IsNull() bool
	Type() string
	StartByte() uint32
	EndByte() uint32
	ChildCount() uint32
	Child(index uint32) *N
	NamedChildCount() uint32
	NamedChild(index uint32) *N
	FieldNameForChild(index uint32) string
	IsNamed() bool
	IsExtra() bool
	IsError() bool
	HasError() bool
	String() string
}

// Syntax is one grammar node. tree keeps the parse result reachable
// for as long as a caller holds the node.
type Syntax[N any, P GrammarNode[N]] struct {
	tree any
	n    P
}

// NewSyntax returns the node as a [Node]. A null pointer is an empty node
// that still holds tree.
func NewSyntax[N any, P GrammarNode[N]](tree any, node P) Node {
	return Syntax[N, P]{tree: tree, n: node}
}

// NameOf returns the registered name whose grammar value is raw.
func NameOf[L comparable](raw L, names []string, get func(string) (L, bool)) string {
	for _, name := range names {
		got, ok := get(name)
		if ok && got == raw {
			return name
		}
	}
	return ""
}

func (s Syntax[N, P]) live() bool { return s.n != nil && !s.n.IsNull() }

func (s Syntax[N, P]) Type() string {
	if !s.live() {
		return ""
	}
	return s.n.Type()
}

func (s Syntax[N, P]) StartByte() uint32 {
	if !s.live() {
		return 0
	}
	return s.n.StartByte()
}

func (s Syntax[N, P]) EndByte() uint32 {
	if !s.live() {
		return 0
	}
	return s.n.EndByte()
}

func (s Syntax[N, P]) ChildCount() uint32 {
	if !s.live() {
		return 0
	}
	return s.n.ChildCount()
}

func (s Syntax[N, P]) Child(index uint32) Node {
	if !s.live() {
		return Syntax[N, P]{tree: s.tree}
	}
	return Syntax[N, P]{tree: s.tree, n: s.n.Child(index)}
}

func (s Syntax[N, P]) NamedChildCount() uint32 {
	if !s.live() {
		return 0
	}
	return s.n.NamedChildCount()
}

func (s Syntax[N, P]) NamedChild(index uint32) Node {
	if !s.live() {
		return Syntax[N, P]{tree: s.tree}
	}
	return Syntax[N, P]{tree: s.tree, n: s.n.NamedChild(index)}
}

func (s Syntax[N, P]) FieldNameForChild(index uint32) string {
	if !s.live() {
		return ""
	}
	return s.n.FieldNameForChild(index)
}

func (s Syntax[N, P]) IsNull() bool  { return !s.live() }
func (s Syntax[N, P]) IsNamed() bool { return s.live() && s.n.IsNamed() }
func (s Syntax[N, P]) IsExtra() bool { return s.live() && s.n.IsExtra() }
func (s Syntax[N, P]) IsError() bool { return s.live() && s.n.IsError() }
func (s Syntax[N, P]) HasError() bool {
	return s.live() && s.n.HasError()
}

func (s Syntax[N, P]) String() string {
	if !s.live() {
		return ""
	}
	return s.n.String()
}
