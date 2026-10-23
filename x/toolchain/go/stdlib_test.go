package gotool

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestAsmObjectHeader(t *testing.T) {
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		test.Skip(t, "assembler covers amd64 and arm64")
	}
	dir := t.TempDir()
	body := "#include \"textflag.h\"\n\nTEXT ·add(SB),NOSPLIT,$0\n\tMOVQ $1, AX\n\tRET\n"
	if runtime.GOARCH == "arm64" {
		body = "#include \"textflag.h\"\n\nTEXT ·add(SB),NOSPLIT,$0\n\tMOVD $1, R0\n\tRET\n"
	}
	src := filepath.Join(dir, "add.s")
	require.NoError(t, os.WriteFile(src, []byte(body), 0o644))
	root, err := gorootDir()
	require.NoError(t, err)
	out := filepath.Join(dir, "add.o")
	require.NoError(t, spawn(t.Context(), "asm", childRequest{
		ImportPath: "p",
		Out:        out,
		Files:      []string{src},
		Includes:   []string{filepath.Join(root, "pkg", "include")},
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
	}))
	obj, err := os.ReadFile(out)
	require.NoError(t, err)
	line, _, _ := strings.Cut(string(obj), "\n")
	require.Contains(t, line, "go object")
	require.Contains(t, line, "go1.27.0")
}

func TestBuildModuleWithoutHostGo(t *testing.T) {
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		test.Skip(t, "assembler covers amd64 and arm64")
	}
	if runtime.GOOS == "windows" {
		test.Skip(t, "host go stub is a shell script")
	}
	dir := t.TempDir()
	mod := filepath.Join(dir, "tiny")
	require.NoError(t, os.MkdirAll(mod, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(mod, "go.mod"), []byte("module example.com/tiny\n\ngo 1.27.0\n"), 0o644))
	src := "package main\n\nimport (\n\t\"fmt\"\n\t\"net\"\n)\n\nfunc add(a, b int) int\n\nfunc main() {\n\tfmt.Println(add(2, 3))\n\tfmt.Println(net.JoinHostPort(\"127.0.0.1\", \"80\"))\n}\n"
	require.NoError(t, os.WriteFile(filepath.Join(mod, "main.go"), []byte(src), 0o644))
	amd64 := "#include \"textflag.h\"\n\nTEXT ·add(SB),NOSPLIT,$0-24\n\tMOVQ a+0(FP), AX\n\tADDQ b+8(FP), AX\n\tMOVQ AX, ret+16(FP)\n\tRET\n"
	arm64 := "#include \"textflag.h\"\n\nTEXT ·add(SB),NOSPLIT,$0-24\n\tMOVD a+0(FP), R0\n\tMOVD b+8(FP), R1\n\tADD R1, R0, R0\n\tMOVD R0, ret+16(FP)\n\tRET\n"
	require.NoError(t, os.WriteFile(filepath.Join(mod, "add_amd64.s"), []byte(amd64), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(mod, "add_arm64.s"), []byte(arm64), 0o644))

	stub := filepath.Join(dir, "bin")
	require.NoError(t, os.MkdirAll(stub, 0o755))
	script := "#!/bin/sh\necho host go should not be called >&2\nexit 99\n"
	require.NoError(t, os.WriteFile(filepath.Join(stub, "go"), []byte(script), 0o755))
	t.Setenv("PATH", stub+string(os.PathListSeparator)+os.Getenv("PATH"))

	bin := filepath.Join(dir, "tinybin")
	require.NoError(t, BuildModule(t.Context(), Module{Dir: mod, Out: bin}))
	out, err := exec.Command(bin).CombinedOutput()
	require.NoError(t, err, string(out))
	require.Equal(t, "5\n127.0.0.1:80\n", string(out))
}
