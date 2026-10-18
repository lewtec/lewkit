package golang

import (
	"go/parser"
	"go/scanner"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/errs"
)

func TestLanguageCompiles(t *testing.T) {
	lang, err := Language()
	require.NoError(t, err)
	t.Logf("states %d forks %d", lang.States(), lang.Forks())
	require.Positive(t, lang.States())
}

func TestGoSnippets(t *testing.T) {
	cases := []string{
		"package p\n",
		"package p\nfunc f() {}\n",
		"package p\nimport \"fmt\"\nfunc f() { fmt.Println(1) }\n",
		"package p\nimport (\n\"fmt\"\n. \"os\"\n)\n",
		"package p\nfunc f(a, b int, c ...string) {}\n",
		"package p\nfunc f(int, string) {}\n",
		"package p\nfunc f(a.B) {}\n",
		"package p\nfunc f(a, b.C) {}\n",
		"package p\nfunc f(\n a int) {}\n",
		"package p\nfunc f(a int,\nb int) {}\n",
		"package p\nfunc F[T any](a T) T { return a }\n",
		"package p\nfunc (R) M[T any]() {}\n",
		"package p\ntype T[A any] int\n",
		"package p\ntype T[A]int\n",
		"package p\nconst (A = 1)\n",
		"package p\nconst (\nA = 1\nB\n)\n",
		"package p\nvar (\na int = 1\nb = 2\nc int\n)\n",
		"package p\ntype S struct {\nA int\nB int `json:\"b\"`\n*U\nV\n}\n",
		"package p\ntype I interface {\nM(int) int\n~int | string\n}\n",
		"package p\nfunc f() {\nx := T{1}\nT{1}\n[]int{1}\n_ = []byte(\"hi\")\n_ = make(chan int)\n_ = chan int(nil)\n_ = map[int]int{1: 2}\nf([]int{1, 2})\nf[int](x)\nf[int, string](x)\na[i][j] = 1\na[i:j:k]\n}\n",
		"package p\nfunc f() {\nif x {\n} else {\n}\nif x := 1; x > 0 {\n} else if y {\n} else {\n}\nif []int{1} {\n}\nif (T{1} == T{2}) {\n}\n}\n",
		"package p\nfunc f() {\nfor {}\nfor i := 0; i < 10; i++ {\n}\nfor range xs {\n}\nfor i, j := range xs {\n}\n}\n",
		"package p\nfunc f() {\nswitch {\ncase 1:\nx\ncase 2:\n}\nswitch x.(type) {\ncase int:\n}\nswitch t := x.(type) {\ncase int:\n_ = t\ndefault:\n}\n}\n",
		"package p\nfunc f() {\nselect {\ncase c <- x:\ncase v := <-c:\ncase <-c:\ndefault:\n}\ngo f()\ndefer f()\nch <- 1\n<-ch\nreturn T{1}\nL: for {\nbreak L\ncontinue L\ngoto L\n}\n{;}\n}\n",
		"package p\nfunc f() { var a [10]int; var b [...]int; _ = a; _ = b }\n",
		"package p\nfunc f() {\nbadloop:\n;\nk := 0\n_ = k\nkeep:\n}\n",
		"package p\nfunc f(x interface{}) {\nswitch x {\ncase 1:\nL:\n}\nswitch typ0 := x; typ := typ0.(type) {\ndefault:\n_ = typ\n}\n}\n",
		"package p\nfunc f(c chan int) {\nselect {\ndefault:\nL:\n}\n_ = c\n}\n",
	}
	for _, src := range cases {
		t.Run(oneLine(src), func(t *testing.T) {
			_, err := Parse([]byte(src))
			require.NoError(t, err)
		})
	}
}

