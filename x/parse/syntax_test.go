package parse

import (
	"testing"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/stretchr/testify/require"
)

type demoTerm string

const (
	demoNumber demoTerm = "number"
	demoPlus   demoTerm = "+"
)

func (term demoTerm) String() string { return string(term) }

func (term demoTerm) IsValid() bool { return term == demoNumber || term == demoPlus }

type demoRule string

const (
	demoExpression demoRule = "expression"
)

func (rule demoRule) String() string { return string(rule) }

func (rule demoRule) IsValid() bool { return rule == demoExpression }

func TestSyntaxTermsAndRules(t *testing.T) {
	var syntax Syntax[demoTerm, demoRule]
	number := syntax.Term(demoNumber, digits)
	plus := syntax.Term(demoPlus, hasBytes("+"))
	grammar := syntax.Compose(
		syntax.Eq(demoExpression, syntax.Alt(
			syntax.Left(1, syntax.Seq(syntax.Ref(demoExpression), plus, syntax.Ref(demoExpression))),
			number,
		)),
		syntax.Start(demoExpression),
	)
	model, err := syntax.Compile("expression", grammar)
	require.NoError(t, err)
	root, err := model.Parse([]byte("1+2"))
	require.NoError(t, err)
	rule, ok := root.Rule()
	require.True(t, ok)
	require.Equal(t, demoExpression, rule)
	require.Equal(t, 3, root.ChildCount())
	operator, ok := root.Child(1).Term()
	require.True(t, ok)
	require.Equal(t, demoPlus, operator)
	_, ok = root.Child(1).Rule()
	require.False(t, ok)
	left, ok := root.Child(0).Rule()
	require.True(t, ok)
	require.Equal(t, demoExpression, left)
}

func TestSyntaxNodeLine(t *testing.T) {
	var syntax Syntax[demoTerm, demoRule]
	number := syntax.Term(demoNumber, digits)
	plus := syntax.Term(demoPlus, hasBytes("+"))
	newline := syntax.Lit("\n")
	grammar := syntax.Compose(
		syntax.Eq(demoExpression, syntax.Seq(number, newline, plus, number)),
		syntax.Start(demoExpression),
	)
	model, err := syntax.Compile("expression", grammar)
	require.NoError(t, err)
	root, err := model.Parse([]byte("1\n+2"))
	require.NoError(t, err)
	require.Equal(t, 1, root.Child(0).Line())
	require.Equal(t, 2, root.Child(2).Line())
	require.Equal(t, 1, (SyntaxNode[demoTerm, demoRule]{}).Line())
}

func TestSyntaxRejectsTwoMatchers(t *testing.T) {
	var syntax Syntax[demoTerm, demoRule]
	_, err := syntax.Compile("expression", syntax.Compose(
		syntax.Term(demoNumber, digits),
		syntax.Term(demoNumber, spaces),
		syntax.Eq(demoExpression, syntax.Ref(demoExpression)),
	))
	require.ErrorIs(t, err, errs.ErrInvalid)
}

func TestSyntaxRejectsInvalid(t *testing.T) {
	var syntax Syntax[demoTerm, demoRule]
	_, err := syntax.Compile("expression", syntax.Term(demoTerm("nope"), digits))
	require.ErrorIs(t, err, errs.ErrInvalid)
}
