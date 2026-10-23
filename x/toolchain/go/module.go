package gotool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/lewtec/lewkit/x/toolchain/go/src/std/platform"
	"github.com/lewtec/lewkit/x/toolchain/load"
)

// Module is one CGO-free build of a main package.
// GOOS and GOARCH empty mean this process. GOARCH other than amd64 or arm64
// is an error. Patterns empty means the package in Dir. Tags are build tags.
// Ldflags are linker arguments. Pkgdir, when set, supplies the standard
// library and any package with assembly. An empty Pkgdir compiles those
// with this compiler and caches the archives by GOOS, GOARCH, and build
// mode. Toolexec, when set, appends a file whose base name starts with
// _cgo_ and ends with _export.go when that file's directory is the
// package being compiled.
// Out is the executable. Trimpath strips Dir from recorded source paths.
// Android, and any other target whose default buildmode is PIE, links as
// PIE. Library forces that PIE for a shared library, including Linux.
// The compiler uses -shared for that PIE except on Windows.
// iOS is one of those PIE targets and links internally. A main package
// that contains assembly is assembled here when Pkgdir is empty, and it
// uses the Pkgdir archive when that directory recorded one.
type Module struct {
	Dir      string
	Out      string
	GOOS     string
	GOARCH   string
	Patterns []string
	Tags     []string
	Ldflags  []string
	Pkgdir   string
	Trimpath bool
	Toolexec string
	Library  bool
}

// BuildModule compiles Module and links its main package.
// The compile graph stays on the caller session. Each package and the
// link are CPU tasks under the current task. The caller must be Control
// so that wait does not hold a CPU slot.
func BuildModule(ctx context.Context, m Module) error {
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	if m.GOOS == "" {
		m.GOOS = runtime.GOOS
	}
	if m.GOARCH == "" {
		m.GOARCH = runtime.GOARCH
	}
	if m.GOARCH != "amd64" && m.GOARCH != "arm64" {
		return fmt.Errorf("%w %s", ErrToolchainCompileSupportsGoarchAmd64OrArm64Have, m.GOARCH)
	}
	if platform.MustLinkExternal(m.GOOS, m.GOARCH, false) {
		return fmt.Errorf("toolchain: %s/%s %w", m.GOOS, m.GOARCH, ErrRequiresExternalLinking)
	}
	mode, shared := moduleMode(m)
	if m.Out == "" {
		return fmt.Errorf("%w: output", ErrEmptyImage)
	}
	dir := m.Dir
	if dir == "" {
		dir = "."
	}
	trim := ""
	if m.Trimpath {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return err
		}
		trim = abs
	}
	g, err := load.LoadTarget(dir, m.Patterns, m.Tags, m.GOOS, m.GOARCH)
	if err != nil {
		return err
	}
	work, err := os.MkdirTemp("", "lewkit-module-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	std, err := carriedArchives(ctx, work, trim, m, g)
	if err != nil {
		return err
	}

	var mains int
	compiled := map[string]struct{}{}
	archives := map[string]string{}
	for _, pkg := range g.Pkgs {
		if pkg.Name == "main" {
			mains++
		}
		if archive, ok := std[pkg.ImportPath]; ok && (pkg.Asm || m.Pkgdir == "") {
			if pkg.Name == "main" {
				archives["main"] = archive
				delete(std, pkg.ImportPath)
			}
			continue
		}
		if pkg.Asm {
			if pkg.Name == "main" {
				return ErrToolchainMainPackageHasAssembly
			}
			return fmt.Errorf("%w: %s", load.ErrNoArchive, pkg.ImportPath)
		}
		id := pkg.ImportPath
		if pkg.Name == "main" {
			id = "main"
		}
		compiled[pkg.ImportPath] = struct{}{}
		archives[id] = filepath.Join(work, "pkg", filepath.FromSlash(id)+".a")
	}
	if mains != 1 {
		return fmt.Errorf("toolchain: %d %w", mains, ErrMainPackages)
	}
	units := make([]Unit, 0, len(compiled))
	for _, pkg := range g.Pkgs {
		if _, ok := compiled[pkg.ImportPath]; !ok {
			continue
		}
		id := pkg.ImportPath
		if pkg.Name == "main" {
			id = "main"
		}
		files := append([]string(nil), pkg.Files...)
		extra, err := cgoExportFiles(m.Toolexec, pkg.Dir)
		if err != nil {
			return err
		}
		files = append(files, extra...)
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
		if err := writeModuleCfg(cfgPath, cfg); err != nil {
			return err
		}
		embedPath, err := writeModuleEmbed(work, id, pkg.Embed)
		if err != nil {
			return err
		}
		var deps []string
		for _, dep := range pkg.Deps {
			if _, ok := compiled[dep]; ok {
				deps = append(deps, dep)
			}
		}
		units = append(units, Unit{
			ImportPath: id,
			Files:      files,
			Deps:       deps,
			ImportCfg:  cfgPath,
			EmbedCfg:   embedPath,
			Lang:       pkg.Lang,
			Archive:    archives[id],
			GOOS:       m.GOOS,
			GOARCH:     m.GOARCH,
			Trimpath:   trim,
			Shared:     shared,
		})
	}
	out := m.Out
	if !filepath.IsAbs(out) {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return err
		}
		out = filepath.Join(absDir, out)
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
	if err := writeModuleCfg(linkPath, link); err != nil {
		return err
	}
	image := &Image{
		Out:       out,
		ImportCfg: linkPath,
		Mode:      mode,
		Archive:   archives["main"],
		Ldflags:   append([]string(nil), m.Ldflags...),
		GOOS:      m.GOOS,
		GOARCH:    m.GOARCH,
	}
	return Build(ctx, units, image)
}

func moduleMode(m Module) (mode string, shared bool) {
	mode = "exe"
	if m.Library || platform.DefaultPIE(m.GOOS, m.GOARCH, false) {
		mode = "pie"
		if m.GOOS != "windows" {
			shared = true
		}
	}
	return mode, shared
}

func writeModuleCfg(path string, files map[string]string) error {
	return writeImportCfg(path, files, nil)
}

func writeImportCfg(path string, files, aliases map[string]string) error {
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	raws := make([]string, 0, len(aliases))
	for raw := range aliases {
		raws = append(raws, raw)
	}
	sort.Strings(raws)
	var b strings.Builder
	b.WriteString("# import config\n")
	for _, raw := range raws {
		fmt.Fprintf(&b, "importmap %s=%s\n", raw, aliases[raw])
	}
	for _, k := range keys {
		fmt.Fprintf(&b, "packagefile %s=%s\n", k, files[k])
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeModuleEmbed(work, id string, emb *load.Embed) (string, error) {
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
