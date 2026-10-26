package gotool

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/toolchain/go/src/std/buildcfg"
	"github.com/lewtec/lewkit/x/toolchain/go/src/std/platform"
	"github.com/lewtec/lewkit/x/toolchain/load"
)

type stdPkg struct {
	ImportPath string
	Dir        string
	GoFiles    []string
	SFiles     []string
	SysoFiles  []string
	Imports    []string
	// Aliases maps a source import to the resolved path when they differ.
	// The standard library's copy of golang.org/x/net lives at src/vendor
	// and compiles as vendor/golang.org/x/net/...
	Aliases map[string]string
	Embed   *load.Embed
}

// carriedArchives is the standard library and the module packages that
// contain assembly, including the pure-Go packages those import.
// An empty Pkgdir compiles them. A set Pkgdir reads that directory.
func carriedArchives(ctx context.Context, work, trim string, m Module, g *load.Graph) (map[string]string, error) {
	if m.Pkgdir != "" {
		return load.Archives(m.Pkgdir)
	}
	if ctx.Err() != nil {
		return nil, context.Cause(ctx)
	}
	root, err := gorootDir()
	if err != nil {
		return nil, err
	}
	_, shared := moduleMode(m)
	bctx, err := stdBuildContext(root, m.GOOS, m.GOARCH, m.Tags)
	if err != nil {
		return nil, err
	}
	pkgs, err := loadStd(bctx, stdRoots(g))
	if err != nil {
		return nil, err
	}
	return compileCarried(ctx, work, trim, root, m, shared, pkgs, earlyPackages(g))
}

func compileCarried(ctx context.Context, work, trim, root string, m Module, shared bool, pkgs map[string]*stdPkg, early []load.Package) (map[string]string, error) {
	cache, err := stdCacheDir(m.GOOS, m.GOARCH, archiveKind(m.GOOS, m.GOARCH, shared), m.Tags)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cache, 0o755); err != nil {
		return nil, err
	}
	archives := map[string]string{}
	for path := range pkgs {
		archives[path] = filepath.Join(cache, filepath.FromSlash(path)+".a")
	}
	earlyBy := map[string]load.Package{}
	for _, pkg := range early {
		id := pkg.ImportPath
		if pkg.Name == "main" {
			id = "main"
		}
		archives[pkg.ImportPath] = filepath.Join(work, "pre", filepath.FromSlash(id)+".a")
		earlyBy[pkg.ImportPath] = pkg
	}
	build := carriedBuild{
		work:           work,
		include:        filepath.Join(root, "pkg", "include"),
		trim:           trim,
		stdTrim:        root,
		goos:           m.GOOS,
		goarch:         m.GOARCH,
		shared:         shared,
		toolexec:       m.Toolexec,
		pkgs:           pkgs,
		archives:       archives,
		runtimeClosure: closureOf(pkgs, "runtime"),
		early:          earlyBy,
	}
	units, err := build.units()
	if err != nil {
		return nil, err
	}
	if err := runIfNeeded(ctx, units); err != nil {
		return nil, err
	}
	if _, ok := archives["runtime"]; !ok {
		return nil, fmt.Errorf("%w: runtime", load.ErrNoArchive)
	}
	return archives, nil
}

type carriedBuild struct {
	work           string
	include        string
	trim           string
	stdTrim        string
	goos           string
	goarch         string
	shared         bool
	toolexec       string
	pkgs           map[string]*stdPkg
	archives       map[string]string
	runtimeClosure map[string]bool
	early          map[string]load.Package
	reuseCfg       string
}

func (b *carriedBuild) units() ([]Unit, error) {
	units := make([]Unit, 0, len(b.pkgs)+len(b.early))
	for path, sp := range b.pkgs {
		unit, err := b.stdUnit(path, sp)
		if err != nil {
			return nil, err
		}
		units = append(units, unit)
	}
	for _, pkg := range b.early {
		unit, err := b.earlyUnit(pkg)
		if err != nil {
			return nil, err
		}
		units = append(units, unit)
	}
	return units, nil
}

