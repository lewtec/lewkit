package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/test"
	gotool "github.com/lewtec/lewkit/x/toolchain/go"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if handled, err := gotool.Child(); handled {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestReleaseGoBuild(t *testing.T) {
	setToolchainEnv(t)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/m\n\ngo 1.27\n"), 0o644))
	mainSrc := "package main\n\nimport \"example.com/m/p\"\n\nfunc main() { println(p.Value()) }\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainSrc), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "p"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "p", "p.go"), []byte("package p\n\nfunc Value() int { return 7 }\n"), 0o644))
	// release go reads archives from -pkgdir. The fixture copies the
	// installed runtime out of the build cache; the command does not run go.
	pkgdir := filepath.Join(dir, "std")
	fillRuntime(t, dir, pkgdir)
	out := filepath.Join(dir, "hello")
	args := normalizeReleaseGoFlags([]string{
		"release", "go", "build", ".",
		"-o", out,
		"-C", dir,
		"-ldflags", "-s -w",
		"-pkgdir", pkgdir,
	})
	app := cmd.ParseOK[cmd.App[root]](t, args...)
	require.NoError(t, app.Args.release.gobuild.build.Run(t.Context()))
	got, err := exec.Command(out).CombinedOutput()
	require.NoErrorf(t, err, "run: %v\n%s", err, got)
	if string(got) != "7\n" {
		require.FailNowf(t, "", "output %q", got)
	}
}

func setToolchainEnv(t *testing.T) {
	t.Helper()
	goBin, err := exec.LookPath("go")
	if err != nil {
		goBin = "/home/grokbox/.local/share/mise/installs/go/1.27.0/bin/go"
	}
	t.Setenv("PATH", filepath.Dir(goBin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, pair := range [][2]string{
		{"GOCACHE", filepath.Join(os.TempDir(), "lewkit-toolchain-gocache")},
		{"GOMODCACHE", filepath.Join(os.TempDir(), "lewkit-toolchain-gomod")},
		{"GOPATH", filepath.Join(os.TempDir(), "lewkit-toolchain-gopath")},
		{"HOME", filepath.Join(os.TempDir(), "lewkit-toolchain-home")},
	} {
		require.NoError(t, os.MkdirAll(pair[1], 0o755))
		t.Setenv(pair[0], pair[1])
	}
	t.Setenv("GOTOOLCHAIN", "local")
	t.Setenv("CGO_ENABLED", "0")
	t.Setenv("GO111MODULE", "on")
	t.Setenv("GOFLAGS", "")
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "safe.directory")
	t.Setenv("GIT_CONFIG_VALUE_0", "*")
}

func TestReleaseGoLewkit(t *testing.T) {
	if testing.Short() {
		test.Skip(t, "builds lewkit with the embedded compiler")
	}
	setToolchainEnv(t)
	modRoot := moduleRoot(t)
	dl := exec.Command("go", "mod", "download")
	dl.Dir = modRoot
	if out, err := dl.CombinedOutput(); err != nil {
		require.FailNowf(t, "", "go mod download: %v\n%s", err, out)
	}
	pkgdir := filepath.Join(t.TempDir(), "pkg")
	fillPrebuilt(t, modRoot, pkgdir)
	out := filepath.Join(t.TempDir(), "lewkit")
	args := normalizeReleaseGoFlags([]string{
		"release", "go", "build", "./cmd/lewkit",
		"-o", out,
		"-C", modRoot,
		"-pkgdir", pkgdir,
	})
	app := cmd.ParseOK[cmd.App[root]](t, args...)
	require.NoError(t, app.Args.release.gobuild.build.Run(t.Context()))
	helloDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(helloDir, "go.mod"), []byte("module example.com/hello\n\ngo 1.27\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(helloDir, "hello.go"), []byte("package main\n\nfunc main() { println(\"from-lewkit\") }\n"), 0o644))
	hello := filepath.Join(helloDir, "hello")
	got, err := exec.Command(out, "--sentry-dsn=", "release", "go", "build", "-C", helloDir, "-o", hello, "-pkgdir", pkgdir, ".").CombinedOutput()
	require.NoErrorf(t, err, "built lewkit release go: %v\n%s", err, got)
	got, err = exec.Command(hello).CombinedOutput()
	require.NoErrorf(t, err, "run built hello: %v\n%s", err, got)
	if string(got) != "from-lewkit\n" {
		require.FailNowf(t, "", "output %q", got)
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		require.FailNow(t, fmt.Sprint("caller"))
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			require.FailNow(t, fmt.Sprint("go.mod"))
		}
		dir = parent
	}
}

func fillPrebuilt(t *testing.T, module, pkgdir string) {
	t.Helper()
	// The fixture asks the installed go command for archives this toolchain
	// cannot produce: the standard library, and packages with assembly.
	cmd := exec.Command("go", "list", "-buildvcs=false", "-deps", "-export", "-f",
		"{{if and .Export (or .Standard (or .SFiles .SysoFiles))}}{{.ImportPath}} {{.Export}}{{end}}",
		"./cmd/lewkit")
	cmd.Dir = module
	out, err := cmd.Output()
	if err != nil {
		stderr := ""
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = string(ee.Stderr)
		}
		require.FailNowf(t, "", "go list: %v\n%s", err, stderr)
	}
	require.NoError(t, os.MkdirAll(pkgdir, 0o755))
	n := 0
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		imp, src, ok := strings.Cut(line, " ")
		if !ok {
			require.FailNowf(t, "", "go list line %q", line)
		}
		dst := filepath.Join(pkgdir, filepath.FromSlash(imp)+".a")
		require.NoError(t, os.MkdirAll(filepath.Dir(dst), 0o755))
		if err := os.Link(src, dst); err != nil {
			require.NoError(t, copyFile(src, dst))
		}
		n++
	}
	if n == 0 {
		require.FailNow(t, fmt.Sprint("no prebuilt archives"))
	}
}

func fillRuntime(t *testing.T, module, pkgdir string) {
	t.Helper()
	cmd := exec.Command("go", "list", "-deps", "-export", "-f", "{{if .Export}}{{.ImportPath}} {{.Export}}{{end}}", "runtime")
	cmd.Dir = module
	out, err := cmd.Output()
	if err != nil {
		stderr := ""
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = string(ee.Stderr)
		}
		require.FailNowf(t, "", "go list runtime: %v\n%s", err, stderr)
	}
	require.NoError(t, os.MkdirAll(pkgdir, 0o755))
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		imp, src, ok := strings.Cut(line, " ")
		if !ok {
			require.FailNowf(t, "", "go list line %q", line)
		}
		dst := filepath.Join(pkgdir, filepath.FromSlash(imp)+".a")
		require.NoError(t, os.MkdirAll(filepath.Dir(dst), 0o755))
		require.NoError(t, copyFile(src, dst))
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
