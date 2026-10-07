package make

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, dir, name, body string) {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
}

func TestExpandAndCondition(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.c", "")
	writeFile(t, dir, "b.c", "")
	writeFile(t, dir, "Makefile", `
MODE =
ifeq ($(MODE),)
MODE = release
else ifeq ($(MODE),debug)
MODE = debug
else
MODE = other
endif
SRCS = $(wildcard *.c)
OBJS = $(SRCS:.c=.o)
define note
mode=$(1)
endef
all:
	echo $(MODE) $(OBJS) $(call note,ok)
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.NoError(t, err)
	g, err := f.Graph(t.Context(), nil)
	require.NoError(t, err)
	require.NotEmpty(t, g.Steps)
	var got string
	for _, step := range g.Steps {
		if step.Name == "all" {
			got = commandText(t, step)
		}
	}
	assert.Contains(t, got, "release")
	assert.Contains(t, got, "a.o")
	assert.Contains(t, got, "b.o")
	assert.Contains(t, got, "mode=ok")
}

func TestCommandLineOverridesMakefile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", `
CC = from-file
all:
	echo $(CC)
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, []string{"CC=from-cli"}, nil)
	require.NoError(t, err)
	g, err := f.Graph(t.Context(), nil)
	require.NoError(t, err)
	require.Len(t, g.Steps, 1)
	assert.Contains(t, commandText(t, g.Steps[0]), "from-cli")
}

func TestPathFunctions(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", `
all:
	echo $(dir src/foo.c)|$(notdir src/foo.c)|$(suffix src/foo.c)|$(basename src/foo.tar.gz)|$(abspath foo.c)
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.NoError(t, err)
	g, err := f.Graph(t.Context(), nil)
	require.NoError(t, err)
	got := commandText(t, g.Steps[0])
	assert.Contains(t, got, "src|foo.c|.c|src/foo.tar|")
	assert.Contains(t, got, filepath.Join(dir, "foo.c"))
	d, base := splitDF("src/foo.c")
	assert.Equal(t, "src", d)
	assert.Equal(t, "foo.c", base)
	assert.Equal(t, ".", dirOf("foo.c"))
	assert.Equal(t, "a/b", dirOf("a/b/"))
	assert.Equal(t, "", notDir("a/b/"))
	assert.Equal(t, "", suffixOf(".bashrc"))
}

func TestIfdefEmptyIsFalse(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", `
empty :=
ref = $(empty)
ifdef empty
got := bad
else
got := good
endif
ifdef ref
nest := yes
else
nest := no
endif
all:
	echo $(got):$(nest)
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.NoError(t, err)
	g, err := f.Graph(t.Context(), nil)
	require.NoError(t, err)
	assert.Contains(t, commandText(t, g.Steps[0]), "good:yes")
}

func TestMissingIncludeRemade(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", `
include gen.mk
all:
	echo $(X)
gen.mk:
	echo X=1 > gen.mk
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.NoError(t, err)
	g, err := f.Graph(t.Context(), nil)
	require.NoError(t, err)
	assert.Contains(t, commandText(t, g.Steps[0]), "1")
}

func TestMissingIncludeNoRule(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", `
include missing.mk
all:
	echo hi
`)
	_, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.ErrorIs(t, err, ErrNoRule)
	assert.Contains(t, err.Error(), "missing.mk")
}

func TestMissingIncludeRecipeFails(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", `
include missing.mk
missing.mk: dep
	echo X=1 > missing.mk
dep:
	echo fail-dep >&2
	false
all:
	echo hi
`)
	_, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing.mk")
	assert.Contains(t, err.Error(), "dep")
}

func TestPatternPrereqStaysFirst(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.c", "int main(void){return 0;}\n")
	writeFile(t, dir, "main.h", "\n")
	writeFile(t, dir, "Makefile", `
main.o: main.h
%.o: %.c
	echo $<
all: main.o
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.NoError(t, err)
	g, err := f.Graph(t.Context(), []string{"main.o"})
	require.NoError(t, err)
	assert.Contains(t, commandText(t, g.Steps[len(g.Steps)-1]), "main.c")
}

func TestWildcardKeepsDotSlash(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "crt/a.c", "int a;\n")
	writeFile(t, dir, "Makefile", `
SRCS := $(wildcard ./crt/*.c)
OBJS := $(patsubst ./%,%.o,$(basename $(SRCS)))
all:
	echo $(OBJS)
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.NoError(t, err)
	g, err := f.Graph(t.Context(), nil)
	require.NoError(t, err)
	assert.Contains(t, commandText(t, g.Steps[0]), "crt/a.o")
}

func TestEmptyTargetAssign(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", `
X :=
$(X): FOO = 1
all:
	echo ok
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.NoError(t, err)
	g, err := f.Graph(t.Context(), nil)
	require.NoError(t, err)
	assert.Contains(t, commandText(t, g.Steps[0]), "ok")
}

