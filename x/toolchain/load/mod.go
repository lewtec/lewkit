package load

import (
	"errors"
	"fmt"
	"go/version"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

// modSrc is one module on the build list. path is the import prefix.
// cachePath and version name the module in GOMODCACHE when dir is empty.
// selected is the version chosen by minimal version selection, before a
// main-module or workspace replace. The highest selected version wins.
type modSrc struct {
	path      string
	cachePath string
	version   string
	dir       string
	lang      string
	selected  string
}

// modSummary is the part of a go.mod file the module graph walks.
// unpruned is true when the go line is older than 1.17, or missing.
type modSummary struct {
	reqs     []module.Version
	unpruned bool
}

// visit is one load of a module version. The same version is loaded
// again when an unpruned path reaches it after a pruned path did.
type visit struct {
	path     string
	version  string
	unpruned bool
}

// buildList selects versions with the go command's module graph rules:
// minimal version selection, pruning at go 1.17, and workspace roots.
// golang.org/x/mod parses the files and compares versions.
func (l *loader) buildList(f *modfile.File) error {
	l.mods = []modSrc{{
		path:      l.module,
		cachePath: l.module,
		dir:       l.modDir,
		lang:      declaredLang(f),
	}}
	l.mains = map[string]bool{l.module: true}
	l.summaries = map[module.Version]modSummary{}
	l.excluded = map[module.Version]struct{}{}
	workPath, err := findWorkspace(l.modDir)
	if err != nil {
		return err
	}
	if workPath != "" {
		return l.buildWorkspace(workPath)
	}
	l.repls = replacesFrom(l.modDir, f.Replace)
	l.excluded = excludesFrom(f)
	seen := map[visit]struct{}{}
	unpruned := moduleUnpruned(f)
	for _, req := range l.keptRequires(f) {
		l.note(req.Path, req.Version)
		if err := l.walk(req, unpruned, seen); err != nil {
			return err
		}
	}
	return nil
}

type modReplace struct {
	oldPath string
	oldVer  string
	newPath string
	newVer  string
	local   string
}

// resolve applies the last matching replace. A go.work replace wins over a
// go.mod replace. An exact version wins over a replace of every version.
func (l *loader) resolve(path, ver string) (cachePath, version, dir string) {
	if rep, ok := chooseReplace(path, ver, l.workRepls); ok {
		return replacedCoords(path, rep)
	}
	if rep, ok := chooseReplace(path, ver, l.repls); ok {
		return replacedCoords(path, rep)
	}
	return path, ver, ""
}

func chooseReplace(path, ver string, repls []modReplace) (modReplace, bool) {
	var wild, exact modReplace
	haveWild, haveExact := false, false
	for _, rep := range repls {
		if rep.oldPath != path {
			continue
		}
		if rep.oldVer == "" {
			wild = rep
			haveWild = true
			continue
		}
		if rep.oldVer == ver {
			exact = rep
			haveExact = true
		}
	}
	if haveExact {
		return exact, true
	}
	if haveWild {
		return wild, true
	}
	return modReplace{}, false
}

func replacedCoords(path string, rep modReplace) (cachePath, version, dir string) {
	if rep.local != "" {
		return path, "", rep.local
	}
	return rep.newPath, rep.newVer, ""
}

func replacesFrom(base string, reps []*modfile.Replace) []modReplace {
	out := make([]modReplace, 0, len(reps))
	for _, rep := range reps {
		if rep == nil || rep.Old.Path == "" {
			continue
		}
		next := modReplace{oldPath: rep.Old.Path, oldVer: rep.Old.Version}
		if rep.New.Version == "" {
			local := rep.New.Path
			if !filepath.IsAbs(local) {
				local = filepath.Join(base, local)
			}
			next.local = filepath.Clean(local)
		} else {
			next.newPath = rep.New.Path
			next.newVer = rep.New.Version
		}
		out = append(out, next)
	}
	return out
}

func excludesFrom(files ...*modfile.File) map[module.Version]struct{} {
	out := map[module.Version]struct{}{}
	for _, file := range files {
		if file == nil {
			continue
		}
		for _, excluded := range file.Exclude {
			if excluded == nil || excluded.Mod.Path == "" {
				continue
			}
			out[excluded.Mod] = struct{}{}
		}
	}
	return out
}

func declaredLang(file *modfile.File) string {
	if file == nil || file.Go == nil || file.Go.Version == "" {
		return "go1.27"
	}
	return langTag(file.Go.Version)
}

// moduleUnpruned reports whether file's go line is older than 1.17.
// A missing go line is Go 1.16, matching the go command.
func moduleUnpruned(file *modfile.File) bool {
	goVersion := "1.16"
	if file != nil && file.Go != nil && file.Go.Version != "" {
		goVersion = file.Go.Version
	}
	return version.Compare("go"+goVersion, "go1.17") < 0
}

// keptRequires returns requirements that participate in the build list.
// Excluded versions and requirements on a workspace module are omitted.
func (l *loader) keptRequires(file *modfile.File) []module.Version {
	if file == nil {
		return nil
	}
	reqs := make([]module.Version, 0, len(file.Require))
	for _, requirement := range file.Require {
		if requirement == nil || requirement.Mod.Path == "" || requirement.Mod.Version == "" {
			continue
		}
		if l.mains[requirement.Mod.Path] {
			continue
		}
		if _, excluded := l.excluded[requirement.Mod]; excluded {
			continue
		}
		reqs = append(reqs, requirement.Mod)
	}
	return reqs
}

// note records path at ver when ver is newer than the selected version.
func (l *loader) note(path, ver string) {
	if path == "" || ver == "" || l.mains[path] {
		return
	}
	for i := range l.mods {
		if l.mods[i].path != path {
			continue
		}
		if semver.Compare(ver, l.mods[i].selected) <= 0 {
			return
		}
		cachePath, lookupVer, dir := l.resolve(path, ver)
		l.mods[i].selected = ver
		l.mods[i].cachePath = cachePath
		l.mods[i].version = lookupVer
		l.mods[i].dir = dir
		l.mods[i].lang = ""
		return
	}
	cachePath, lookupVer, dir := l.resolve(path, ver)
	l.mods = append(l.mods, modSrc{
		path:      path,
		cachePath: cachePath,
		version:   lookupVer,
		dir:       dir,
		selected:  ver,
	})
}

// walk loads the go.mod of m.
// unpruned follows every transitive requirement. A pruned visit of a
// module at go 1.17 or higher records that module's own requirements
// and does not read further.
func (l *loader) walk(m module.Version, unpruned bool, seen map[visit]struct{}) error {
	if m.Path == "" || m.Version == "" || m.Version == "none" || l.mains[m.Path] {
		return nil
	}
	key := visit{path: m.Path, version: m.Version, unpruned: unpruned}
	if _, ok := seen[key]; ok {
		return nil
	}
	seen[key] = struct{}{}
	summary, err := l.readRequires(m)
	if err != nil {
		return err
	}
	nextUnpruned := summary.unpruned
	if unpruned {
		nextUnpruned = true
	}
	for _, requirement := range summary.reqs {
		l.note(requirement.Path, requirement.Version)
		if unpruned || summary.unpruned {
			if err := l.walk(requirement, nextUnpruned, seen); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *loader) readRequires(m module.Version) (modSummary, error) {
	if summary, ok := l.summaries[m]; ok {
		return summary, nil
	}
	dir, err := l.versionDir(m.Path, m.Version)
	if err != nil {
		return modSummary{}, err
	}
	body, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			summary := modSummary{unpruned: true}
			l.summaries[m] = summary
			return summary, nil
		}
		return modSummary{}, err
	}
	parsed, err := modfile.ParseLax("go.mod", body, nil)
	if err != nil {
		return modSummary{}, err
	}
	summary := modSummary{
		reqs:     l.keptRequires(parsed),
		unpruned: moduleUnpruned(parsed),
	}
	l.summaries[m] = summary
	return summary, nil
}

func (l *loader) versionDir(path, ver string) (string, error) {
	cachePath, lookupVer, dir := l.resolve(path, ver)
	if dir == "" {
		resolved, err := moduleCacheDir(l.cache, cachePath, lookupVer)
		if err != nil {
			return "", err
		}
		dir = resolved
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		if lookupVer == "" {
			return "", fmt.Errorf("%w: %s", ErrMissing, path)
		}
		return "", fmt.Errorf("%w: %s@%s", ErrMissing, cachePath, lookupVer)
	}
	return dir, nil
}

func findWorkspace(modDir string) (string, error) {
	switch gowork := os.Getenv("GOWORK"); gowork {
	case "off":
		return "", nil
	case "", "auto":
		return findWorkspaceFile(modDir), nil
	default:
		if !filepath.IsAbs(gowork) {
			return "", fmt.Errorf("%w: GOWORK is not an absolute path", ErrWorkspace)
		}
		info, err := os.Stat(gowork)
		if err != nil || info.IsDir() {
			return "", fmt.Errorf("%w: %s", ErrWorkspace, gowork)
		}
		return gowork, nil
	}
}

func findWorkspaceFile(dir string) string {
	dir = filepath.Clean(dir)
	for {
		candidate := filepath.Join(dir, "go.work")
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

type workspaceModule struct {
	path string
	dir  string
	file *modfile.File
}

func (l *loader) buildWorkspace(workPath string) error {
	body, err := os.ReadFile(workPath)
	if err != nil {
		return err
	}
	work, err := modfile.ParseWork(workPath, body, nil)
	if err != nil {
		return err
	}
	workDir := filepath.Dir(workPath)
	modules, err := l.workspaceModules(workDir, work)
	if err != nil {
		return err
	}
	if !l.mains[l.module] {
		return fmt.Errorf("%w: %s is not in the workspace", ErrWorkspace, l.module)
	}
	l.workRepls = replacesFrom(workDir, work.Replace)
	for _, rep := range l.workRepls {
		if rep.oldVer == "" && l.mains[rep.oldPath] {
			return fmt.Errorf("%w: %s is replaced at all versions", ErrWorkspace, rep.oldPath)
		}
	}
	files := make([]*modfile.File, len(modules))
	for i, wm := range modules {
		files[i] = wm.file
		if wm.path == l.module {
			continue
		}
		l.mods = append(l.mods, modSrc{
			path:      wm.path,
			cachePath: wm.path,
			dir:       wm.dir,
			lang:      declaredLang(wm.file),
		})
	}
	l.excluded = excludesFrom(files...)
	l.repls, err = workspaceModReplaces(modules, l.workRepls)
	if err != nil {
		return err
	}
	seen := map[visit]struct{}{}
	roots := make([]module.Version, 0, len(modules))
	for _, wm := range modules {
		reqs := l.keptRequires(wm.file)
		root := module.Version{Path: wm.path}
		l.summaries[root] = modSummary{reqs: reqs, unpruned: moduleUnpruned(wm.file)}
		roots = append(roots, root)
		for _, requirement := range reqs {
			l.note(requirement.Path, requirement.Version)
			if err := l.walk(requirement, moduleUnpruned(wm.file), seen); err != nil {
				return err
			}
		}
	}
	return l.promoteWorkspace(roots, seen)
}

func (l *loader) workspaceModules(workDir string, work *modfile.WorkFile) ([]workspaceModule, error) {
	l.mains = map[string]bool{}
	modules := make([]workspaceModule, 0, len(work.Use))
	for _, use := range work.Use {
		if use == nil || use.Path == "" {
			return nil, fmt.Errorf("%w: empty use path", ErrWorkspace)
		}
		dir := use.Path
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(workDir, dir)
		}
		dir = filepath.Clean(dir)
		file, err := readWorkspaceMod(dir)
		if err != nil {
			return nil, err
		}
		path := file.Module.Mod.Path
		if l.mains[path] {
			return nil, fmt.Errorf("%w: %s appears twice", ErrWorkspace, path)
		}
		l.mains[path] = true
		modules = append(modules, workspaceModule{path: path, dir: dir, file: file})
	}
	return modules, nil
}

func readWorkspaceMod(dir string) (*modfile.File, error) {
	path := filepath.Join(dir, "go.mod")
	body, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrWorkspace, dir)
		}
		return nil, err
	}
	file, err := modfile.Parse(path, body, nil)
	if err != nil {
		return nil, err
	}
	if file.Module == nil || file.Module.Mod.Path == "" {
		return nil, fmt.Errorf("%w: module path", ErrNoModule)
	}
	return file, nil
}

