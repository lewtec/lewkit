package treesitter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type raw struct {
	kind     string
	start    uint32
	end      uint32
	children []*raw
	named    []*raw
	field    string
	isNamed  bool
	isExtra  bool
	isError  bool
	hasError bool
	null     bool
	text     string
}

func (n *raw) IsNull() bool {
	return n == nil || n.null
}
func (n *raw) Type() string {
	if n == nil {
		return ""
	}
	return n.kind
}
func (n *raw) StartByte() uint32 {
	if n == nil {
		return 0
	}
	return n.start
}
func (n *raw) EndByte() uint32 {
	if n == nil {
		return 0
	}
	return n.end
}
func (n *raw) ChildCount() uint32 {
	if n == nil {
		return 0
	}
	return uint32(len(n.children))
}
func (n *raw) Child(index uint32) *raw {
	if n == nil || int(index) >= len(n.children) {
		return nil
	}
	return n.children[index]
}
func (n *raw) NamedChildCount() uint32 {
	if n == nil {
		return 0
	}
	return uint32(len(n.named))
}
func (n *raw) NamedChild(index uint32) *raw {
	if n == nil || int(index) >= len(n.named) {
		return nil
	}
	return n.named[index]
}
func (n *raw) FieldNameForChild(uint32) string {
	if n == nil {
		return ""
	}
	return n.field
}
func (n *raw) IsNamed() bool  { return n != nil && n.isNamed }
func (n *raw) IsExtra() bool  { return n != nil && n.isExtra }
func (n *raw) IsError() bool  { return n != nil && n.isError }
func (n *raw) HasError() bool { return n != nil && n.hasError }
func (n *raw) String() string {
	if n == nil {
		return ""
	}
	return n.text
}

func TestSyntaxKeepsTheTreeOnANullChild(t *testing.T) {
	tree := &struct{ id int }{id: 7}
	root := &raw{
		kind:     "root",
		start:    1,
		end:      4,
		children: []*raw{nil},
		isNamed:  true,
		text:     "(root)",
	}
	node := NewSyntax[raw](tree, root)
	require.Equal(t, "root", node.Type())
	require.Equal(t, uint32(1), node.StartByte())
	require.Equal(t, uint32(4), node.EndByte())
	require.False(t, node.IsNull())
	require.True(t, node.IsNamed())
	require.Equal(t, "(root)", node.String())

	child := node.Child(0)
	require.True(t, child.IsNull())
	require.Empty(t, child.Type())
	require.Equal(t, uint32(0), child.ChildCount())
	held := child.(Syntax[raw, *raw])
	require.Same(t, tree, held.tree)
}

func TestNameOf(t *testing.T) {
	got := NameOf("json", []string{"c", "json"}, func(name string) (string, bool) {
		if name == "json" {
			return "json", true
		}
		return "", false
	})
	require.Equal(t, "json", got)
	require.Empty(t, NameOf("make", []string{"c"}, func(string) (string, bool) {
		return "", false
	}))
}
