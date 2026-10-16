package ninja

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/stretchr/testify/require"
)

func TestComposeRoundTrip(t *testing.T) {
	t.Parallel()
	file := New()
	file.Variable("cflags").Set(Text("-Wall"), Text(" -Wextra"))
	file.Variable("empty").Set()
	file.Rule("cc").
		Bind("command", Ref("cxx"), Text(" -c "), Ref("in"), Text(" -o "), Ref("out")).
		Bind("depfile", Ref("out"), Text(".d"))
	file.Pool("link_pool").Bind("depth", Text("4")).Bind("note", Text("kept"))
	file.Build("cc").
		Output(Text("foo.o")).
		ImplicitOutput(Text("foo.o.imp")).
		Input(Text("foo.c")).
		Implicit(Text("headers.h")).
		OrderOnly(Text("order.h")).
		Validation(Text("check")).
		Bind("pool", Text("link_pool"))
	file.Build("phony").Output(Ref("builddir"), Text("/all"))
	file.Default([]Piece{Text("foo.o")}, []Piece{Ref("builddir"), Text("/all")})
	file.Include(Text("extras.ninja"))
	file.Subninja(Ref("root"), Text("/other.ninja"))

	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	second, err := Take(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	require.Equal(t, file, second)
}

func TestSyntaxRoundTrip(t *testing.T) {
	t.Parallel()
	src := "" +
		"# header\n" +
		"ninja_required_version = 1.5\n" +
		"\n" +
		"cflags = -Wall $\n" +
		"    -Wextra\n" +
		"space = a $ b\n" +
		"colon = a$:b\n" +
		"dollar = $$\n" +
		"tail = end \n" +
		"out_d = $out.d\n" +
		"braced = ${foo.bar}\n" +
		"crlf = 1\r\n" +
		"tabbed = a\tb\n" +
		"rule cc\n" +
		"  command = gcc $cflags -c $in -o $out\n" +
		"  # comment stays inside the block\n" +
		"  description = CC $out\n" +
		"  extra = kept\n" +
		"rule build\n" +
		"  command = echo\n" +
		"pool console\n" +
		"  depth = 1\n" +
		"  label = main\n" +
		"build foo.o | foo.imp: cc foo.c | hdr.h || gen.h |@ check\n" +
		"  pool = console\n" +
		"build space$ name: phony\n" +
		"build foo$:bar: phony\n" +
		"build | onlyimp: phony\n" +
		"build ord: phony || later\n" +
		"build $builddir/lib.a: ar a.o $\n" +
		"    b.o\n" +
		"default foo.o space$ name\n" +
		"include extras.ninja\n" +
		"subninja ${root}/nested.ninja\n"
	first := roundTrip(t, src)

	require.Equal(t, []Piece{Text("-Wall -Wextra")}, valueNamed(t, first, "cflags"))
	require.Equal(t, []Piece{Text("a  b")}, valueNamed(t, first, "space"))
	require.Equal(t, []Piece{Text("a:b")}, valueNamed(t, first, "colon"))
	require.Equal(t, []Piece{Text("$")}, valueNamed(t, first, "dollar"))
	require.Equal(t, []Piece{Text("end ")}, valueNamed(t, first, "tail"))
	require.Equal(t, []Piece{Ref("out"), Text(".d")}, valueNamed(t, first, "out_d"))
	require.Equal(t, []Piece{Ref("foo.bar")}, valueNamed(t, first, "braced"))
	require.Equal(t, []Piece{Text("a\tb")}, valueNamed(t, first, "tabbed"))

	cc := ruleNamed(t, first, "cc")
	require.Equal(t, []Binding{
		{Name: "command", Value: []Piece{
			Text("gcc "), Ref("cflags"), Text(" -c "), Ref("in"), Text(" -o "), Ref("out"),
		}},
		{Name: "description", Value: []Piece{Text("CC "), Ref("out")}},
		{Name: "extra", Value: []Piece{Text("kept")}},
	}, cc.Bindings)
	require.Equal(t, "build", ruleNamed(t, first, "build").Name)

	edge := edgeNamed(t, first, "foo.o")
	require.Equal(t, [][]Piece{{Text("foo.imp")}}, edge.ImplicitOutputs)
	require.Equal(t, "cc", edge.Rule)
	require.Equal(t, [][]Piece{{Text("foo.c")}}, edge.Inputs)
	require.Equal(t, [][]Piece{{Text("hdr.h")}}, edge.ImplicitDeps)
	require.Equal(t, [][]Piece{{Text("gen.h")}}, edge.OrderOnlyDeps)
	require.Equal(t, [][]Piece{{Text("check")}}, edge.Validations)
	require.Equal(t, []Binding{{Name: "pool", Value: []Piece{Text("console")}}}, edge.Bindings)

	require.Equal(t, [][]Piece{{Text("space name")}}, edgeNamed(t, first, "space name").Outputs)
	require.Equal(t, [][]Piece{{Text("foo:bar")}}, edgeNamed(t, first, "foo:bar").Outputs)
	only := edgeRule(t, first, "onlyimp")
	require.Nil(t, only.Outputs)
	require.Equal(t, [][]Piece{{Text("onlyimp")}}, only.ImplicitOutputs)
	found := false
	for _, s := range first.Stmts {
		e, ok := s.(*Edge)
		if !ok || e.Rule != "ar" {
			continue
		}
		found = true
		require.Equal(t, [][]Piece{{Ref("builddir"), Text("/lib.a")}}, e.Outputs)
		require.Equal(t, [][]Piece{{Text("a.o")}, {Text("b.o")}}, e.Inputs)
	}
	require.True(t, found)

	var def *Default
	var inc, sub int
	for _, s := range first.Stmts {
		switch n := s.(type) {
		case *Default:
			def = n
		case *Include:
			if n.Subninja {
				sub++
				require.Equal(t, []Piece{Ref("root"), Text("/nested.ninja")}, n.Path)
			} else {
				inc++
				require.Equal(t, []Piece{Text("extras.ninja")}, n.Path)
			}
		}
	}
	require.Equal(t, [][]Piece{{Text("foo.o")}, {Text("space name")}}, def.Targets)
	require.Equal(t, 1, inc)
	require.Equal(t, 1, sub)
}

func TestRejects(t *testing.T) {
	t.Parallel()
	_, err := Take(nil)
	require.ErrorIs(t, err, errRead)
	require.ErrorIs(t, err, errs.ErrInvalid)

	_, err = Take(errReader{})
	require.ErrorIs(t, err, errRead)

	require.ErrorIs(t, (*File)(nil).Make(io.Discard), errFile)
	require.ErrorIs(t, New().Make(nil), errWrite)

	_, err = Take(strings.NewReader("\tbuild a: phony\n"))
	require.ErrorIs(t, err, errs.ErrSyntax)
	require.ErrorIs(t, err, errSyntax)
	require.Contains(t, err.Error(), "tabs are not allowed")

	_, err = Take(strings.NewReader("x = $~\n"))
	require.ErrorIs(t, err, errSyntax)
	require.Contains(t, err.Error(), "bad $-escape")

	_, err = Take(strings.NewReader("build = 1\n"))
	require.ErrorIs(t, err, errSyntax)

	_, err = Take(strings.NewReader("x = 1"))
	require.ErrorIs(t, err, errSyntax)
	require.Contains(t, err.Error(), "unexpected EOF")

	_, err = Take(strings.NewReader("rule cc\n  command = echo\n\n  description = hi\n"))
	require.ErrorIs(t, err, errSyntax)
	require.Contains(t, err.Error(), "unexpected indent")

	bad := New()
	bad.Variable("build").Set(Text("1"))
	require.ErrorIs(t, bad.Make(io.Discard), errFile)

	bad = New()
	bad.Variable("x").Set(Piece{Text: "a", Var: "b"})
	require.ErrorIs(t, bad.Make(io.Discard), errFile)

	bad = New()
	bad.Variable("x").Set(Piece{})
	require.ErrorIs(t, bad.Make(io.Discard), errFile)

	bad = New()
	bad.Variable("x").Set(Text("a\nb"))
	require.ErrorIs(t, bad.Make(io.Discard), errFile)

	bad = New()
	bad.Build("phony").Output(Text("a|b"))
	require.ErrorIs(t, bad.Make(io.Discard), errFile)

	require.Nil(t, (*File)(nil).Variable("a"))
	require.Nil(t, (*Variable)(nil).Set(Text("x")))
	require.Nil(t, (*File)(nil).Rule("cc"))
	require.Nil(t, (*Rule)(nil).Bind("command", Text("x")))
	require.Nil(t, (*File)(nil).Build("phony"))
	require.Nil(t, (*Edge)(nil).Output(Text("a")).Input(Text("b")).Implicit(Text("c")).OrderOnly(Text("d")).Validation(Text("e")).Bind("k", Text("v")))
	require.Nil(t, (*File)(nil).Pool("p"))
	require.Nil(t, (*Pool)(nil).Bind("depth", Text("1")))
	require.Nil(t, (*File)(nil).Default([]Piece{Text("a")}))
	require.Nil(t, (*File)(nil).Include(Text("a")))
	require.Nil(t, (*File)(nil).Subninja(Text("a")))
}

func TestUpstreamNinja(t *testing.T) {
	t.Parallel()
	slow := roundTripFile(t, "long-slow-build.ninja")
	require.Equal(t, []Binding{
		{Name: "command", Value: []Piece{Text("sleep 1")}},
		{Name: "description", Value: []Piece{Text("SLEEP "), Ref("out")}},
	}, ruleNamed(t, slow, "sleep").Bindings)
	require.Equal(t, 31, countEdges(slow))
	require.Equal(t, "phony", edgeNamed(t, slow, "all").Rule)
	require.Len(t, edgeNamed(t, slow, "all").Inputs, 10)

	fds := roundTripFile(t, "inherited-fds.ninja")
	require.Equal(t, []Piece{Text("sleep 10000")}, bindingValue(t, ruleNamed(t, fds, "sleep"), "command"))
	require.Equal(t, 6, countEdges(fds))
	require.Equal(t, [][]Piece{
		{Text("a")}, {Text("b")}, {Text("c")}, {Text("d")}, {Text("e")},
	}, edgeNamed(t, fds, "all").Inputs)

	afl := roundTripFile(t, "build.ninja")
	require.Equal(t, []Piece{
		Text("clang -MMD -MF "), Ref("out"), Text(".d -o "), Ref("out"), Text(" -c "), Ref("in"),
	}, bindingValue(t, ruleNamed(t, afl, "b"), "command"))
	require.Equal(t, []Piece{Text("building "), Ref("out")}, bindingValue(t, ruleNamed(t, afl, "b"), "description"))
	require.Equal(t, "b", edgeNamed(t, afl, "a.o").Rule)
	require.Equal(t, [][]Piece{{Text("a.c")}}, edgeNamed(t, afl, "a.o").Inputs)
}

func roundTripFile(t *testing.T, name string) *File {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return roundTrip(t, string(raw))
}

func countEdges(f *File) int {
	n := 0
	for _, s := range f.Stmts {
		if _, ok := s.(*Edge); ok {
			n++
		}
	}
	return n
}

func bindingValue(t *testing.T, r *Rule, name string) []Piece {
	t.Helper()
	if r == nil {
		return nil
	}
	for _, b := range r.Bindings {
		if b.Name == name {
			return b.Value
		}
	}
	require.Failf(t, "missing binding", "%s", name)
	return nil
}

func roundTrip(t *testing.T, src string) *File {
	t.Helper()
	first, err := Take(strings.NewReader(src))
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, first.Make(&buf))
	printed := buf.String()
	second, err := Take(strings.NewReader(printed))
	require.NoError(t, err)
	require.Equal(t, first, second)
	var again bytes.Buffer
	require.NoError(t, second.Make(&again))
	require.Equal(t, printed, again.String())
	return first
}