func (b *carriedBuild) stdUnit(path string, sp *stdPkg) (Unit, error) {
	archive := b.archives[path]
	cached := archiveReady(archive)
	deps := []string{}
	cfg := map[string]string{}
	if !cached {
		for _, imp := range sp.Imports {
			if skipImport(imp) || imp == path {
				continue
			}
			file, ok := b.archives[imp]
			if !ok {
				return Unit{}, fmt.Errorf("%w: %s", ErrStdlib, imp)
			}
			cfg[imp] = file
			deps = append(deps, imp)
		}
		if !b.runtimeClosure[path] {
			if _, ok := cfg["runtime"]; !ok {
				cfg["runtime"] = b.archives["runtime"]
				deps = append(deps, "runtime")
			}
		}
	}
	cfgPath, err := b.importCfg(path, cfg, sp.Aliases, cached)
	if err != nil {
		return Unit{}, err
	}
	embedPath := ""
	if !cached && sp.Embed != nil {
		embedPath, err = writeModuleEmbed(b.work, path, sp.Embed)
		if err != nil {
			return Unit{}, err
		}
	}
	sfiles := sp.SFiles
	syso := sp.SysoFiles
	if cached {
		deps = nil
	}
	return Unit{
		ImportPath: path,
		Files:      sp.GoFiles,
		Deps:       uniq(deps),
		ImportCfg:  cfgPath,
		EmbedCfg:   embedPath,
		Lang:       "go1.27",
		Archive:    archive,
		Shared:     b.shared,
		SFiles:     sfiles,
		SysoFiles:  syso,
		Std:        true,
		NoComplete: stdOmitsComplete(path) || len(sfiles) > 0 || len(syso) > 0,
		Include:    b.include,
		Reuse:      cached,
		GOOS:       b.goos,
		GOARCH:     b.goarch,
		Trimpath:   b.stdTrim,
	}, nil
}

func (b *carriedBuild) earlyUnit(pkg load.Package) (Unit, error) {
	id := pkg.ImportPath
	if pkg.Name == "main" {
		id = "main"
	}
	sfiles, syso := splitAsm(pkg.AsmFiles)
	files := append([]string(nil), pkg.Files...)
	extra, err := cgoExportFiles(b.toolexec, pkg.Dir)
	if err != nil {
		return Unit{}, err
	}
	files = append(files, extra...)
	if len(files) == 0 {
		return Unit{}, fmt.Errorf("%w: %s", ErrEmptyUnit, pkg.ImportPath)
	}
	cfg := map[string]string{}
	deps := []string{}
	for _, imp := range pkg.Imports {
		if skipImport(imp) {
			continue
		}
		file, ok := b.archives[imp]
		if !ok {
			return Unit{}, fmt.Errorf("%w: %s", load.ErrNoArchive, imp)
		}
		cfg[imp] = file
		if _, isStd := b.pkgs[imp]; isStd {
			deps = append(deps, imp)
		} else if _, isEarly := b.early[imp]; isEarly {
			deps = append(deps, b.stepName(imp))
		}
	}
	for _, dep := range pkg.Deps {
		if _, ok := b.early[dep]; !ok {
			continue
		}
		deps = append(deps, b.stepName(dep))
		if _, ok := cfg[dep]; !ok {
			cfg[dep] = b.archives[dep]
		}
	}
	if _, ok := cfg["runtime"]; !ok {
		cfg["runtime"] = b.archives["runtime"]
		deps = append(deps, "runtime")
	}
	cfgPath := filepath.Join(b.work, "cfg", "pre", filepath.FromSlash(id)+".cfg")
	if err := writeModuleCfg(cfgPath, cfg); err != nil {
		return Unit{}, err
	}
	embedPath, err := writeModuleEmbed(b.work, id, pkg.Embed)
	if err != nil {
		return Unit{}, err
	}
	return Unit{
		ImportPath: id,
		Files:      files,
		Deps:       uniq(deps),
		ImportCfg:  cfgPath,
		EmbedCfg:   embedPath,
		Lang:       pkg.Lang,
		Archive:    b.archives[pkg.ImportPath],
		Shared:     b.shared,
		SFiles:     sfiles,
		SysoFiles:  syso,
		NoComplete: len(sfiles) > 0 || len(syso) > 0,
		Include:    b.include,
		GOOS:       b.goos,
		GOARCH:     b.goarch,
		Trimpath:   b.trim,
	}, nil
}

func (b *carriedBuild) stepName(path string) string {
	if pkg, ok := b.early[path]; ok && pkg.Name == "main" {
		return "main"
	}
	return path
}

func (b *carriedBuild) importCfg(path string, cfg, aliases map[string]string, cached bool) (string, error) {
	if cached {
		if b.reuseCfg == "" {
			b.reuseCfg = filepath.Join(b.work, "cfg", "reuse.cfg")
			if err := writeModuleCfg(b.reuseCfg, nil); err != nil {
				return "", err
			}
		}
		return b.reuseCfg, nil
	}
	cfgPath := filepath.Join(b.work, "cfg", "std", filepath.FromSlash(path)+".cfg")
	if err := writeImportCfg(cfgPath, cfg, aliases); err != nil {
		return "", err
	}
	return cfgPath, nil
}

