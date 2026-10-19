package parse

import (
	"errors"
	"testing"
	"unicode"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/stretchr/testify/require"
)

func digits(src []byte, at int) int {
	if at >= len(src) || src[at] < '0' || src[at] > '9' {
		return 0
	}
	i := at + 1
	for i < len(src) && src[i] >= '0' && src[i] <= '9' {
		i++
	}
	return i - at
}

func spaces(src []byte, at int) int {
	i := at
	for i < len(src) && (src[i] == ' ' || src[i] == '\t') {
		i++
	}
	if i == at {
		return 0
	}
	return i - at
}

func exprLang(t *testing.T) *Language {
	t.Helper()
	g := New("expr")
	num := g.Token("num").Match(digits)
	ws := g.Token("ws").Match(spaces)
	g.Skip(ws)
	plus := g.Literal("+")
	minus := g.Literal("-")
	star := g.Literal("*")
	lp := g.Literal("(")
	rp := g.Literal(")")
	e := g.Rule("expr")
	e.Seq(g.Sym("expr"), plus, g.Sym("expr")).Left(1)
	e.Seq(g.Sym("expr"), minus, g.Sym("expr")).Left(1)
	e.Seq(g.Sym("expr"), star, g.Sym("expr")).Left(2)
	e.Seq(minus, g.Sym("expr")).Prec(3)
	e.Seq(num)
	e.Seq(lp, g.Sym("expr"), rp)
	g.Start("expr")
	lang, err := g.Compile()
	require.NoError(t, err)
	return lang
}

func TestExprPrecedence(t *testing.T) {
	lang := exprLang(t)
	tree, err := lang.Parse([]byte("1+2*3"))
	require.NoError(t, err)
	require.Equal(t, "(expr (expr (num 1)) + (expr (expr (num 2)) * (expr (num 3))))", tree.SExpr())

	tree, err = lang.Parse([]byte("1+2+3"))
	require.NoError(t, err)
	require.Equal(t, "(expr (expr (expr (num 1)) + (expr (num 2))) + (expr (num 3)))", tree.SExpr())

	tree, err = lang.Parse([]byte("-1+2"))
	require.NoError(t, err)
	require.Equal(t, "(expr (expr - (expr (num 1))) + (expr (num 2)))", tree.SExpr())

	tree, err = lang.Parse([]byte("(1+2)*3"))
	require.NoError(t, err)
	require.Equal(t, "(expr (expr ( (expr (expr (num 1)) + (expr (num 2))) )) * (expr (num 3)))", tree.SExpr())
}

func TestExprWhitespace(t *testing.T) {
	lang := exprLang(t)
	tree, err := lang.Parse([]byte(" 1 + 2 "))
	require.NoError(t, err)
	require.Equal(t, "(expr (expr (num 1)) + (expr (num 2)))", tree.SExpr())
	leaves := tree.Leaves()
	require.Equal(t, []Leaf{
		{Name: "num", Text: "1", Start: 1, End: 2},
		{Name: "+", Text: "+", Start: 3, End: 4},
		{Name: "num", Text: "2", Start: 5, End: 6},
	}, leaves)
}

func TestRightAssoc(t *testing.T) {
	g := New("pow")
	num := g.Token("num").Match(digits)
	caret := g.Literal("^")
	e := g.Rule("expr")
	e.Seq(g.Sym("expr"), caret, g.Sym("expr")).Right(1)
	e.Seq(num)
	g.Start("expr")
	lang, err := g.Compile()
	require.NoError(t, err)
	tree, err := lang.Parse([]byte("1^2^3"))
	require.NoError(t, err)
	require.Equal(t, "(expr (expr (num 1)) ^ (expr (expr (num 2)) ^ (expr (num 3))))", tree.SExpr())
}