func workspaceModReplaces(modules []workspaceModule, workRepls []modReplace) ([]modReplace, error) {
	var repls []modReplace
	for _, wm := range modules {
		for _, rep := range replacesFrom(wm.dir, wm.file.Replace) {
			if coveredByWork(rep, workRepls) {
				continue
			}
			var err error
			repls, err = addReplace(repls, rep)
			if err != nil {
				return nil, err
			}
		}
	}
	return repls, nil
}

func coveredByWork(rep modReplace, workRepls []modReplace) bool {
	for _, work := range workRepls {
		if work.oldPath != rep.oldPath {
			continue
		}
		if work.oldVer == "" || work.oldVer == rep.oldVer {
			return true
		}
	}
	return false
}

func addReplace(repls []modReplace, rep modReplace) ([]modReplace, error) {
	for _, old := range repls {
		if old.oldPath != rep.oldPath || old.oldVer != rep.oldVer {
			continue
		}
		if old.local == rep.local && old.newPath == rep.newPath && old.newVer == rep.newVer {
			return repls, nil
		}
		return nil, fmt.Errorf("%w: conflicting replacements for %s", ErrWorkspace, rep.oldPath)
	}
	return append(repls, rep), nil
}

// promoteWorkspace loads a selected dependency that is newer than the
// version required by an already loaded module. Workspace pruning still
// applies to that dependency's own requirements.
func (l *loader) promoteWorkspace(roots []module.Version, seen map[visit]struct{}) error {
	has := make(map[string]bool, len(roots))
	for _, root := range roots {
		has[root.Path] = true
	}
	loaded := map[module.Version]bool{}
	for {
		queue := make([]module.Version, 0)
		queued := map[module.Version]bool{}
		for path := range has {
			selected := module.Version{Path: path, Version: l.selectedVersion(path)}
			if !loaded[selected] {
				if !queued[selected] {
					queue = append(queue, selected)
					queued[selected] = true
				}
				continue
			}
			summary, ok := l.summaries[selected]
			if !ok {
				continue
			}
			for _, requirement := range summary.reqs {
				chosen := l.selectedVersion(requirement.Path)
				if semver.Compare(chosen, requirement.Version) <= 0 {
					continue
				}
				upgraded := module.Version{Path: requirement.Path, Version: chosen}
				if !loaded[upgraded] && !queued[upgraded] {
					queue = append(queue, upgraded)
					queued[upgraded] = true
				}
			}
		}
		if len(queue) == 0 {
			return nil
		}
		for _, m := range queue {
			loaded[m] = true
			has[m.Path] = true
			if err := l.walkDirect(m, seen); err != nil {
				return err
			}
		}
	}
}

func (l *loader) selectedVersion(path string) string {
	for i := range l.mods {
		if l.mods[i].path != path {
			continue
		}
		if l.mains[path] {
			return ""
		}
		return l.mods[i].selected
	}
	return "none"
}

// walkDirect loads the direct requirements of an upgraded module.
// Their own pruning decides whether the walk continues.
func (l *loader) walkDirect(m module.Version, seen map[visit]struct{}) error {
	if m.Path == "" || m.Version == "" || m.Version == "none" || l.mains[m.Path] {
		return nil
	}
	summary, err := l.readRequires(m)
	if err != nil {
		return err
	}
	for _, requirement := range summary.reqs {
		l.note(requirement.Path, requirement.Version)
		if err := l.walk(requirement, summary.unpruned, seen); err != nil {
			return err
		}
	}
	return nil
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