func runIfNeeded(ctx context.Context, units []Unit) error {
	if len(units) == 0 {
		return nil
	}
	for _, unit := range units {
		if !unit.Reuse {
			return Build(ctx, units, nil)
		}
	}
	return nil
}

func stdRoots(g *load.Graph) []string {
	inGraph := map[string]bool{}
	for _, pkg := range g.Pkgs {
		inGraph[pkg.ImportPath] = true
	}
	seen := map[string]bool{"runtime": true}
	roots := []string{"runtime"}
	for _, pkg := range g.Pkgs {
		for _, imp := range pkg.Imports {
			if skipImport(imp) || inGraph[imp] || seen[imp] || !stdImport(imp) {
				continue
			}
			seen[imp] = true
			roots = append(roots, imp)
		}
	}
	return roots
}

func earlyPackages(g *load.Graph) []load.Package {
	byPath := map[string]load.Package{}
	for _, pkg := range g.Pkgs {
		byPath[pkg.ImportPath] = pkg
	}
	want := map[string]bool{}
	var add func(string)
	add = func(path string) {
		if want[path] {
			return
		}
		pkg, ok := byPath[path]
		if !ok {
			return
		}
		want[path] = true
		for _, dep := range pkg.Deps {
			add(dep)
		}
	}
	for _, pkg := range g.Pkgs {
		if pkg.Asm {
			add(pkg.ImportPath)
		}
	}
	out := []load.Package{}
	for _, pkg := range g.Pkgs {
		if want[pkg.ImportPath] {
			out = append(out, pkg)
		}
	}
	return out
}

func loadStd(bctx build.Context, roots []string) (map[string]*stdPkg, error) {
	pkgs := map[string]*stdPkg{}
	color := map[string]int{}
	// srcDir is the importing directory so go/build finds GOROOT/src/vendor.
	var walk func(path, srcDir string) (string, error)
	walk = func(path, srcDir string) (string, error) {
		if skipImport(path) {
			return "", nil
		}
		bp, err := bctx.Import(path, srcDir, 0)
		if err != nil {
			return "", fmt.Errorf("%w: %s: %w", ErrStdlib, path, err)
		}
		resolved := bp.ImportPath
		switch color[resolved] {
		case 2:
			return resolved, nil
		case 1:
			return "", fmt.Errorf("%w: %s", errs.ErrCycle, resolved)
		}
		if !bp.Goroot || len(bp.GoFiles) == 0 {
			return "", fmt.Errorf("%w: %s", ErrStdlib, resolved)
		}
		color[resolved] = 1
		aliases := map[string]string{}
		imports := make([]string, 0, len(bp.Imports))
		for _, raw := range bp.Imports {
			final, err := walk(raw, bp.Dir)
			if err != nil {
				return "", err
			}
			if final == "" {
				continue
			}
			imports = append(imports, final)
			if final != raw {
				aliases[raw] = final
			}
		}
		emb, err := load.ResolveEmbed(bp.Dir, bp.EmbedPatterns)
		if err != nil {
			return "", err
		}
		pkgs[resolved] = &stdPkg{
			ImportPath: resolved,
			Dir:        bp.Dir,
			GoFiles:    joinNames(bp.Dir, bp.GoFiles),
			SFiles:     joinNames(bp.Dir, bp.SFiles),
			SysoFiles:  joinNames(bp.Dir, bp.SysoFiles),
			Imports:    imports,
			Aliases:    aliases,
			Embed:      emb,
		}
		color[resolved] = 2
		return resolved, nil
	}
	for _, root := range roots {
		if _, err := walk(root, ""); err != nil {
			return nil, err
		}
	}
	return pkgs, nil
}

func stdBuildContext(goroot, goos, goarch string, tags []string) (build.Context, error) {
	exp, err := buildcfg.ParseGOEXPERIMENT(goos, goarch, os.Getenv("GOEXPERIMENT"))
	if err != nil {
		return build.Context{}, fmt.Errorf("%w: %w", ErrStdlib, err)
	}
	tool := make([]string, 0, len(exp.Enabled()))
	for _, name := range exp.Enabled() {
		tool = append(tool, "goexperiment."+name)
	}
	rel := make([]string, 0, 27)
	for v := 1; v <= 27; v++ {
		rel = append(rel, "go1."+strconv.Itoa(v))
	}
	ctx := build.Default
	ctx.GOOS = goos
	ctx.GOARCH = goarch
	ctx.GOROOT = goroot
	ctx.GOPATH = ""
	ctx.CgoEnabled = false
	ctx.Compiler = "gc"
	ctx.BuildTags = append([]string(nil), tags...)
	ctx.ToolTags = tool
	ctx.ReleaseTags = rel
	return ctx, nil
}