// TestReconvergingStacks keeps two stacks that share a state inside a
// call and diverge again at the next token. Collapsing them on state
// alone accepts only the expression composite and misses the block.
func TestReconvergingStacks(t *testing.T) {
	g := New("rec")
	id := g.Token("id").Word(unicode.IsLetter, unicode.IsLetter)
	g.Skip(g.Token("ws").Match(spaces))
	lp := g.Literal("(")
	rp := g.Literal(")")
	lb := g.Literal("{")
	rb := g.Literal("}")
	semi := g.Literal(";")
	kw := g.Keyword("if", id)
	cond := g.Rule("cond")
	cond.Seq(id)
	cond.Seq(g.Sym("cond"), lp, id, rp)
	expr := g.Rule("expr")
	expr.Seq(id)
	expr.Seq(g.Sym("expr"), lp, id, rp).Dynamic(1)
	expr.Seq(g.Sym("expr"), lb, rb).Prec(-1)
	g.Rule("block").Seq(lb, rb)
	st := g.Rule("stmt")
	st.Seq(kw, g.Sym("cond"), g.Sym("block"))
	st.Seq(kw, g.Sym("expr"), semi)
	g.Start("stmt")
	lang, err := g.Compile()
	require.NoError(t, err)
	tree, err := lang.Parse([]byte("if a(x){}"))
	require.NoError(t, err)
	require.Contains(t, tree.SExpr(), "cond")
}

func TestDynamicFork(t *testing.T) {
	g := New("amb")
	x := g.Literal("x")
	g.Rule("a").Seq(x).Dynamic(1)
	g.Rule("b").Seq(x).Dynamic(2)
	s := g.Rule("s")
	s.Seq(g.Sym("a"))
	s.Seq(g.Sym("b"))
	g.Start("s")
	lang, err := g.Compile()
	require.NoError(t, err)
	require.Greater(t, lang.Forks(), 0)
	tree, err := lang.Parse([]byte("x"))
	require.NoError(t, err)
	require.Equal(t, "(s (b x))", tree.SExpr())
}

func TestAbandonedForkIsNotInTheTree(t *testing.T) {
	g := New("amb")
	x := g.Literal("x")
	g.Rule("a").Seq(x).Dynamic(1)
	g.Rule("b").Seq(x).Dynamic(2)
	s := g.Rule("s")
	s.Seq(g.Sym("a"))
	s.Seq(g.Sym("b"))
	g.Start("s")
	lang, err := g.Compile()
	require.NoError(t, err)
	tree, err := lang.Parse([]byte("x"))
	require.NoError(t, err)
	require.Equal(t, countReachable(tree), len(tree.nodes)-1)
	require.NotContains(t, namesFrom(tree, tree.root), "a")
}

func TestBytesBorrowsSource(t *testing.T) {
	lang := exprLang(t)
	src := []byte("1+2*3")
	tree, err := lang.Parse(src)
	require.NoError(t, err)
	leaf := tree.Root().Child(0).Child(0)
	require.Equal(t, "num", leaf.Name())
	got := leaf.Bytes()
	require.Equal(t, []byte("1"), got)
	require.Equal(t, "1", leaf.Text())
	start := leaf.Start()
	require.Same(t, &src[start], &got[0])
	require.Same(t, &tree.Src[start], &got[0])
}

func TestPathBoundIsSyntax(t *testing.T) {
	g := New("expr")
	num := g.Token("num").Match(digits)
	plus := g.Literal("+")
	e := g.Rule("expr")
	e.Seq(g.Sym("expr"), plus, g.Sym("expr"))
	e.Seq(num)
	g.Start("expr")
	lang, err := g.Compile()
	require.NoError(t, err)
	require.Greater(t, lang.Forks(), 0)

	tree, err := lang.Parse([]byte("1+2+3"))
	require.NoError(t, err)
	var nums []string
	for _, leaf := range tree.Leaves() {
		if leaf.Name == "num" {
			nums = append(nums, leaf.Text)
		}
	}
	require.Equal(t, []string{"1", "2", "3"}, nums)

	old := pathLimit
	pathLimit = 1
	t.Cleanup(func() { pathLimit = old })
	_, err = lang.Parse([]byte("1+2+3"))
	require.ErrorIs(t, err, errs.ErrSyntax)
	require.ErrorContains(t, err, "too many stacks")
}

func countReachable(tree *Tree) int {
	if tree == nil || tree.root == 0 {
		return 0
	}
	seen := make([]bool, len(tree.nodes))
	var n int
	var walk func(uint32)
	walk = func(id uint32) {
		if id == 0 || int(id) >= len(tree.nodes) || seen[id] {
			return
		}
		seen[id] = true
		n++
		nd := tree.nodes[id]
		for i := uint32(0); i < nd.nchild; i++ {
			at := nd.child + i
			if int(at) < len(tree.kids) {
				walk(tree.kids[at])
			}
		}
	}
	walk(tree.root)
	return n
}

