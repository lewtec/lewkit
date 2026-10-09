package ninja

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariablesAndEdges(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "build.ninja", `
builddir = out
cc = gcc
cflags = -O2
rule cc
  command = $cc $cflags -c $in -o $out
  description = CC $out
rule link
  command = $cc $in -o $out
  description = LINK $out
build $builddir/main.o: cc main.c
build $builddir/hello: link $builddir/main.o
default $builddir/hello
`)
	f, err := Load(t.Context(), filepath.Join(dir, "build.ninja"), dir)
	require.NoError(t, err)
	g, err := f.Graph(nil)
	require.NoError(t, err)
	require.Equal(t, []string{"out/hello"}, g.Defaults)
	byName := map[string]workflow.Step{}
	for _, step := range g.Steps {
		byName[step.Name] = step
	}
	cc, ok := byName["out/main.o"]
	require.True(t, ok)
	cmd, ok := cc.Tasks[0].(workflow.Command)
	require.True(t, ok)
	assert.Equal(t, "gcc -O2 -c main.c -o out/main.o", cmd.Text)
	assert.Equal(t, "CC out/main.o", cc.Desc)
	assert.Equal(t, []string{"out/main.o"}, byName["out/hello"].Deps)
}

func TestHashInCommand(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "build.ninja", `
# comment
rule gen
  command = echo "#include \"tool.h\"" > $out
build tool.c: gen
`)
	f, err := Load(t.Context(), filepath.Join(dir, "build.ninja"), dir)
	require.NoError(t, err)
	g, err := f.Graph(nil)
	require.NoError(t, err)
	cmd, ok := g.Steps[0].Tasks[0].(workflow.Command)
	require.True(t, ok)
	assert.Contains(t, cmd.Text, `#include`)
}

func TestImplicitOutputs(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "build.ninja", `
rule cc
  command = gcc -c $in -o $out
build main.o | alias.o: cc main.c
`)
	f, err := Load(t.Context(), filepath.Join(dir, "build.ninja"), dir)
	require.NoError(t, err)
	g, err := f.Graph(nil)
	require.NoError(t, err)
	require.Equal(t, "main.o", g.Alias["alias.o"])
	require.Equal(t, []string{"main.o"}, g.Defaults)
}

func TestDepfileInputs(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "main.o.d", "main.o: main.c util.h\n")
	write(t, dir, "build.ninja", `
rule cc
  command = gcc -c $in -o $out
  depfile = $out.d
  description = CC $out
build main.o: cc main.c
`)
	f, err := Load(t.Context(), filepath.Join(dir, "build.ninja"), dir)
	require.NoError(t, err)
	g, err := f.Graph(nil)
	require.NoError(t, err)
	require.Len(t, g.Steps, 1)
	assert.Contains(t, g.Steps[0].Inputs, "util.h")
}

func TestLinuxBuild(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not installed")
	}
	dir := t.TempDir()
	writeSources(t, dir)
	write(t, dir, "build.ninja", linuxNinja)
	require.NoError(t, build(t, dir, nil))
	require.NoError(t, exec.Command(filepath.Join(dir, "hello")).Run())
}

func TestCondaCompilerOnPath(t *testing.T) {
	gcc, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("gcc not installed")
	}
	dir := t.TempDir()
	writeSources(t, dir)
	write(t, dir, "build.ninja", linuxNinja)
	conda := filepath.Join(dir, "conda")
	bin := filepath.Join(conda, "bin")
	require.NoError(t, os.MkdirAll(bin, 0o755))
	log := filepath.Join(dir, "conda.log")
	script := "#!/bin/sh\nprintf '%s\\n' \"$0 $*\" >> \"" + log + "\"\nexec " + gcc + " \"$@\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(bin, "gcc"), []byte(script), 0o755))
	t.Setenv("CONDA_PREFIX", conda)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	require.NoError(t, build(t, dir, nil))
	body, err := os.ReadFile(log)
	require.NoError(t, err)
	assert.Contains(t, string(body), "gcc")
	assert.Contains(t, string(body), "-c")
	require.NoError(t, exec.Command(filepath.Join(dir, "hello")).Run())
	require.NoError(t, build(t, dir, nil))
	again, err := os.ReadFile(log)
	require.NoError(t, err)
	assert.Equal(t, string(body), string(again))
}

const linuxNinja = `
cc = gcc
cflags = -O2 -Wall
rule cc
  command = $cc $cflags -c $in -o $out
  description = CC $out
rule link
  command = $cc $in -o $out
  description = LINK $out
build main.o: cc main.c
build util.o: cc util.c
build hello: link main.o util.o
default hello
`

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644))
}

func writeSources(t *testing.T, dir string) {
	t.Helper()
	write(t, dir, "main.c", "#include \"util.h\"\nint main(void) { return answer() == 42 ? 0 : 1; }\n")
	write(t, dir, "util.c", "#include \"util.h\"\nint answer(void) { return 42; }\n")
	write(t, dir, "util.h", "int answer(void);\n")
}

func TestLoadRejectsNilContext(t *testing.T) {
	_, err := Load(nil, "build.ninja", "")
	require.Error(t, err)
}

func build(t *testing.T, dir string, targets []string) error {
	t.Helper()
	f, err := Load(t.Context(), filepath.Join(dir, "build.ninja"), dir)
	if err != nil {
		return err
	}
	g, err := f.Graph(targets)
	if err != nil {
		return err
	}
	_, err = workflow.Run(t.Context(), g, targets)
	return err
}