func TestEmptyIfElse(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", `
A :=
B := yes
all:
	echo $(if $(A),no,)
	echo $(if $(B),yes,)
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.NoError(t, err)
	g, err := f.Graph(t.Context(), nil)
	require.NoError(t, err)
	require.Len(t, g.Steps[0].Tasks, 2)
	assert.Equal(t, "echo", strings.TrimSpace(commandText(t, g.Steps[0])))
	second, ok := g.Steps[0].Tasks[1].(workflow.Command)
	require.True(t, ok)
	assert.Contains(t, second.Text, "yes")
}

func TestShellBalancedParens(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", `
try-run = $(shell if ($(1)); then echo $(2); else echo $(3); fi)
all:
	echo $(call try-run,true,yes,no)
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.NoError(t, err)
	g, err := f.Graph(t.Context(), nil)
	require.NoError(t, err)
	assert.Contains(t, commandText(t, g.Steps[0]), "yes")
}

func TestOptionalIncludeMissing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", `
-include missing.mk
all:
	echo hi
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.NoError(t, err)
	g, err := f.Graph(t.Context(), nil)
	require.NoError(t, err)
	assert.Contains(t, commandText(t, g.Steps[0]), "hi")
}

func TestTabbedAssignOutsideRule(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", `
ifeq (1,1)
	FOO := from-nested
endif
all:
	echo $(FOO)
	BAR=kept echo $$BAR
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.NoError(t, err)
	g, err := f.Graph(t.Context(), nil)
	require.NoError(t, err)
	var got strings.Builder
	for _, task := range g.Steps[0].Tasks {
		cmd, ok := task.(workflow.Command)
		require.True(t, ok)
		got.WriteString(cmd.Text)
		got.WriteByte('\n')
	}
	assert.Contains(t, got.String(), "from-nested")
	assert.Contains(t, got.String(), "BAR=kept")
}

func TestTabbedCommentOutsideRule(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", `
ifeq ($(X),y)
FOO := other
else
	# keep this assignment
	FOO := skipped
endif
all:
	echo $(FOO)
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, []string{"all"})
	require.NoError(t, err)
	g, err := f.Graph(t.Context(), []string{"all"})
	require.NoError(t, err)
	assert.Contains(t, commandText(t, g.Steps[0]), "skipped")
}

func TestQuietPrefixAfterExpand(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", `
Q = @
all:
	$(Q)echo hi
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.NoError(t, err)
	g, err := f.Graph(t.Context(), nil)
	require.NoError(t, err)
	assert.Equal(t, "echo hi", commandText(t, g.Steps[0]))
	assert.Contains(t, f.vars["MAKE"].value, "workflow make")
	assert.Contains(t, f.vars["MAKEFLAGS"].value, "--no-print-directory")
}

func TestIfFilterCommaStaysNested(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", `
empty :=
hit := __all
no := $(if $(filter __%,$(empty)),bad,good)
yes := $(if $(filter __%,$(hit)),bad,good)
$(if $(filter __%,$(empty)), \
	$(error should-not-fire))
all:
	echo $(no):$(yes)
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.NoError(t, err)
	g, err := f.Graph(t.Context(), nil)
	require.NoError(t, err)
	assert.Contains(t, commandText(t, g.Steps[0]), "good:bad")
}

func TestNestedFunctions(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", `
NODEPS := clean distclean
RESULT := $(words $(findstring foo,$(filter foo bar,foo bar baz)))
all:
	echo $(RESULT)
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.NoError(t, err)
	g, err := f.Graph(t.Context(), nil)
	require.NoError(t, err)
	assert.Contains(t, commandText(t, g.Steps[0]), "1")
}