func namesFrom(tree *Tree, id uint32) []string {
	if tree == nil || id == 0 || int(id) >= len(tree.nodes) {
		return nil
	}
	nd := tree.nodes[id]
	out := []string{tree.lang.syms[nd.kind].name}
	for i := uint32(0); i < nd.nchild; i++ {
		at := nd.child + i
		if int(at) < len(tree.kids) {
			out = append(out, namesFrom(tree, tree.kids[at])...)
		}
	}
	return out
}

func TestKeywordVersusWord(t *testing.T) {
	g := New("kw")
	ident := g.Token("ident").Word(unicode.IsLetter, unicode.IsLetter)
	kw := g.Keyword("if", ident)
	g.Skip(g.Token("ws").Match(spaces))
	g.Rule("stmt").Seq(kw, ident)
	g.Rule("stmt").Seq(ident)
	g.Start("stmt")
	lang, err := g.Compile()
	require.NoError(t, err)

	tree, err := lang.Parse([]byte("if x"))
	require.NoError(t, err)
	require.Equal(t, "(stmt if (ident x))", tree.SExpr())

	tree, err = lang.Parse([]byte("iff"))
	require.NoError(t, err)
	require.Equal(t, "(stmt (ident iff))", tree.SExpr())

	_, err = lang.Parse([]byte("if"))
	require.Error(t, err)
}

func TestHiddenList(t *testing.T) {
	g := New("list")
	num := g.Token("num").Match(digits)
	comma := g.Literal(",")
	ws := g.Token("ws").Match(spaces)
	g.Skip(ws)
	g.Rule("list").Seq(g.Sep(num, comma))
	g.Start("list")
	lang, err := g.Compile()
	require.NoError(t, err)
	tree, err := lang.Parse([]byte("1, 2, 3"))
	require.NoError(t, err)
	require.Equal(t, "(list (num 1) , (num 2) , (num 3))", tree.SExpr())
	tree, err = lang.Parse([]byte(""))
	require.NoError(t, err)
	require.Equal(t, "(list)", tree.SExpr())
}

func TestSyntaxError(t *testing.T) {
	lang := exprLang(t)
	_, err := lang.Parse([]byte("1+"))
	require.Error(t, err)
	require.ErrorIs(t, err, errs.ErrSyntax)
	var syn *Error
	require.True(t, errors.As(err, &syn))
	require.Equal(t, 2, syn.Offset)
	require.Equal(t, 1, syn.Line)
	require.Equal(t, 3, syn.Col)

	_, err = lang.Parse([]byte("1+\n@"))
	require.Error(t, err)
	require.True(t, errors.As(err, &syn))
	require.Equal(t, 2, syn.Offset)
	require.Equal(t, 1, syn.Line)

	g := New("nl")
	num := g.Token("num").Match(digits)
	g.Skip(g.Token("ws").Match(func(src []byte, at int) int {
		i := at
		for i < len(src) && (src[i] == ' ' || src[i] == '\n') {
			i++
		}
		if i == at {
			return 0
		}
		return i - at
	}))
	e := g.Rule("expr")
	e.Seq(g.Sym("expr"), g.Literal("+"), g.Sym("expr")).Left(1)
	e.Seq(num)
	g.Start("expr")
	nl, err := g.Compile()
	require.NoError(t, err)
	_, err = nl.Parse([]byte("1+\n@"))
	require.Error(t, err)
	require.True(t, errors.As(err, &syn))
	require.Equal(t, 2, syn.Line)
	require.Equal(t, 1, syn.Col)
}

func TestCompileRejects(t *testing.T) {
	g := New("bad")
	g.Rule("expr")
	_, err := g.Compile()
	require.ErrorIs(t, err, errs.ErrInvalid)

	g = New("bad")
	skip := g.Token("ws").Match(spaces)
	g.Skip(skip)
	g.Rule("expr").Seq(skip)
	_, err = g.Compile()
	require.ErrorIs(t, err, errs.ErrInvalid)
}
