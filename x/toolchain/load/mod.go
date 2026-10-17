package load

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

// modSrc is one module on the build list. path is the import prefix.
// cachePath and version name the module in GOMODCACHE when dir is empty.
// reqVer is the requirement version that selected the module, before a
// main-module replace. The highest reqVer wins.
type modSrc struct {
	path      string
	cachePath string
	version   string
	dir       string
	lang      string
	reqVer    string
}

func (l *loader) buildList(f *modfile.File) error {
	lang := "go1.27"
	if f.Go != nil && f.Go.Version != "" {
		lang = langTag(f.Go.Version)
	}
	l.mods = []modSrc{{
		path:      l.module,
		cachePath: l.module,
		dir:       l.modDir,
		lang:      lang,
	}}
	repls := make([]modReplace, 0, len(f.Replace))
	for _, r := range f.Replace {
		if r == nil || r.Old.Path == "" {
			continue
		}
		rep := modReplace{oldPath: r.Old.Path, oldVer: r.Old.Version}
		if r.New.Version == "" {
			local := r.New.Path
			if !filepath.IsAbs(local) {
				local = filepath.Join(l.modDir, local)
			}
			rep.local = filepath.Clean(local)
		} else {
			rep.newPath = r.New.Path
			rep.newVer = r.New.Version
		}
		repls = append(repls, rep)
	}
	l.repls = repls
	for _, req := range f.Require {
		if req == nil {
			continue
		}
		l.addRequire(req.Mod.Path, req.Mod.Version)
	}
	return l.expandModules()
}

type modReplace struct {
	oldPath string
	oldVer  string
	newPath string
	newVer  string
	local   string
}

func applyReplace(path, ver string, repls []modReplace) (cachePath, version, dir string) {
	var hit *modReplace
	for i := range repls {
		r := &repls[i]
		if r.oldPath != path {
			continue
		}
		if r.oldVer != "" && r.oldVer != ver {
			continue
		}
		hit = r
	}
	if hit == nil {
		return path, ver, ""
	}
	if hit.local != "" {
		return path, "", hit.local
	}
	return hit.newPath, hit.newVer, ""
}

// moduleRequire is one requirement read from a module's go.mod.
type moduleRequire struct {
	path    string
	version string
}

// expandModules adds requirements from each selected module's go.mod.
// A module that is not in the cache stays on the list; importing it
// reports ErrMissing later. The main module is already expanded.
func (l *loader) expandModules() error {
	seen := map[string]string{}
	for {
		progress := false
		for i := 0; i < len(l.mods); i++ {
			if l.mods[i].path == l.module {
				continue
			}
			path := l.mods[i].path
			if seen[path] == moduleIdentity(&l.mods[i]) {
				continue
			}
			reqs, err := l.moduleRequires(&l.mods[i])
			if err != nil {
				if errors.Is(err, ErrMissing) {
					seen[path] = moduleIdentity(&l.mods[i])
					continue
				}
				return err
			}
			id := moduleIdentity(&l.mods[i])
			for _, req := range reqs {
				if l.addRequire(req.path, req.version) {
					progress = true
				}
			}
			seen[path] = id
		}
		if !progress {
			return nil
		}
	}
}

// addRequire records path at ver when ver is newer than the selected
// requirement. The main module's replace directives decide the directory.
// A false result means the build list already has this version or a newer one.
func (l *loader) addRequire(path, ver string) bool {
	if path == "" || path == l.module {
		return false
	}
	for i := range l.mods {
		if l.mods[i].path != path {
			continue
		}
		if semver.Compare(ver, l.mods[i].reqVer) <= 0 {
			return false
		}
		cachePath, lookupVer, dir := applyReplace(path, ver, l.repls)
		l.mods[i].reqVer = ver
		l.mods[i].cachePath = cachePath
		l.mods[i].version = lookupVer
		l.mods[i].dir = dir
		l.mods[i].lang = ""
		return true
	}
	cachePath, lookupVer, dir := applyReplace(path, ver, l.repls)
	l.mods = append(l.mods, modSrc{
		path:      path,
		cachePath: cachePath,
		version:   lookupVer,
		dir:       dir,
		reqVer:    ver,
	})
	return true
}

func (l *loader) moduleRequires(m *modSrc) ([]moduleRequire, error) {
	root, err := l.modRoot(m)
	if err != nil {
		return nil, err
	}
	body, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	parsed, err := modfile.Parse("go.mod", body, nil)
	if err != nil {
		return nil, err
	}
	reqs := make([]moduleRequire, 0, len(parsed.Require))
	for _, req := range parsed.Require {
		if req == nil || req.Mod.Path == "" {
			continue
		}
		reqs = append(reqs, moduleRequire{path: req.Mod.Path, version: req.Mod.Version})
	}
	return reqs, nil
}

// moduleIdentity names the selected requirement and where it was found.
// A higher requirement changes it, so expandModules reads that module again.
func moduleIdentity(m *modSrc) string {
	return m.reqVer + "\x00" + m.version + "\x00" + m.cachePath + "\x00" + m.dir
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
