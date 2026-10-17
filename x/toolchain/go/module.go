package gotool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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
// library and any package with assembly. An empty Pkgdir asks the host
// go command for those archives with go list -export. Toolexec, when set,
// is that list's -toolexec value. The basename must start with _cgo_ and
// end with _export.go, so an assembly package can see that file.
// Out is the executable. Trimpath strips Dir from recorded source paths.
// Android, and any other target whose default buildmode is PIE, links as
// PIE. Library forces that PIE for a shared library, including Linux.
// The compiler uses -shared for that PIE except on Windows. go list
// receives -buildmode=pie for the same link, so the archives match.
// A main package that contains assembly uses the go list archive when
// that list recorded one. The link still runs when every package came
// from that list.
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
	if err := ctx.Err(); err != nil {
		return err
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
	g, err := load.LoadTarget(ctx, dir, m.Patterns, m.Tags, m.GOOS, m.GOARCH)
	if err != nil {
		return err
	}
	std, err := moduleArchives(ctx, dir, m, mode)
	if err != nil {
		return err
	}
	work, err := os.MkdirTemp("", "lewkit-module-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)

	var mains int
	compiled := map[string]struct{}{}
	archives := map[string]string{}
	for _, pkg := range g.Pkgs {
		if pkg.Name == "main" {
			mains++
			if pkg.Asm {
				archive, ok := std[pkg.ImportPath]
				if !ok {
					return ErrToolchainMainPackageHasAssembly
				}
				archives["main"] = archive
				continue
			}
		}
		if pkg.Asm {
			if _, ok := std[pkg.ImportPath]; !ok {
				return fmt.Errorf("%w: %s", load.ErrNoArchive, pkg.ImportPath)
			}
			continue
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
			Files:      pkg.Files,
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

func moduleArchives(ctx context.Context, dir string, m Module, mode string) (map[string]string, error) {
	if m.Pkgdir != "" {
		return load.Archives(m.Pkgdir)
	}
	patterns := m.Patterns
	if len(patterns) == 0 {
		patterns = []string{"."}
	}
	args := []string{
		"list", "-buildvcs=false", "-deps", "-export",
		"-f", "{{if and .Export (or .Standard (or .SFiles .SysoFiles))}}{{.ImportPath}} {{.Export}}{{end}}",
	}
	if mode == "pie" {
		args = append(args, "-buildmode=pie")
	}
	if len(m.Tags) > 0 {
		args = append(args, "-tags", strings.Join(m.Tags, ","))
	}
	if m.Toolexec != "" {
		args = append(args, "-toolexec", m.Toolexec)
	}
	args = append(args, patterns...)
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	cmd.Env = overlayEnv(os.Environ(), "GOOS", m.GOOS, "GOARCH", m.GOARCH, "CGO_ENABLED", "0")
	out, err := cmd.Output()
	if err != nil {
		stderr := ""
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = string(ee.Stderr)
		}
		return nil, fmt.Errorf("%w: %w\n%s", ErrModuleArchivesGoList, err, stderr)
	}
	archives := map[string]string{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		imp, file, ok := strings.Cut(line, " ")
		if !ok {
			return nil, fmt.Errorf("%w %q", ErrModuleArchivesGoListLine, line)
		}
		archives[imp] = file
	}
	if _, ok := archives["runtime"]; !ok {
		return nil, fmt.Errorf("%w: runtime", load.ErrNoArchive)
	}
	return archives, nil
}

func overlayEnv(env []string, kv ...string) []string {
	drop := map[string]bool{}
	for i := 0; i < len(kv); i += 2 {
		drop[kv[i]] = true
	}
	out := dropKeys(env, drop)
	for i := 0; i < len(kv); i += 2 {
		out = append(out, kv[i]+"="+kv[i+1])
	}
	return out
}

func dropKeys(env []string, drop map[string]bool) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		k, _, _ := strings.Cut(e, "=")
		if drop[k] {
			continue
		}
		out = append(out, e)
	}
	return out
}

func writeModuleCfg(path string, files map[string]string) error {
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
