package parse

import (
	"testing"
	"unicode"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/stretchr/testify/require"
)

func arithOps() *Form {
	return Eq("expr", Alt(
		Left(1, Seq(Ref("expr"), Lit("+"), Ref("expr"))),
		Left(1, Seq(Ref("expr"), Lit("-"), Ref("expr"))),
		Left(2, Seq(Ref("expr"), Lit("*"), Ref("expr"))),
		Prec(3, Seq(Lit("-"), Ref("expr"))),
	))
}

func arithAtoms() *Form {
	return Compose(
		Eq("expr", Alt(
			Ref("num"),
			Seq(Lit("("), Ref("expr"), Lit(")")),
		)),
		Tok("num", digits),
		Skip(Tok("ws", spaces)),
		Start("expr"),
	)
}

func arith() *Form {
	return Compose(arithOps(), arithAtoms())
}

func compileForm(t *testing.T, name string, f *Form) *Language {
	t.Helper()
	g, err := Build(name, f)
	require.NoError(t, err)
	lang, err := g.Compile()
	require.NoError(t, err)
	return lang
}

func TestComposeExpr(t *testing.T) {
	lang := compileForm(t, "expr", arith())
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

	tree, err = lang.Parse([]byte(" 1 + 2 "))
	require.NoError(t, err)
	require.Equal(t, "(expr (expr (num 1)) + (expr (num 2)))", tree.SExpr())
}

func TestComposeReusesTheForm(t *testing.T) {
	f := arith()
	a := compileForm(t, "a", f)
	b := compileForm(t, "b", f)
	tree, err := a.Parse([]byte("1+2*3"))
	require.NoError(t, err)
	require.Equal(t, "(expr (expr (num 1)) + (expr (expr (num 2)) * (expr (num 3))))", tree.SExpr())
	tree, err = b.Parse([]byte("1+2*3"))
	require.NoError(t, err)
	require.Equal(t, "(expr (expr (num 1)) + (expr (expr (num 2)) * (expr (num 3))))", tree.SExpr())
}

func TestComposeAddsAnAlternative(t *testing.T) {
	f := Compose(arith(), Eq("expr", Seq(Ref("expr"), Lit("!"))))
	lang := compileForm(t, "fact", f)
	tree, err := lang.Parse([]byte("2!"))
	require.NoError(t, err)
	require.Equal(t, "(expr (expr (num 2)) !)", tree.SExpr())
}

func TestComposeRight(t *testing.T) {
	f := Compose(
		Tok("num", digits),
		Eq("expr", Alt(
			Right(1, Seq(Ref("expr"), Lit("^"), Ref("expr"))),
			Ref("num"),
		)),
		Start("expr"),
	)
	lang := compileForm(t, "pow", f)
	tree, err := lang.Parse([]byte("1^2^3"))
	require.NoError(t, err)
	require.Equal(t, "(expr (expr (num 1)) ^ (expr (expr (num 2)) ^ (expr (num 3))))", tree.SExpr())
}

func TestComposeKeyword(t *testing.T) {
	f := Compose(
		Word("id", unicode.IsLetter, unicode.IsLetter),
		Key("if", "id"),
		Skip(Tok("ws", spaces)),
		Eq("stmt", Alt(
			Seq(Ref("if"), Ref("id")),
			Ref("id"),
		)),
		Start("stmt"),
	)
	lang := compileForm(t, "kw", f)
	tree, err := lang.Parse([]byte("if x"))
	require.NoError(t, err)
	require.Equal(t, "(stmt if (id x))", tree.SExpr())
	tree, err = lang.Parse([]byte("iff"))
	require.NoError(t, err)
	require.Equal(t, "(stmt (id iff))", tree.SExpr())
}

func TestComposeScan(t *testing.T) {
	fn := func(src []byte, at int, _ func(SymbolID) bool, st *LexState) (SymbolID, int, int, bool) {
		if at >= len(src) {
			return SymEOF, at, at, true
		}
		id, ok := st.Symbol("x")
		if !ok {
			return 0, at, at, false
		}
		return id, at, at + 1, true
	}
	f := Compose(
		Tok("x", nil),
		Eq("s", Repeat1(Ref("x"))),
		Scan(fn),
		Start("s"),
	)
	lang := compileForm(t, "s", f)
	tree, err := lang.Parse([]byte("ab"))
	require.NoError(t, err)
	require.Equal(t, "(s (x a) (x b))", tree.SExpr())
}

func TestComposeRejects(t *testing.T) {
	_, err := Build("x", Compose(Tok("num", digits), Tok("num", spaces)))
	require.ErrorIs(t, err, errs.ErrInvalid)

	_, err = Build("x", Compose(Start("a"), Start("b")))
	require.ErrorIs(t, err, errs.ErrInvalid)

	fn := func([]byte, int, func(SymbolID) bool, *LexState) (SymbolID, int, int, bool) {
		return SymEOF, 0, 0, true
	}
	_, err = Build("x", Compose(Scan(fn), Scan(fn)))
	require.ErrorIs(t, err, errs.ErrInvalid)
}