func closureOf(pkgs map[string]*stdPkg, root string) map[string]bool {
	out := map[string]bool{}
	var walk func(string)
	walk = func(path string) {
		if out[path] {
			return
		}
		sp := pkgs[path]
		if sp == nil {
			return
		}
		out[path] = true
		for _, imp := range sp.Imports {
			if skipImport(imp) {
				continue
			}
			walk(imp)
		}
	}
	walk(root)
	return out
}

func archiveKind(goos, goarch string, shared bool) string {
	if shared {
		return "pie-shared"
	}
	if platform.DefaultPIE(goos, goarch, false) {
		return "pie"
	}
	return "exe"
}

func stdCacheDir(goos, goarch, kind string, tags []string) (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	sum := sha256.New()
	fmt.Fprintf(sum, "goexperiment=%s\ngoamd64=%s\ngoarm64=%s\n", os.Getenv("GOEXPERIMENT"), os.Getenv("GOAMD64"), os.Getenv("GOARM64"))
	sorted := append([]string(nil), tags...)
	sort.Strings(sorted)
	for _, tag := range sorted {
		fmt.Fprintf(sum, "tag=%s\n", tag)
	}
	digest := hex.EncodeToString(sum.Sum(nil))[:16]
	return filepath.Join(base, "lewkit", "std", "go1.27.0-"+gorootHash(), goos+"_"+goarch+"_"+kind+"_"+digest), nil
}

func archiveReady(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Size() > 0
}

func stdOmitsComplete(path string) bool {
	switch path {
	case "bytes", "internal/poll", "net", "os", "runtime/metrics", "runtime/pprof", "runtime/trace", "sync", "syscall", "time":
		return true
	default:
		return false
	}
}

func stdImport(path string) bool {
	first, _, _ := strings.Cut(path, "/")
	return first != "" && !strings.Contains(first, ".")
}

func skipImport(path string) bool {
	return path == "" || path == "C" || path == "unsafe"
}

func splitAsm(files []string) (sfiles, syso []string) {
	for _, file := range files {
		if strings.HasSuffix(file, ".syso") {
			syso = append(syso, file)
			continue
		}
		sfiles = append(sfiles, file)
	}
	return sfiles, syso
}

func joinNames(dir string, names []string) []string {
	out := make([]string, len(names))
	for i, name := range names {
		out[i] = filepath.Join(dir, name)
	}
	return out
}

func uniq(values []string) []string {
	if len(values) < 2 {
		return values
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

// cgoExportFiles lists the _cgo_*_export.go files this compile must see.
// cmd/go skips a name that starts with _. -toolexec names the file the
// host wrapper would append. That path is the lewkit checkout building
// the app. The package being compiled may be another copy, such as the
// module cache. The file next to that package matches its assembly.
// A package without that name is left alone, so the checkout file is
// not compiled into main.
func cgoExportFiles(toolexec, pkgDir string) ([]string, error) {
	if toolexec == "" {
		return nil, nil
	}
	words, err := splitToolexec(toolexec)
	if err != nil {
		return nil, err
	}
	var out []string
	seen := map[string]struct{}{}
	for _, word := range words {
		base := filepath.Base(word)
		if !cgoExportName(base) {
			continue
		}
		path := word
		if filepath.Clean(filepath.Dir(word)) != filepath.Clean(pkgDir) {
			if pkgDir == "" {
				continue
			}
			local := filepath.Join(pkgDir, base)
			info, err := os.Stat(local)
			if err != nil || info.IsDir() {
				continue
			}
			path = local
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out, nil
}

func cgoExportName(base string) bool {
	return strings.HasPrefix(base, "_cgo_") && strings.HasSuffix(base, "_export.go")
}

func splitToolexec(s string) ([]string, error) {
	var words []string
	var b strings.Builder
	var quote rune
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
				continue
			}
			b.WriteRune(r)
		case r == '\'' || r == '"':
			quote = r
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			if b.Len() > 0 {
				words = append(words, b.String())
				b.Reset()
			}
		default:
			b.WriteRune(r)
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("%w: unclosed quote", ErrToolexec)
	}
	if b.Len() > 0 {
		words = append(words, b.String())
	}
	return words, nil
}

func hostGOOS() string {
	if value := os.Getenv("GOOS"); value != "" {
		return value
	}
	return runtime.GOOS
}

func hostGOARCH() string {
	if value := os.Getenv("GOARCH"); value != "" {
		return value
	}
	return runtime.GOARCH
}
