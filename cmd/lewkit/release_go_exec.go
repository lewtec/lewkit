package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lewtec/lewkit/x/cmd"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/toolchain/go"
	"github.com/lewtec/lewkit/x/toolchain/load"
)

func (c *goBuildCmd) Run(ctx context.Context) error {
	return c.execute(ctx, false, nil)
}

func (c *goRunCmd) Run(ctx context.Context) error {
	return c.execute(ctx, true, cmd.Values(c.args))
}

func (c *goBuildCmd) execute(ctx context.Context, run bool, args []string) error {
	if ctx == nil {
		return gotool.ErrNilContext
	}
	ldflags, err := load.SplitFlags(c.ldflags.Value())
	if err != nil {
		return err
	}
	dir := c.dir.Value()
	if dir == "" {
		dir = "."
	}
	g, err := load.Load(dir, cmd.Values(c.pkgs), load.SplitTags(c.tags.Value()))
	if err != nil {
		return err
	}
	std, err := load.Archives(c.pkgdir.Value())
	if err != nil {
		return err
	}
	work, err := os.MkdirTemp("", "lewkit-go-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)

	var mains int
	for _, pkg := range g.Pkgs {
		if pkg.Name == "main" {
			mains++
		}
	}
	if mains > 1 {
		return fmt.Errorf("release go: %d main packages", mains)
	}
	if mains == 0 && (run || c.output.Value() != "") {
		return fmt.Errorf("release go: -o needs a main package")
	}
	// Assembly and system objects stay as archives in -pkgdir. cmd/asm is
	// not part of this toolchain. Their pure-Go imports are still compiled.
	compiled := map[string]struct{}{}
	archives := map[string]string{}
	for _, pkg := range g.Pkgs {
		if pkg.Name == "main" && pkg.Asm {
			return fmt.Errorf("release go: main package has assembly")
		}
		if pkg.Asm {
			if _, ok := std[pkg.ImportPath]; !ok {
				return fmt.Errorf("%w: %s", load.ErrNoArchive, pkg.ImportPath)
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
		archives[id] = filepath.Join(work, "pkg", filepath.FromSlash(id)+".a")
	}
	units := make([]gotool.Unit, 0, len(compiled))
	for _, pkg := range g.Pkgs {
		if pkg.Asm {
			continue
		}
		id := pkg.ImportPath
		if pkg.Name == "main" {
			id = "main"
		}
		cfg := map[string]string{"runtime": std["runtime"]}
		for _, imp := range pkg.Imports {
			if imp == "unsafe" {
				continue
			}
			if a, ok := archives[imp]; ok {
				cfg[imp] = a
				continue
			}
			a, ok := std[imp]
			if !ok {
				return fmt.Errorf("%w: %s", load.ErrNoArchive, imp)
			}
			cfg[imp] = a
		}
		cfgPath := filepath.Join(work, "cfg", filepath.FromSlash(id)+".cfg")
		if err := writeImportCfg(cfgPath, cfg); err != nil {
			return err
		}
		embedPath, err := writeEmbedCfg(work, id, pkg.Embed)
		if err != nil {
			return err
		}
		var deps []string
		for _, dep := range pkg.Deps {
			if _, ok := compiled[dep]; ok {
				deps = append(deps, dep)
			}
		}
		units = append(units, gotool.Unit{
			ImportPath: id,
			Files:      pkg.Files,
			Deps:       deps,
			ImportCfg:  cfgPath,
			EmbedCfg:   embedPath,
			Lang:       pkg.Lang,
			Archive:    archives[id],
		})
	}
	var image *gotool.Image
	if mains == 1 {
		out := c.output.Value()
		if out == "" {
			out = filepath.Base(g.Dir)
		}
		if !filepath.IsAbs(out) {
			out = filepath.Join(g.Dir, out)
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		link := map[string]string{}
		for path, archive := range std {
			link[path] = archive
		}
		for path, archive := range archives {
			link[path] = archive
		}
		linkPath := filepath.Join(work, "link.importcfg")
		if err := writeImportCfg(linkPath, link); err != nil {
			return err
		}
		image = &gotool.Image{
			Out:       out,
			ImportCfg: linkPath,
			Mode:      "exe",
			Archive:   archives["main"],
			// Wait for every compiled package. A prebuilt assembly archive
			// can import one that main does not import directly.
			Ldflags: ldflags,
		}
	}
	if err := gotool.Build(ctx, units, image); err != nil {
		return err
	}
	if !run {
		return nil
	}
	cmd := execdriver.MustCommand(ctx, image.Out, args...)
	cmd.Dir = g.Dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return execdriver.Run(ctx, cmd)
}

func writeEmbedCfg(work, id string, emb *load.Embed) (string, error) {
	if emb == nil {
		return "", nil
	}
	body, err := json.Marshal(emb)
	if err != nil {
		return "", err
	}
	path := filepath.Join(work, "cfg", filepath.FromSlash(id)+".embed")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func writeImportCfg(path string, files map[string]string) error {
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("# import config\n")
	for _, k := range keys {
		fmt.Fprintf(&b, "packagefile %s=%s\n", k, files[k])
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
