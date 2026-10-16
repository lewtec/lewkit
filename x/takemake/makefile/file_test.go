package makefile

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/stretchr/testify/require"
)

func TestComposeRoundTrip(t *testing.T) {
	t.Parallel()
	file := New()
	file.Variable("CC").Set("gcc").Exported()
	file.Variable("CFLAGS").Set("-Wall").Simple().Overridden()
	file.Variable("LDFLAGS").Immediate().Set("-Wl,-E")
	file.Variable("GEN").Shell().Set("uname -s")
	file.Variable("LIBS").Append().Set("-lm")
	file.Variable("MODE").IfUnset().Set("debug")
	file.Variable("LOCAL").On("all").Privately().Set("1")
	file.Rule("all", "lib").Prereq("$(OBJS)").Order("headers").Commands("$(CC) -o $@ $^", "touch $@")
	file.Rule("static").Static("%.o").Prereq("%.c").Double()
	file.Rule("group").Group().Prereq("a")
	file.Function("QUOTE").Set("a b").Simple().Exported()
	file.Include("config.mk").Maybe()
	file.Include("extra.mk")
	file.Export()
	file.Unexport("UNUSED")
	file.VPath("%.c", "src:inc")
	file.VPathClear()
	file.Undefine("OLD")
	cond := file.Conditional()
	arm := cond.IfEq("$(CC)", "gcc")
	arm.Variable("W").Set("-Wextra")
	arm.Rule("check").Commands("true")
	cond.IfNeq("$(CC)", "gcc").Variable("W").Set("")
	cond.Else().Variable("W").IfUnset().Set("0")
	nested := file.IfDef("DEBUG")
	nested.IfNdef("QUIET").Rule("noisy").Commands("@echo hi")

	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	second, err := Take(&buf)
	require.NoError(t, err)
	require.Equal(t, file, second)
}

func TestSyntaxRoundTrip(t *testing.T) {
	t.Parallel()
	src := "" +
		"HASH = foo \\\n" +
		"# comment \\\n" +
		"bar\n" +
		"VAR = a\\\n" +
		"b\\\n" +
		"  c\n" +
		"F = end \n" +
		"C =one\n" +
		"B=  two\n" +
		"A=\ttab\n" +
		"ESC = a\\#b\n" +
		"DBL = a\\\\#comment\n" +
		"ODD = x\\\\\\\n" +
		"y\n" +
		"TAIL = end \\\n" +
		"\n" +
		"all: lib | hdr; echo hi\n" +
		"\ttrue\n" +
		"static: %.o: %.c | dep\n" +
		"group&: a b\n" +
		"dbl:: here\n" +
		"foo: CFLAGS := -g\n" +
		"export override private NAME ?= x\n" +
		"export\n" +
		"unexport FOO\n" +
		"vpath %.c src:inc\n" +
		"vpath\n" +
		"undefine OLD\n" +
		"-include cfg\n" +
		"sinclude other\n" +
		"ifeq (a,b)\n" +
		"\t@echo in\n" +
		"else ifneq \"c d\" 'e'\n" +
		"X = 1\n" +
		"else ifdef Z\n" +
		"else\n" +
		"endif\n" +
		"define QUOTE :=\n" +
		"line \\\n" +
		"\tnext # keep\n" +
		"endef\n" +
		"M N = y\n"
	_, err := Take(strings.NewReader(src))
	require.ErrorIs(t, err, errs.ErrSyntax)
	require.ErrorIs(t, err, errSyntax)
	require.Contains(t, err.Error(), "M N = y")

	ok := strings.TrimSuffix(src, "M N = y\n")
	first := roundTrip(t, ok)
	require.Equal(t, "foo ", varValue(t, first, "HASH"))
	require.Equal(t, "a b c", varValue(t, first, "VAR"))
	require.Equal(t, "end ", varValue(t, first, "F"))
	require.Equal(t, "one", varValue(t, first, "C"))
	require.Equal(t, "two", varValue(t, first, "B"))
	require.Equal(t, "tab", varValue(t, first, "A"))
	require.Equal(t, "a#b", varValue(t, first, "ESC"))
	require.Equal(t, "a\\", varValue(t, first, "DBL"))
	require.Equal(t, "x\\ y", varValue(t, first, "ODD"))
	require.Equal(t, "end ", varValue(t, first, "TAIL"))

	all := ruleNamed(t, first, "all")
	require.Equal(t, []string{"lib"}, all.Prerequisites)
	require.Equal(t, []string{"hdr"}, all.OrderOnly)
	require.Equal(t, []string{" echo hi", "true"}, all.Recipe)

	st := ruleNamed(t, first, "static")
	require.Equal(t, "%.o", st.Pattern)
	require.Equal(t, []string{"%.c"}, st.Prerequisites)
	require.Equal(t, []string{"dep"}, st.OrderOnly)

	require.True(t, ruleNamed(t, first, "group").Grouped)
	require.True(t, ruleNamed(t, first, "dbl").DoubleColon)

	spec := targetVar(t, first, "CFLAGS")
	require.Equal(t, []string{"foo"}, spec.Targets)
	require.Equal(t, OpSimple, spec.Op)
	require.Equal(t, "-g", spec.Value)

	name := varValue(t, first, "NAME")
	require.Equal(t, "x", name)
	nv := varNamed(t, first, "NAME")
	require.True(t, nv.Export && nv.Override && nv.Private)
	require.Equal(t, OpConditional, nv.Op)

	fn := functionNamed(t, first, "QUOTE")
	require.Equal(t, OpSimple, fn.Op)
	require.Equal(t, "line next # keep", fn.Body)
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

	_, err = Take(strings.NewReader("\techo\n"))
	require.ErrorIs(t, err, errs.ErrSyntax)

	_, err = Take(strings.NewReader("ifeq (a,b)\nendif\nendif\n"))
	require.ErrorIs(t, err, errSyntax)

	_, err = Take(strings.NewReader("define FOO\n"))
	require.ErrorIs(t, err, errSyntax)

	require.Nil(t, (*File)(nil).Variable("A"))
	require.Nil(t, (*Variable)(nil).Set("x").Simple())
	require.Nil(t, (*Rule)(nil).Prereq("a").Commands("b"))
}

