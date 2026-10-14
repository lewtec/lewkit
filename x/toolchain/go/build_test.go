package gotool

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/workflow"
	"github.com/stretchr/testify/require"
)

func TestBuildRejectsShape(t *testing.T) {
	require.ErrorIs(t, Build(t.Context(), nil, nil), ErrNoWork)
	empty := []Unit{{ImportPath: "a"}}
	require.ErrorIs(t, Build(t.Context(), empty, nil), ErrEmptyUnit)
	dup := []Unit{
		{ImportPath: "a", Files: []string{"a.go"}, Archive: "a.a", ImportCfg: "a.cfg"},
		{ImportPath: "a", Files: []string{"a.go"}, Archive: "a.a", ImportCfg: "a.cfg"},
	}
	require.ErrorIs(t, Build(t.Context(), dup, nil), ErrDuplicate)
	missing := []Unit{{
		ImportPath: "a",
		Files:      []string{"a.go"},
		Archive:    "a.a",
		ImportCfg:  "a.cfg",
		Deps:       []string{"missing"},
	}}
	require.ErrorIs(t, Build(t.Context(), missing, nil), ErrUnknownDep)
	cycle := []Unit{
		{ImportPath: "a", Files: []string{"a.go"}, Archive: "a.a", ImportCfg: "a.cfg", Deps: []string{"b"}},
		{ImportPath: "b", Files: []string{"b.go"}, Archive: "b.a", ImportCfg: "b.cfg", Deps: []string{"a"}},
	}
	require.ErrorIs(t, Build(t.Context(), cycle, nil), workflow.ErrCycle)
	require.ErrorIs(t, Build(t.Context(), nil, &Image{}), ErrEmptyImage)
}

func TestBuildDependentPackages(t *testing.T) {
	work := t.TempDir()
	cfg, _ := listRuntime(t, work)
	runtimeCfg, err := os.ReadFile(cfg)
	require.NoError(t, err)
	pkg := filepath.Join(work, "p.go")
	require.NoError(t, os.WriteFile(pkg, []byte("package p\n\nfunc Value() int { return 7 }\n"), 0o644))
	mainSrc := filepath.Join(work, "main.go")
	body := "package main\n\nimport \"example.com/p\"\n\nfunc main() { println(p.Value()) }\n"
	require.NoError(t, os.WriteFile(mainSrc, []byte(body), 0o644))
	pArchive := filepath.Join(work, "p.a")
	mainArchive := filepath.Join(work, "main.a")
	mainCfg := filepath.Join(work, "main.importcfg")
	mainBody := string(runtimeCfg) + "packagefile example.com/p=" + pArchive + "\n"
	require.NoError(t, os.WriteFile(mainCfg, []byte(mainBody), 0o644))
	deps, err := os.ReadFile(filepath.Join(work, "deps"))
	require.NoError(t, err)
	linkCfg := filepath.Join(work, "link.importcfg")
	linkBody := "# import config\npackagefile main=" + mainArchive + "\npackagefile example.com/p=" + pArchive + "\n" + string(deps)
	require.NoError(t, os.WriteFile(linkCfg, []byte(linkBody), 0o644))
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
	require.NoError(t, Build(t.Context(), units, image))
	out, err := exec.Command(bin).CombinedOutput()
	require.NoErrorf(t, err, "run: %v\n%s", err, out)
	if string(out) != "7\n" {
		require.FailNowf(t, "", "output %q", out)
	}
}
