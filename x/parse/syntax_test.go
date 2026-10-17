package parse

import (
	"testing"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/stretchr/testify/require"
)

type demoTerm string

const (
	demoNum  demoTerm = "num"
	demoPlus demoTerm = "+"
)

func (t demoTerm) String() string { return string(t) }

func (t demoTerm) IsValid() bool { return t == demoNum || t == demoPlus }

type demoRule string

const (
	demoExpr demoRule = "expr"
)

func (r demoRule) String() string { return string(r) }

func (r demoRule) IsValid() bool { return r == demoExpr }

func TestSyntaxTermsAndRules(t *testing.T) {
	var sx Syntax[demoTerm, demoRule]
	num := sx.Term(demoNum, digits)
	plus := sx.Term(demoPlus, hasBytes("+"))
	g := sx.Compose(
		sx.Eq(demoExpr, sx.Alt(
			sx.Left(1, sx.Seq(sx.Ref(demoExpr), plus, sx.Ref(demoExpr))),
			num,
		)),
		sx.Start(demoExpr),
	)
	m, err := sx.Compile("expr", g)
	require.NoError(t, err)
	root, err := m.Parse([]byte("1+2"))
	require.NoError(t, err)
	rule, ok := root.Rule()
	require.True(t, ok)
	require.Equal(t, demoExpr, rule)
	require.Equal(t, 3, root.ChildCount())
	op, ok := root.Child(1).Term()
	require.True(t, ok)
	require.Equal(t, demoPlus, op)
	_, ok = root.Child(1).Rule()
	require.False(t, ok)
	left, ok := root.Child(0).Rule()
	require.True(t, ok)
	require.Equal(t, demoExpr, left)
}

func TestSyntaxRejectsTwoMatchers(t *testing.T) {
	var sx Syntax[demoTerm, demoRule]
	_, err := sx.Compile("expr", sx.Compose(
		sx.Term(demoNum, digits),
		sx.Term(demoNum, spaces),
		sx.Eq(demoExpr, sx.Ref(demoExpr)),
	))
	require.ErrorIs(t, err, errs.ErrInvalid)
}

func TestSyntaxRejectsInvalid(t *testing.T) {
	var sx Syntax[demoTerm, demoRule]
	_, err := sx.Compile("expr", sx.Term(demoTerm("nope"), digits))
	require.ErrorIs(t, err, errs.ErrInvalid)
}