func TestLuaMakefile(t *testing.T) {
	t.Parallel()
	first := roundTripFile(t, "makefile")
	all := ruleNamed(t, first, "all")
	require.Equal(t, []string{"touch all"}, all.Recipe)
	require.Equal(t, []string{"$(ALL_T)"}, all.Prerequisites)
	compareMakeEquals(t, "makefile", first)
}

func TestZlibMakefile(t *testing.T) {
	t.Parallel()
	first := roundTripFile(t, "Makefile.in")
	shell := varNamed(t, first, "SHELL")
	require.Equal(t, "/bin/sh", shell.Value)
	require.Equal(t, OpRecursive, shell.Op)
	compareMakeEquals(t, "Makefile.in", first)
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

func roundTripFile(t *testing.T, name string) *File {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return roundTrip(t, string(raw))
}

func varNamed(t *testing.T, f *File, name string) *Variable {
	t.Helper()
	var found *Variable
	walk(f.Stmts, func(s Stmt) {
		v, ok := s.(*Variable)
		if ok && v.Name == name && len(v.Targets) == 0 {
			found = v
		}
	})
	require.NotNil(t, found, name)
	return found
}

func varValue(t *testing.T, f *File, name string) string {
	t.Helper()
	return varNamed(t, f, name).Value
}

func targetVar(t *testing.T, f *File, name string) *Variable {
	t.Helper()
	var found *Variable
	walk(f.Stmts, func(s Stmt) {
		v, ok := s.(*Variable)
		if ok && v.Name == name && len(v.Targets) > 0 {
			found = v
		}
	})
	require.NotNil(t, found)
	return found
}

func ruleNamed(t *testing.T, f *File, target string) *Rule {
	t.Helper()
	var found *Rule
	walk(f.Stmts, func(s Stmt) {
		r, ok := s.(*Rule)
		if !ok {
			return
		}
		for _, name := range r.Targets {
			if name == target {
				found = r
			}
		}
	})
	require.NotNil(t, found, target)
	return found
}

func functionNamed(t *testing.T, f *File, name string) *Function {
	t.Helper()
	var found *Function
	walk(f.Stmts, func(s Stmt) {
		fn, ok := s.(*Function)
		if ok && fn.Name == name {
			found = fn
		}
	})
	require.NotNil(t, found)
	return found
}

func walk(stmts []Stmt, fn func(Stmt)) {
	for _, stmt := range stmts {
		fn(stmt)
		if c, ok := stmt.(*Conditional); ok {
			for _, br := range c.Branches {
				walk(br.Body, fn)
			}
		}
	}
}

func compareMakeEquals(t *testing.T, name string, f *File) {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("testdata", name))
	require.NoError(t, err)
	cmd := exec.Command("make", "-p", "-f", path, "-q")
	out, _ := cmd.Output()
	got := makeEquals(string(out), path)
	require.NotEmpty(t, got)
	want := map[string]string{}
	for _, stmt := range f.Stmts {
		v, ok := stmt.(*Variable)
		if !ok || v.Op != OpRecursive || len(v.Targets) > 0 {
			continue
		}
		want[v.Name] = v.Value
	}
	for name, value := range want {
		have, ok := got[name]
		require.Truef(t, ok, "make -p missing %s", name)
		require.Equalf(t, have, value, "variable %s", name)
	}
}

func makeEquals(out, path string) map[string]string {
	vals := map[string]string{}
	from := ""
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "# makefile (from '") {
			from = line
			continue
		}
		if !strings.Contains(from, "'"+path+"'") {
			continue
		}
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "\t") || line == "" {
			continue
		}
		idx := strings.Index(line, " =")
		if idx <= 0 {
			continue
		}
		name := line[:idx]
		if strings.Contains(name, " ") || strings.Contains(name, ":") {
			continue
		}
		rest := line[idx+2:]
		if strings.HasPrefix(rest, " ") {
			rest = rest[1:]
		}
		vals[name] = rest
	}
	return vals
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}
