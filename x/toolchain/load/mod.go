package load

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lewtec/lewkit/x/errs"
	gobase "github.com/lewtec/lewkit/x/toolchain/go/src/go/base"
	modcfg "github.com/lewtec/lewkit/x/toolchain/go/src/go/cfg"
	"github.com/lewtec/lewkit/x/toolchain/go/src/go/modload"
	"github.com/lewtec/lewkit/x/toolchain/go/src/go/mvs"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

// modSrc is one module on the build list. path is the import prefix.
// cachePath and version name the module in GOMODCACHE when dir is empty.
// dir is set for the main module, a workspace module, or a local replace.
type modSrc struct {
	path      string
	cachePath string
	version   string
	dir       string
	lang      string
}

// The vendored module loader keeps its configuration in package globals.
var modloadMu sync.Mutex

// buildList selects versions with the vendored go command module loader.
func (l *loader) buildList(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := rejectWorkspace(); err != nil {
		return err
	}
	modloadMu.Lock()
	defer modloadMu.Unlock()
	return l.selectModules(ctx)
}

func (l *loader) selectModules(ctx context.Context) (err error) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		signal, ok := recovered.(gobase.ExitSignal)
		if !ok {
			panic(recovered)
		}
		err = fmt.Errorf("%w: module loader exited: %d", errs.ErrInvalid, int(signal))
	}()
	saved := saveModConfig()
	defer saved.restore()
	modcfg.CmdName = "mod graph"
	modcfg.BuildMod = "mod"
	modcfg.BuildModExplicit = false
	modcfg.GOMODCACHE = l.cache
	modcfg.GOPROXY = "off"
	modcfg.GOSUMDB = "off"

	moduleLoader, graph, err := modload.SelectGraph(ctx, l.modDir)
	if err != nil {
		return missingModule(err)
	}
	l.modsFromGraph(moduleLoader, graph)
	return nil
}

type modConfig struct {
	cmdName        string
	buildMod       string
	buildModSet    bool
	modCache       string
	proxy          string
	sumDB          string
	modulesEnabled bool
}

func saveModConfig() modConfig {
	return modConfig{
		cmdName:        modcfg.CmdName,
		buildMod:       modcfg.BuildMod,
		buildModSet:    modcfg.BuildModExplicit,
		modCache:       modcfg.GOMODCACHE,
		proxy:          modcfg.GOPROXY,
		sumDB:          modcfg.GOSUMDB,
		modulesEnabled: modcfg.ModulesEnabled,
	}
}

func (c modConfig) restore() {
	modcfg.CmdName = c.cmdName
	modcfg.BuildMod = c.buildMod
	modcfg.BuildModExplicit = c.buildModSet
	modcfg.GOMODCACHE = c.modCache
	modcfg.GOPROXY = c.proxy
	modcfg.GOSUMDB = c.sumDB
	modcfg.ModulesEnabled = c.modulesEnabled
}

// rejectWorkspace reports a GOWORK value the go command would reject.
// FindGoWork exits the process for a relative path, so check it first.
func rejectWorkspace() error {
	gowork := os.Getenv("GOWORK")
	switch gowork {
	case "", "auto", "off":
		return nil
	}
	if !filepath.IsAbs(gowork) {
		return fmt.Errorf("%w: GOWORK is not an absolute path", ErrWorkspace)
	}
	info, err := os.Stat(gowork)
	if err != nil || info.IsDir() {
		return fmt.Errorf("%w: %s", ErrWorkspace, gowork)
	}
	return nil
}

func missingModule(err error) error {
	if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	var list *mvs.BuildListError
	if errors.As(err, &list) {
		m := list.Module()
		if m.Path != "" {
			if m.Version == "" {
				return fmt.Errorf("%w: %s", ErrMissing, m.Path)
			}
			return fmt.Errorf("%w: %s@%s", ErrMissing, m.Path, m.Version)
		}
	}
	return fmt.Errorf("%w: %s", ErrMissing, err.Error())
}