func TestNested(t *testing.T) {
	cases := []string{
		"package p\nfunc f() {\nif a.b() {\nreturn\n}\n}\n",
		"package p\nfunc f() {\nif a.b() {\nif y {\n}\n}\n}\n",
		"package p\nfunc f() {\nif a.b() {\nif c.d() {\n}\n}\n}\n",
		"package p\nfunc f() {\nif x {\nreturn\n}\nif y {\nif z {\n}\n}\n}\n",
		"package p\nfunc f() {\nif a.b.c() {\nreturn\n}\n}\n",
		"package p\nfunc f() {\nif a.b().c() {\nreturn\n}\n}\n",
		"package p\nfunc f() {\nx := T{A: 1, B: 2}\n}\n",
		"package p\nfunc f() {\nif x {\nreturn nil, err\n}\n}\n",
		"package p\nfunc f() {\nif x := a; x != \"\" && !c.d(y) {\nreturn nil, err\n}\n}\n",
		"package p\nfunc f() {\nif x := a; x != \"\" && !c.d(y) {\nreturn nil, err\n}\nif a.b(y) {\nreturn\n}\n}\n",
		"package p\nfunc f() {\nif x := a; x != \"\" && !c.d(y) {\nreturn nil, err\n}\nif a.b(y) {\nif c.d() {\n}\n}\n}\n",
		"package p\nfunc f() {\nif x != \"\" && !c.d(y) {\nreturn nil, err\n}\nif a.b(y) {\nif c.d() {\n}\n}\n}\n",
		"package p\nfunc f() {\nif x := a; x {\nif z {\n}\n}\nif y {\nif z {\n}\n}\n}\n",
		"package p\nfunc f() {\nif !c.d(y) {\nreturn\n}\nif a.b(y) {\nif c.d() {\n}\n}\n}\n",
		"package p\nfunc f() {\nif x != \"\" {\nreturn\n}\nif a {\nif b {\n}\n}\n}\n",
		"package p\nfunc f() {\nif x := a.b(); x != \"\" && !c.d(y) {\nreturn nil, err\n}\nif a.b(y) {\nif c.d() {\nreturn nil, err\n}\n}\n}\n",
	}
	for _, src := range cases {
		t.Run(oneLine(src), func(t *testing.T) {
			_, err := Parse([]byte(src))
			require.NoError(t, err)
		})
	}
}

func TestGoRejects(t *testing.T) {
	cases := []string{
		"package p\nfunc f() { if x == T{1} {} }\n",
		"package p\nfunc f() { if T{1} {} }\n",
		"package p\nfunc f(a int\n) {}\n",
		"package p\nfunc f() {\n if x {\n}\n else {\n}\n}\n",
		"package p\nfunc f()\n{\n}\n",
		"package p\nfunc f() { x y }\n",
		"package p\nx := 1\n",
		"package p\nfunc f() { go x }\n",
		"package p\nfunc f() { func F[T any]() {} }\n",
		"package p\nfunc f(x int) {\nswitch x {\ncase 1:\nL:\ncase 2:\n}\n}\n",
		"package p\nfunc f() {\nselect {\ndefault:\nL:\ndefault:\n}\n}\n",
	}
	for _, src := range cases {
		t.Run(oneLine(src), func(t *testing.T) {
			_, err := Parse([]byte(src))
			require.Error(t, err)
			require.ErrorIs(t, err, errs.ErrSyntax)
			_, perr := parser.ParseFile(token.NewFileSet(), "p.go", src, parser.SkipObjectResolution)
			require.Error(t, perr)
		})
	}
}

func TestTokensMatchScanner(t *testing.T) {
	src := []byte("package p\nfunc f(a int) {\n\tx := 1\n\t_ = x\n}\n")
	tree, err := Parse(src)
	require.NoError(t, err)
	got := tree.Leaves()
	want := scanGo(t, src)
	require.Equal(t, len(want), len(got), "token count")
	for i := range want {
		require.Equal(t, want[i].kind, got[i].Name, "kind %d", i)
		if want[i].lit != "" {
			require.Equal(t, want[i].lit, got[i].Text, "lit %d %s", i, want[i].kind)
		}
	}
}

type scanned struct {
	kind string
	lit  string
}

func scanGo(t *testing.T, src []byte) []scanned {
	t.Helper()
	var s scanner.Scanner
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	s.Init(file, src, func(token.Position, string) {
		t.Fatal("scan error")
	}, 0)
	var out []scanned
	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		out = append(out, scanned{kind: tok.String(), lit: lit})
	}
	return out
}

func oneLine(src string) string {
	s := strings.Join(strings.Fields(src), " ")
	if len(s) > 48 {
		return s[:48]
	}
	return s
}
