package load

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoadModule(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/m\n\ngo 1.27\n")
	write("main.go", "package main\n\nimport \"example.com/m/p\"\n\nfunc main() { println(p.Value()) }\n")
	write("p/p.go", "package p\n\nfunc Value() int { return 7 }\n")
	write("p/skip.go", "//go:build extra\n\npackage p\n")
	write("p/p_windows.go", "package p\n")

	g, err := Load(dir, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if g.Module != "example.com/m" || len(g.Pkgs) != 2 {
		t.Fatalf("graph %+v", g)
	}
	p := g.Pkgs[0]
	hostFiles := 1
	if runtime.GOOS == "windows" {
		hostFiles = 2
	}
	if p.ImportPath != "example.com/m/p" || p.Name != "p" || len(p.Files) != hostFiles {
		t.Fatalf("p %+v", p)
	}
	main := g.Pkgs[1]
	if main.Name != "main" || len(main.Deps) != 1 || main.Deps[0] != "example.com/m/p" {
		t.Fatalf("main %+v", main)
	}

	win, err := LoadTarget(dir, []string{"./p"}, nil, "windows", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if len(win.Pkgs) != 1 || len(win.Pkgs[0].Files) != 2 {
		t.Fatalf("windows %+v", win.Pkgs)
	}

	tagged, err := Load(dir, []string{"./p"}, []string{"extra"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tagged.Pkgs) != 1 || len(tagged.Pkgs[0].Files) != hostFiles+1 {
		t.Fatalf("tagged %+v", tagged.Pkgs)
	}

	walked, err := Load(dir, []string{"./..."}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(walked.Pkgs) != 2 {
		t.Fatalf("walk %+v", walked.Pkgs)
	}

	if _, err := Load(dir, []string{"example.com/other"}, nil); !errors.Is(err, ErrExternal) {
		t.Fatalf("external: %v", err)
	}
}

func TestLoadCgo(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/c\n\ngo 1.27\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	body := "package c\n\nimport \"C\"\n"
	if err := os.WriteFile(filepath.Join(dir, "c.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir, nil, nil); !errors.Is(err, ErrCgo) {
		t.Fatalf("cgo: %v", err)
	}
}

func TestLoadCache(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("GOMODCACHE", cache)
	write := func(name, body string) {
		t.Helper()
		path := filepath.Join(cache, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("example.com/dep@v1.2.3/go.mod", "module example.com/dep\n\ngo 1.22.0\n")
	write("example.com/dep@v1.2.3/p.go", "package dep\n\nfunc V() int { return 1 }\n")
	write("example.com/dep@v1.2.3/p.s", "")
	write("example.com/dep/sub@v1.0.0/go.mod", "module example.com/dep/sub\n\ngo 1.21\n")
	write("example.com/dep/sub@v1.0.0/q.go", "package sub\n\nfunc Q() int { return 2 }\n")

	dir := t.TempDir()
	mod := "module example.com/m\n\ngo 1.27.0\n\nrequire (\n\texample.com/dep v1.2.3\n\texample.com/dep/sub v1.0.0\n)\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o644); err != nil {
		t.Fatal(err)
	}
	src := "package main\n\nimport (\n\t\"example.com/dep\"\n\t\"example.com/dep/sub\"\n)\n\nfunc main() { println(dep.V(), sub.Q()) }\n"
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	g, err := Load(dir, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Pkgs) != 3 {
		t.Fatalf("pkgs %d", len(g.Pkgs))
	}
	dep := g.Pkgs[0]
	if dep.ImportPath != "example.com/dep" || !dep.Asm || dep.Lang != "go1.22" || len(dep.AsmFiles) != 1 || !strings.HasSuffix(dep.AsmFiles[0], "p.s") {
		t.Fatalf("dep %+v", dep)
	}
	sub := g.Pkgs[1]
	if sub.ImportPath != "example.com/dep/sub" || sub.Asm || sub.Lang != "go1.21" {
		t.Fatalf("sub %+v", sub)
	}
	if g.Pkgs[2].Lang != "go1.27" || len(g.Pkgs[2].Deps) != 2 {
		t.Fatalf("main %+v", g.Pkgs[2])
	}

	missing := "module example.com/m\n\ngo 1.27\n\nrequire example.com/nope v0.1.0\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(missing), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nimport _ \"example.com/nope\"\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir, nil, nil); !errors.Is(err, ErrMissing) {
		t.Fatalf("missing: %v", err)
	}
}

func TestLoadReplace(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/m\n\ngo 1.27\n\nrequire example.com/dep v1.2.3\n\nreplace example.com/dep => ./local\n")
	write("main.go", "package main\n\nimport \"example.com/dep\"\n\nfunc main() { println(dep.V()) }\n")
	write("local/go.mod", "module example.com/dep\n\ngo 1.20\n")
	write("local/p.go", "package dep\n\nfunc V() int { return 3 }\n")
	g, err := Load(dir, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Pkgs) != 2 || g.Pkgs[0].Asm || g.Pkgs[0].Lang != "go1.20" {
		t.Fatalf("replace %+v", g.Pkgs)
	}
}

func TestLoadEmbed(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/m\n\ngo 1.27\n")
	write("p.go", "package p\n\nimport _ \"embed\"\n\n//go:embed msg.txt\nvar msg string\n\n//go:embed d\nvar files embed.FS\n")
	write("msg.txt", "hi\n")
	write("d/a.txt", "a\n")
	write("d/.hidden", "no\n")
	g, err := Load(dir, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Pkgs) != 1 || g.Pkgs[0].Embed == nil {
		t.Fatalf("embed %+v", g.Pkgs)
	}
	emb := g.Pkgs[0].Embed
	if len(emb.Patterns["msg.txt"]) != 1 || emb.Patterns["msg.txt"][0] != "msg.txt" {
		t.Fatalf("msg %#v", emb.Patterns)
	}
	got := emb.Patterns["d"]
	if len(got) != 1 || got[0] != "d/a.txt" {
		t.Fatalf("dir %#v", emb.Patterns)
	}
	if _, ok := emb.Files["d/a.txt"]; !ok {
		t.Fatalf("files %#v", emb.Files)
	}
}

func TestLoadEmbedAlias(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/m\n\ngo 1.27\n")
	write("p.go", "package p\n\nimport _ \"embed\"\n\n//go:embed msg.txt\nvar msg string\n")
	write("msg.txt", "hi\n")
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(dir, alias); err != nil {
		t.Fatal(err)
	}
	g, err := Load(alias, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	emb := g.Pkgs[0].Embed
	if emb == nil || len(emb.Patterns["msg.txt"]) != 1 || emb.Patterns["msg.txt"][0] != "msg.txt" {
		t.Fatalf("alias %#v", emb)
	}
}

func TestSplitFlags(t *testing.T) {
	got, err := SplitFlags(`-s -w -X main.v=one`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-s", "-w", "-X", "main.v=one"}
	if len(got) != len(want) {
		t.Fatalf("flags %q", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("flags %q", got)
		}
	}
	quoted, err := SplitFlags(`-E cpuinit -T 0x10010000`)
	if err != nil || len(quoted) != 4 || quoted[1] != "cpuinit" {
		t.Fatalf("quoted %q %v", quoted, err)
	}
	if _, err := SplitFlags(`-X "main.v`); err == nil {
		t.Fatal("unterminated quote")
	}
}