func (l *loader) modsFromGraph(moduleLoader *modload.Loader, graph *modload.ModuleGraph) {
	l.mods = nil
	for _, m := range graph.BuildList() {
		if m.Path == "go" || m.Path == "toolchain" || m.Version == "none" {
			continue
		}
		if m.Version == "" && moduleLoader.MainModules != nil && moduleLoader.MainModules.Contains(m.Path) {
			l.mods = append(l.mods, modSrc{
				path: m.Path,
				dir:  moduleLoader.MainModules.ModRoot(m),
			})
			continue
		}
		src := modSrc{path: m.Path, cachePath: m.Path, version: m.Version}
		replacement := modload.Replacement(moduleLoader, m)
		if replacement.Version != "" {
			src.cachePath = replacement.Path
			src.version = replacement.Version
		} else if replacement.Path != "" {
			src.dir = l.replacementDir(moduleLoader, replacement.Path)
			src.version = ""
		}
		l.mods = append(l.mods, src)
	}
}

func (l *loader) replacementDir(moduleLoader *modload.Loader, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	base := l.modDir
	if work := modload.WorkFilePath(moduleLoader); work != "" {
		base = filepath.Dir(work)
	}
	return filepath.Clean(filepath.Join(base, path))
}

func (l *loader) longest(path string) *modSrc {
	var best *modSrc
	for i := range l.mods {
		m := &l.mods[i]
		if path != m.path && !strings.HasPrefix(path, m.path+"/") {
			continue
		}
		if best == nil || len(m.path) > len(best.path) {
			best = m
		}
	}
	return best
}

func (l *loader) dirForImport(path string) (string, error) {
	if isStd(path) {
		return "", fmt.Errorf("%w: %s", ErrExternal, path)
	}
	m := l.longest(path)
	if m == nil {
		return "", fmt.Errorf("%w: %s", ErrExternal, path)
	}
	root, err := l.modRoot(m)
	if err != nil {
		return "", err
	}
	rel := strings.TrimPrefix(path, m.path)
	rel = strings.TrimPrefix(rel, "/")
	return filepath.Join(root, filepath.FromSlash(rel)), nil
}

func (l *loader) modRoot(m *modSrc) (string, error) {
	if m.dir == "" {
		root, err := moduleCacheDir(l.cache, m.cachePath, m.version)
		if err != nil {
			return "", err
		}
		st, err := os.Stat(root)
		if err != nil || !st.IsDir() {
			if m.version == "" {
				return "", fmt.Errorf("%w: %s", ErrMissing, m.path)
			}
			return "", fmt.Errorf("%w: %s@%s", ErrMissing, m.cachePath, m.version)
		}
		m.dir = root
	} else if st, err := os.Stat(m.dir); err != nil || !st.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrMissing, m.path)
	}
	if m.lang == "" {
		m.lang = readLang(m.dir)
	}
	return m.dir, nil
}

func (l *loader) langFor(path string) string {
	m := l.longest(path)
	if m == nil || m.lang == "" {
		return "go1.27"
	}
	return m.lang
}

func readLang(dir string) string {
	body, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "go1.27"
	}
	f, err := modfile.Parse("go.mod", body, nil)
	if err != nil || f.Go == nil || f.Go.Version == "" {
		return "go1.27"
	}
	return langTag(f.Go.Version)
}

func langTag(v string) string {
	v = strings.TrimPrefix(v, "go")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	major, rest, ok := strings.Cut(v, ".")
	if !ok || major == "" {
		return "go1.27"
	}
	minor, _, _ := strings.Cut(rest, ".")
	if minor == "" {
		return "go1.27"
	}
	return "go" + major + "." + minor
}

func moduleCacheDir(cache, path, version string) (string, error) {
	enc, err := module.EscapePath(path)
	if err != nil {
		return "", err
	}
	if version == "" {
		return "", fmt.Errorf("%w: %s", ErrMissing, path)
	}
	encVer, err := module.EscapeVersion(version)
	if err != nil {
		return "", err
	}
	return filepath.Join(cache, enc+"@"+encVer), nil
}

func modCache() (string, error) {
	if c := os.Getenv("GOMODCACHE"); c != "" {
		return filepath.Abs(c)
	}
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		gopath = filepath.Join(home, "go")
	} else {
		gopath = strings.Split(gopath, string(os.PathListSeparator))[0]
	}
	return filepath.Join(gopath, "pkg", "mod"), nil
}