func varNamed(t *testing.T, f *File, name string) *Variable {
	t.Helper()
	for _, s := range f.Stmts {
		v, ok := s.(*Variable)
		if ok && v.Name == name {
			return v
		}
	}
	require.Failf(t, "missing variable", "%s", name)
	return nil
}

func valueNamed(t *testing.T, f *File, name string) []Piece {
	t.Helper()
	v := varNamed(t, f, name)
	if v == nil {
		return nil
	}
	return v.Value
}

func ruleNamed(t *testing.T, f *File, name string) *Rule {
	t.Helper()
	for _, s := range f.Stmts {
		r, ok := s.(*Rule)
		if ok && r.Name == name {
			return r
		}
	}
	require.Failf(t, "missing rule", "%s", name)
	return nil
}

func edgeNamed(t *testing.T, f *File, output string) *Edge {
	t.Helper()
	for _, s := range f.Stmts {
		e, ok := s.(*Edge)
		if !ok {
			continue
		}
		for _, p := range e.Outputs {
			if pieceText(p) == output {
				return e
			}
		}
	}
	require.Failf(t, "missing edge", "%s", output)
	return nil
}

func edgeRule(t *testing.T, f *File, output string) *Edge {
	t.Helper()
	for _, s := range f.Stmts {
		e, ok := s.(*Edge)
		if !ok {
			continue
		}
		for _, p := range e.ImplicitOutputs {
			if pieceText(p) == output {
				return e
			}
		}
	}
	require.Failf(t, "missing edge", "%s", output)
	return nil
}

func pieceText(ps []Piece) string {
	var b strings.Builder
	for _, p := range ps {
		if p.Var != "" {
			b.WriteByte('$')
			b.WriteString(p.Var)
			continue
		}
		b.WriteString(p.Text)
	}
	return b.String()
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
