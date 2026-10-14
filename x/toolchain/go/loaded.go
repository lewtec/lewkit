package gotool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lewtec/lewkit/x/toolchain/load"
)

// Loaded is one `lewkit release go` compile of a graph [load.Load] already
// selected. Std maps an import path to a standard-library or assembly
// archive. Work holds the archives and import configs; the caller removes it.
// Out is the requested -o path. Empty uses the base name of the graph
// directory when there is one main package. RequireMain is set when the
// caller asked to run the program or passed -o. Ldflags are extra linker
// arguments.
type Loaded struct {
	Graph       *load.Graph
	Std         map[string]string
	Work        string
	Out         string
	RequireMain bool
	Ldflags     []string
}

// BuildLoaded compiles Loaded and, when the graph has one main package,
// links it. The returned path is that binary. More than one main package
// is "release go: N main packages". A run or -o with no main is
// "release go: -o needs a main package". Assembly in the main package is
// "release go: main package has assembly". A missing archive is
// [load.ErrNoArchive]. The link waits for every package this call compiled.
func BuildLoaded(ctx context.Context, m Loaded) (string, error) {
	if m.Graph == nil {
		return "", fmt.Errorf("%w: graph", ErrNoWork)
	}
	var mains int
	for _, pkg := range m.Graph.Pkgs {
		if pkg.Name == "main" {
			mains++
		}
	}
	if mains > 1 {
		return "", fmt.Errorf("release go: %d %w", mains, ErrMainPackages)
	}
	if mains == 0 && m.RequireMain {
		return "", ErrReleaseGoONeedsAMainPackage
	}
	// Assembly and system objects stay as archives in -pkgdir. cmd/asm is
	// not part of this toolchain. Their pure-Go imports are still compiled.
	compiled := map[string]struct{}{}
	archives := map[string]string{}
	for _, pkg := range m.Graph.Pkgs {
		if pkg.Name == "main" && pkg.Asm {
			return "", ErrReleaseGoMainPackageHasAssembly
		}
		if pkg.Asm {
			if _, ok := m.Std[pkg.ImportPath]; !ok {
				return "", fmt.Errorf("%w: %s", load.ErrNoArchive, pkg.ImportPath)
			}
			continue
		}
		// The linker looks up main.main. The compiler's -p for that
		// package is main, not the module path.
		id := pkg.ImportPath
		if pkg.Name == "main" {
			id = "main"
		}
		compiled[pkg.ImportPath] = struct{}{}
		archives[id] = filepath.Join(m.Work, "pkg", filepath.FromSlash(id)+".a")
	}
	units := make([]Unit, 0, len(compiled))
	for _, pkg := range m.Graph.Pkgs {
		if pkg.Asm {
			continue
		}
		id := pkg.ImportPath
		if pkg.Name == "main" {
			id = "main"
		}
		cfg := map[string]string{"runtime": m.Std["runtime"]}
		for _, imp := range pkg.Imports {
			if imp == "unsafe" {
				continue
			}
			if a, ok := archives[imp]; ok {
				cfg[imp] = a
				continue
			}
			a, ok := m.Std[imp]
			if !ok {
				return "", fmt.Errorf("%w: %s", load.ErrNoArchive, imp)
			}
			cfg[imp] = a
		}
		cfgPath := filepath.Join(m.Work, "cfg", filepath.FromSlash(id)+".cfg")
		if err := writeModuleCfg(cfgPath, cfg); err != nil {
			return "", err
		}
		embedPath, err := writeModuleEmbed(m.Work, id, pkg.Embed)
		if err != nil {
			return "", err
		}
		var deps []string
		for _, dep := range pkg.Deps {
			if _, ok := compiled[dep]; ok {
				deps = append(deps, dep)
			}
		}
		units = append(units, Unit{
			ImportPath: id,
			Files:      pkg.Files,
			Deps:       deps,
			ImportCfg:  cfgPath,
			EmbedCfg:   embedPath,
			Lang:       pkg.Lang,
			Archive:    archives[id],
		})
	}
	var image *Image
	var linked string
	if mains == 1 {
		out := m.Out
		if out == "" {
			out = filepath.Base(m.Graph.Dir)
		}
		if !filepath.IsAbs(out) {
			out = filepath.Join(m.Graph.Dir, out)
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return "", err
		}
		link := map[string]string{}
		for path, archive := range m.Std {
			link[path] = archive
		}
		for path, archive := range archives {
			link[path] = archive
		}
		linkPath := filepath.Join(m.Work, "link.importcfg")
		if err := writeModuleCfg(linkPath, link); err != nil {
			return "", err
		}
		image = &Image{
			Out:       out,
			ImportCfg: linkPath,
			Mode:      "exe",
			Archive:   archives["main"],
			// Wait for every compiled package. A prebuilt assembly archive
			// can import one that main does not import directly.
			Ldflags: m.Ldflags,
		}
		linked = out
	}
	if err := Build(ctx, units, image); err != nil {
		return "", err
	}
	return linked, nil
}
