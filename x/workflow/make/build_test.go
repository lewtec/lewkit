package make

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	mainC = `#include "util.h"
int main(void) { return answer() == 42 ? 0 : 1; }
`
	utilC = `#include "util.h"
int answer(void) { return 42; }
`
	utilH = `int answer(void);
`
	makeFile = `
CC ?= cc
CFLAGS ?= -O2 -Wall

.PHONY: all clean

all: hello

hello: main.o util.o
	$(CC) $(LDFLAGS) -o $@ main.o util.o

%.o: %.c
	$(CC) $(CFLAGS) -c -o $@ $<

clean:
	$(RM) hello main.o util.o
`
)

func TestLinuxBuild(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not installed")
	}
	dir := t.TempDir()
	writeSources(t, dir)
	writeFile(t, dir, "Makefile", makeFile)
	require.NoError(t, buildMake(t, dir, nil))
	cmd := exec.Command(filepath.Join(dir, "hello"))
	require.NoError(t, cmd.Run())
}

func TestCondaCompiler(t *testing.T) {
	gcc, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("gcc not installed")
	}
	dir := t.TempDir()
	writeSources(t, dir)
	writeFile(t, dir, "Makefile", makeFile)
	conda := filepath.Join(dir, "conda")
	bin := filepath.Join(conda, "bin")
	require.NoError(t, os.MkdirAll(bin, 0o755))
	log := filepath.Join(dir, "conda.log")
	wrapper := filepath.Join(bin, "x86_64-conda-linux-gnu-gcc")
	script := "#!/bin/sh\nprintf '%s\\n' \"$0 $*\" >> \"" + log + "\"\nexec " + gcc + " \"$@\"\n"
	require.NoError(t, os.WriteFile(wrapper, []byte(script), 0o755))
	t.Setenv("CONDA_PREFIX", conda)
	t.Setenv("CC", wrapper)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	require.NoError(t, buildMake(t, dir, nil))
	body, err := os.ReadFile(log)
	require.NoError(t, err)
	assert.Contains(t, string(body), "x86_64-conda-linux-gnu-gcc")
	assert.Contains(t, string(body), "-c")
	assert.Contains(t, string(body), "-o")
	require.NoError(t, exec.Command(filepath.Join(dir, "hello")).Run())
	// A second build is fresh: the conda compiler is not invoked again.
	require.NoError(t, buildMake(t, dir, nil))
	again, err := os.ReadFile(log)
	require.NoError(t, err)
	assert.Equal(t, string(body), string(again))
}

func writeSources(t *testing.T, dir string) {
	t.Helper()
	writeFile(t, dir, "main.c", mainC)
	writeFile(t, dir, "util.c", utilC)
	writeFile(t, dir, "util.h", utilH)
}

func buildMake(t *testing.T, dir string, targets []string) error {
	t.Helper()
	f, err := Load(t.Context(), filepath.Join(dir, "Makefile"), dir, nil, nil)
	if err != nil {
		return err
	}
	g, err := f.Graph(t.Context(), targets)
	if err != nil {
		return err
	}
	_, err = workflow.Run(t.Context(), g, targets)
	return err
}