func TestEnvCCSurvivesConditionalAssign(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CC", "/opt/conda/bin/x86_64-conda-linux-gnu-gcc")
	writeFile(t, dir, "Makefile", `
CC ?= gcc
all:
	echo $(CC)
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.NoError(t, err)
	g, err := f.Graph(t.Context(), nil)
	require.NoError(t, err)
	assert.Contains(t, commandText(t, g.Steps[0]), "x86_64-conda-linux-gnu-gcc")
}

func TestDepfileMergesHeader(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.c", "")
	writeFile(t, dir, "util.h", "")
	writeFile(t, dir, "main.d", "main.o: main.c util.h\n")
	writeFile(t, dir, "Makefile", `
-include main.d
main.o: main.c
	echo $^ > deps.txt
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.NoError(t, err)
	g, err := f.Graph(t.Context(), []string{"main.o"})
	require.NoError(t, err)
	require.Len(t, g.Steps, 1)
	assert.Contains(t, commandText(t, g.Steps[0]), "main.c")
	assert.Contains(t, commandText(t, g.Steps[0]), "util.h")
}

func TestCycle(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", `
a: b
	echo a
b: a
	echo b
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.NoError(t, err)
	_, err = f.Graph(t.Context(), []string{"a"})
	require.ErrorIs(t, err, ErrCycle)
}

func commandText(t *testing.T, step workflow.Step) string {
	t.Helper()
	require.NotEmpty(t, step.Tasks)
	cmd, ok := step.Tasks[0].(workflow.Command)
	require.True(t, ok)
	return cmd.Text
}

func TestEmptyVPATHDoesNotSwallow(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", `
VPATH :=
ifeq ($(X),)
ok := yes
endif
all:
	echo $(ok) [$(VPATH)]
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.NoError(t, err)
	g, err := f.Graph(t.Context(), nil)
	require.NoError(t, err)
	require.NotEmpty(t, g.Steps)
	got := commandText(t, g.Steps[0])
	assert.Contains(t, got, "yes")
	assert.Contains(t, got, "[]")
}

func TestComputedAssignName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", `
CONFIG_FOO := y
obj-y :=
obj-$(CONFIG_FOO) += foo.o
obj-$(CONFIG_BAR) += bar.o
all:
	echo [$(obj-y)]
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.NoError(t, err)
	g, err := f.Graph(t.Context(), nil)
	require.NoError(t, err)
	assert.Contains(t, commandText(t, g.Steps[0]), "[foo.o]")
}

func TestEmptySubstRef(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", `
dirs := foo/ bar/ baz
board-dirs := $(sort $(notdir $(dirs:/=)))
all:
	echo [$(board-dirs)][$$(dirs:/=)]
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.NoError(t, err)
	g, err := f.Graph(t.Context(), nil)
	require.NoError(t, err)
	got := commandText(t, g.Steps[0])
	assert.Contains(t, got, "[bar baz foo]")
	assert.Contains(t, got, "[$(dirs:/=)]")
}

func TestLiteralDollarPipeline(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", `
AR := ar
cmd = $$($(AR) t $@ | sed)
all:
	echo $(cmd)
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.NoError(t, err)
	g, err := f.Graph(t.Context(), nil)
	require.NoError(t, err)
	assert.Contains(t, commandText(t, g.Steps[0]), "$(ar t all | sed)")
}

func TestTargetSpecificStaysOnTarget(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", `
all: FOO = only-all
all: BAR := plain
all:
	echo A:$(FOO):$(BAR)
other:
	echo B:$(FOO):$(BAR)
`)
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	require.NoError(t, err)
	g, err := f.Graph(t.Context(), []string{"all", "other"})
	require.NoError(t, err)
	var all, other string
	for _, step := range g.Steps {
		switch step.Name {
		case "all":
			all = commandText(t, step)
		case "other":
			other = commandText(t, step)
		}
	}
	assert.Contains(t, all, "A:only-all:plain")
	assert.Contains(t, other, "B::")
}

func TestParseArgs(t *testing.T) {
	assigns, targets := ParseArgs([]string{"CC=gcc", "all", "CFLAGS=-O2"})
	assert.Equal(t, []string{"CC=gcc", "CFLAGS=-O2"}, assigns)
	assert.Equal(t, []string{"all"}, targets)
	assert.True(t, strings.Contains(strings.Join(assigns, " "), "CC"))
}

func TestLoadRejectsNilContext(t *testing.T) {
	_, err := Load(nil, "Makefile", "", nil, nil)
	require.Error(t, err)
}

func TestShellStopsWhenContextCancels(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", "X := $(shell sleep 30)\nall:\n\techo $(X)\n")
	ctx, cancel := context.WithCancel(t.Context())
	errCh := make(chan error, 1)
	go func() {
		_, err := Load(ctx, filepath.Join(dir, "Makefile"), dir, nil, nil)
		errCh <- err
	}()
	time.Sleep(150 * time.Millisecond)
	cancel()
	select {
	case err := <-errCh:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(3 * time.Second):
		t.Fatal("$(shell) kept running after cancel")
	}
}
