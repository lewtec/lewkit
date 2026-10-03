package glsl

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashSeparatesStage(t *testing.T) {
	src := []byte("void main() {}\n")
	require.Equal(t, Hash(StageVertex, src), Hash(StageVertex, src))
	require.NotEqual(t, Hash(StageVertex, src), Hash(StageFragment, src))
	require.NotEqual(t, Hash(StageVertex, src), Hash(StageVertex, []byte("other")))
}

func TestRegisterHash(t *testing.T) {
	src := []byte("registry-same")
	sum := Hash(StageCompute, src)
	spirv := sampleSPIRV(0x11)
	require.NoError(t, RegisterHash(StageCompute, sum, spirv))
	require.NoError(t, RegisterHash(StageCompute, sum, spirv))
	require.ErrorIs(t, RegisterHash(StageCompute, sum, sampleSPIRV(0x22)), ErrExist)
	require.ErrorIs(t, RegisterHash(9, sum, spirv), ErrCompile)
	require.ErrorIs(t, RegisterHash(StageCompute, sum, []byte{0x03, 0x02, 0x23, 0x07}), ErrCompile)

	got, ok := Lookup(StageCompute, src)
	require.True(t, ok)
	require.Equal(t, spirv, got)
	got[4] = 0xff
	again, ok := Lookup(StageCompute, src)
	require.True(t, ok)
	require.Equal(t, spirv, again)
	_, ok = Lookup(StageVertex, src)
	require.False(t, ok)
}

func TestCompileStageUsesRegistry(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("glslc stand-in is a shell script")
	}
	src := []byte("registry-beats-glslc")
	want := sampleSPIRV(0x31)
	require.NoError(t, RegisterHash(StageVertex, Hash(StageVertex, src), want))
	dir := t.TempDir()
	writeGLSLC(t, dir, sampleSPIRV(0x32), 0)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	mark := filepath.Join(dir, "mark")
	t.Setenv("GLSLC_MARK", mark)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	got, err := CompileStage(ctx, StageVertex, src)
	require.NoError(t, err)
	require.Equal(t, want, got)
	_, err = os.Stat(mark)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestCompileStageRunsGLSLC(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("glslc stand-in is a shell script")
	}
	want := sampleSPIRV(0x41)
	dir := t.TempDir()
	writeGLSLC(t, dir, want, 0)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	got, err := CompileStage(t.Context(), StageFragment, []byte("glslc-stand-in"))
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestMustRegisterHashPanics(t *testing.T) {
	require.Panics(t, func() {
		MustRegisterHash(StageVertex, Hash(StageVertex, []byte("bad")), []byte{1, 2, 3, 4})
	})
}

func sampleSPIRV(mark byte) []byte {
	spv := []byte{
		0x03, 0x02, 0x23, 0x07,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
	spv[4] = mark
	return spv
}

func writeGLSLC(t *testing.T, dir string, spirv []byte, exitCode int) {
	t.Helper()
	var body bytes.Buffer
	body.WriteString("#!/bin/sh\n")
	body.WriteString("if [ -n \"$GLSLC_MARK\" ]; then echo ran >> \"$GLSLC_MARK\"; fi\n")
	body.WriteString("out=\n")
	body.WriteString("while [ $# -gt 0 ]; do\n")
	body.WriteString("  if [ \"$1\" = \"-o\" ]; then out=$2; shift 2; continue; fi\n")
	body.WriteString("  shift\ndone\n")
	if exitCode != 0 {
		body.WriteString("exit 1\n")
	} else {
		body.WriteString("if [ -z \"$out\" ]; then echo missing -o >&2; exit 1; fi\n")
		body.WriteString("printf '")
		for _, b := range spirv {
			body.WriteString("\\")
			body.WriteByte('0' + (b >> 6))
			body.WriteByte('0' + ((b >> 3) & 7))
			body.WriteByte('0' + (b & 7))
		}
		body.WriteString("' > \"$out\"\n")
	}
	path := filepath.Join(dir, "glslc")
	require.NoError(t, os.WriteFile(path, body.Bytes(), 0o755))
}
