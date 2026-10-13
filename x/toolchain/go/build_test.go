package gotool

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/workflow"
)

func TestBuildRejectsShape(t *testing.T) {
	if !errors.Is(Build(nil, nil, nil), ErrNilContext) {
		t.Fatal("nil context")
	}
	if !errors.Is(Build(t.Context(), nil, nil), ErrNoWork) {
		t.Fatal("no work")
	}
	empty := []Unit{{ImportPath: "a"}}
	if !errors.Is(Build(t.Context(), empty, nil), ErrEmptyUnit) {
		t.Fatal("empty unit")
	}
	dup := []Unit{
		{ImportPath: "a", Files: []string{"a.go"}, Archive: "a.a", ImportCfg: "a.cfg"},
		{ImportPath: "a", Files: []string{"a.go"}, Archive: "a.a", ImportCfg: "a.cfg"},
	}
	if !errors.Is(Build(t.Context(), dup, nil), ErrDuplicate) {
		t.Fatal("duplicate")
	}
	missing := []Unit{{
		ImportPath: "a",
		Files:      []string{"a.go"},
		Archive:    "a.a",
		ImportCfg:  "a.cfg",
		Deps:       []string{"missing"},
	}}
	if !errors.Is(Build(t.Context(), missing, nil), ErrUnknownDep) {
		t.Fatal("unknown dep")
	}
	cycle := []Unit{
		{ImportPath: "a", Files: []string{"a.go"}, Archive: "a.a", ImportCfg: "a.cfg", Deps: []string{"b"}},
		{ImportPath: "b", Files: []string{"b.go"}, Archive: "b.a", ImportCfg: "b.cfg", Deps: []string{"a"}},
	}
	if !errors.Is(Build(t.Context(), cycle, nil), workflow.ErrCycle) {
		t.Fatal("cycle")
	}
	if !errors.Is(Build(t.Context(), nil, &Image{}), ErrEmptyImage) {
		t.Fatal("empty link")
	}
}

func TestBuildDependentPackages(t *testing.T) {
	work := t.TempDir()
	cfg, _ := listRuntime(t, work)
	runtimeCfg, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	pkg := filepath.Join(work, "p.go")
	if err := os.WriteFile(pkg, []byte("package p\n\nfunc Value() int { return 7 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mainSrc := filepath.Join(work, "main.go")
	body := "package main\n\nimport \"example.com/p\"\n\nfunc main() { println(p.Value()) }\n"
	if err := os.WriteFile(mainSrc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	pArchive := filepath.Join(work, "p.a")
	mainArchive := filepath.Join(work, "main.a")
	mainCfg := filepath.Join(work, "main.importcfg")
	mainBody := string(runtimeCfg) + "packagefile example.com/p=" + pArchive + "\n"
	if err := os.WriteFile(mainCfg, []byte(mainBody), 0o644); err != nil {
		t.Fatal(err)
	}
	deps, err := os.ReadFile(filepath.Join(work, "deps"))
	if err != nil {
		t.Fatal(err)
	}
	linkCfg := filepath.Join(work, "link.importcfg")
	linkBody := "# import config\npackagefile main=" + mainArchive + "\npackagefile example.com/p=" + pArchive + "\n" + string(deps)
	if err := os.WriteFile(linkCfg, []byte(linkBody), 0o644); err != nil {
		t.Fatal(err)
	}
	mode, shared := hostImage()
	bin := hostBin(filepath.Join(work, "hello"))
	units := []Unit{
		{ImportPath: "example.com/p", Files: []string{pkg}, ImportCfg: cfg, Archive: pArchive, Shared: shared},
		{ImportPath: "main", Files: []string{mainSrc}, Deps: []string{"example.com/p"}, ImportCfg: mainCfg, Archive: mainArchive, Shared: shared},
	}
	image := &Image{
		Out: bin, ImportCfg: linkCfg, Mode: mode, Archive: mainArchive, Dep: "main",
		Ldflags: []string{"-s", "-w"},
	}
	if err := Build(t.Context(), units, image); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(bin).CombinedOutput()
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if string(out) != "7\n" {
		t.Fatalf("output %q", out)
	}
}
