// Package load reads a module and selects the Go files to compile.
//
// It does not run the go command and it does not download modules.
// Imports outside the main module resolve through the go.mod build list
// into GOMODCACHE, or into a replace directory. Standard-library imports
// are names; the archives themselves come from [Archives].
// A package with a matching .s or .syso file has [Package.Asm] set and
// lists those paths in [Package.AsmFiles]. Those objects are not produced
// here.
package load

import (
	"errors"
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/mod/modfile"
)

var (
	// ErrNoModule means no go.mod was found from the build directory.
	ErrNoModule = errors.New("load: no go.mod")
	// ErrNoPackage means a directory has no buildable Go files.
	ErrNoPackage = errors.New("load: no go files")
	// ErrCgo means the package has no buildable files except cgo.
	ErrCgo = errors.New("load: cgo is not available")
	// ErrExternal means an import is outside this module's build list and the standard library.
	ErrExternal = errors.New("load: import is outside this module")
	// ErrMissing means a required module is not in GOMODCACHE.
	ErrMissing = errors.New("load: module is not in the cache")
	// ErrEmbed means a //go:embed pattern did not resolve.
	ErrEmbed = errors.New("load: bad embed")
	// ErrMixed means .go files were combined with package paths.
	ErrMixed = errors.New("load: cannot mix .go files with packages")
	// ErrNoArchive means an archive is not in the pkgdir.
	ErrNoArchive = errors.New("load: archive is missing")
	// ErrLdflags means -ldflags is quoted incorrectly.
	ErrLdflags = errors.New("load: bad ldflags")
)

// Embed is the compiler's -embedcfg document for one package.
type Embed struct {
	Patterns map[string][]string
	Files    map[string]string
}

// Package is one package in the module graph.
type Package struct {
	ImportPath string
	Name       string
	Dir        string
	Files      []string
	Imports    []string
	// Deps are the in-graph imports. Standard-library imports are not listed.
	Deps []string
	// Asm means a matching .s or .syso file is part of this package.
	Asm bool
	// AsmFiles are the matching .s and .syso paths. The compiler does
	// not assemble them.
	AsmFiles []string
	// Lang is the -lang value from the module's go line, such as go1.27.
	Lang string
	// Embed is nil when the package has no //go:embed patterns.
	Embed *Embed
}

// Graph is the packages reachable from the requested patterns.
type Graph struct {
	// Dir is the absolute directory the patterns were resolved from.
	Dir    string
	Module string
	Pkgs   []Package
}

// Load selects packages under the module that contains dir.
// An empty pattern list is the package in dir. A pattern ending in
// "..." walks that tree. A list of .go files is one package.
// Files are chosen for this process's GOOS and GOARCH.
func Load(dir string, patterns, tags []string) (*Graph, error) {
	return LoadTarget(dir, patterns, tags, runtime.GOOS, runtime.GOARCH)
}

// LoadTarget is [Load] for one GOOS and GOARCH. Empty values use this process.
func LoadTarget(dir string, patterns, tags []string, goos, goarch string) (*Graph, error) {
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	if dir == "" {
		dir = "."
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	modDir, mf, err := findModule(abs)
	if err != nil {
		return nil, err
	}
	cache, err := modCache()
	if err != nil {
		return nil, err
	}
	l := &loader{
		modDir: modDir,
		module: mf.Module.Mod.Path,
		cache:  cache,
		seen:   map[string]*Package{},
		bctx: build.Context{
			GOOS:        goos,
			GOARCH:      goarch,
			BuildTags:   append([]string(nil), tags...),
			CgoEnabled:  false,
			Compiler:    "gc",
			ReleaseTags: build.Default.ReleaseTags,
		},
	}
	if err := l.buildList(mf); err != nil {
		return nil, err
	}
	dirs, files, err := l.patterns(abs, patterns)
	if err != nil {
		return nil, err
	}
	if len(files) > 0 {
		if err := l.loadFiles(files); err != nil {
			return nil, err
		}
	}
	for _, d := range dirs {
		imp, err := l.pathForDir(d)
		if err != nil {
			return nil, err
		}
		if _, err := l.loadDir(d, imp); err != nil {
			return nil, err
		}
	}
	g := &Graph{Dir: abs, Module: l.module}
	for _, path := range l.order {
		g.Pkgs = append(g.Pkgs, *l.seen[path])
	}
	return g, nil
}

// Archives reads pkgdir. Each file import/path.a is the archive for
// that import path. An empty pkgdir is GOROOT/pkg/GOOS_GOARCH.
// runtime must be present.
func Archives(pkgdir string) (map[string]string, error) {
	if pkgdir == "" {
		pkgdir = filepath.Join(runtime.GOROOT(), "pkg", runtime.GOOS+"_"+runtime.GOARCH)
	}
	abs, err := filepath.Abs(pkgdir)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	err = filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".a") {
			return nil
		}
		rel, err := filepath.Rel(abs, path)
		if err != nil {
			return err
		}
		imp := filepath.ToSlash(strings.TrimSuffix(rel, ".a"))
		out[imp] = path
		return nil
	})
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrNoArchive, abs)
		}
		return nil, err
	}
	if _, ok := out["runtime"]; !ok {
		return nil, fmt.Errorf("%w: runtime", ErrNoArchive)
	}
	return out, nil
}

// SplitTags splits a -tags value on commas and spaces.
// A blank value is nil. Empty fields are dropped.
func SplitTags(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var tags []string
	for _, part := range strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t'
	}) {
		if part != "" {
			tags = append(tags, part)
		}
	}
	return tags
}

// SplitFlags splits a -ldflags value the way the go command does.
// Single or double quotes group a field. Quotes are not nested.
func SplitFlags(s string) ([]string, error) {
	var f []string
	for len(s) > 0 {
		for len(s) > 0 && isSpace(s[0]) {
			s = s[1:]
		}
		if len(s) == 0 {
			break
		}
		if s[0] == '"' || s[0] == '\'' {
			quote := s[0]
			s = s[1:]
			i := strings.IndexByte(s, quote)
			if i < 0 {
				return nil, fmt.Errorf("%w: unterminated %c string", ErrLdflags, quote)
			}
			f = append(f, s[:i])
			s = s[i+1:]
			continue
		}
		i := 0
		for i < len(s) && !isSpace(s[i]) {
			i++
		}
		f = append(f, s[:i])
		s = s[i:]
	}
	return f, nil
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

type loader struct {
	modDir string
	module string
	cache  string
	mods   []modSrc
	seen   map[string]*Package
	order  []string
	bctx   build.Context
}

func (l *loader) patterns(base string, patterns []string) (dirs, files []string, err error) {
	if len(patterns) == 0 {
		return []string{base}, nil, nil
	}
	for _, p := range patterns {
		if strings.HasSuffix(p, ".go") {
			if !filepath.IsAbs(p) {
				p = filepath.Join(base, p)
			}
			files = append(files, p)
			continue
		}
		if len(files) > 0 {
			return nil, nil, ErrMixed
		}
		d, walk, err := l.one(base, p)
		if err != nil {
			return nil, nil, err
		}
		if !walk {
			dirs = append(dirs, d)
			continue
		}
		found, err := l.walkPkgs(d)
		if err != nil {
			return nil, nil, err
		}
		dirs = append(dirs, found...)
	}
	if len(files) > 0 && len(dirs) > 0 {
		return nil, nil, ErrMixed
	}
	return dirs, files, nil
}

func (l *loader) one(base, pattern string) (string, bool, error) {
	walk := false
	p := pattern
	if p == "..." || strings.HasSuffix(p, "/...") || strings.HasSuffix(p, string(filepath.Separator)+"...") {
		walk = true
		p = strings.TrimSuffix(p, "...")
		p = strings.TrimRight(p, `/\`)
		if p == "" || p == "." {
			return base, true, nil
		}
	}
	if p == "" || p == "." {
		return base, walk, nil
	}
	if filepath.IsAbs(p) || strings.HasPrefix(p, "./") || strings.HasPrefix(p, `.\`) {
		abs, err := filepath.Abs(p)
		if strings.HasPrefix(p, ".") {
			abs, err = filepath.Abs(filepath.Join(base, p))
		}
		if err != nil {
			return "", false, err
		}
		return abs, walk, nil
	}
	if st, err := os.Stat(filepath.Join(base, filepath.FromSlash(p))); err == nil && st.IsDir() {
		return filepath.Join(base, filepath.FromSlash(p)), walk, nil
	}
	dir, err := l.dirForImport(p)
	if err != nil {
		return "", false, err
	}
	return dir, walk, nil
}

func (l *loader) walkPkgs(root string) ([]string, error) {
	var dirs []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if path != root && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "testdata" || name == "vendor") {
			return filepath.SkipDir
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, e := range entries {
			n := e.Name()
			if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
				continue
			}
			ok, err := l.bctx.MatchFile(path, n)
			if err != nil {
				return err
			}
			if ok {
				dirs = append(dirs, path)
				break
			}
		}
		return nil
	})
	return dirs, err
}

func (l *loader) loadFiles(files []string) error {
	abs := make([]string, 0, len(files))
	var dir string
	for _, f := range files {
		a, err := filepath.Abs(f)
		if err != nil {
			return err
		}
		d := filepath.Dir(a)
		if dir == "" {
			dir = d
		} else if d != dir {
			return fmt.Errorf("%w: %s", ErrMixed, a)
		}
		abs = append(abs, a)
	}
	imp, err := l.importPath(dir)
	if err != nil {
		return err
	}
	sort.Strings(abs)
	var name string
	imports := map[string]struct{}{}
	fset := token.NewFileSet()
	for _, file := range abs {
		f, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		if f.Name == nil {
			return fmt.Errorf("%w: %s", ErrNoPackage, file)
		}
		if name == "" {
			name = f.Name.Name
		} else if f.Name.Name != name {
			return fmt.Errorf("load: %s is package %s, want %s", file, f.Name.Name, name)
		}
		for _, spec := range f.Imports {
			path, err := strconvUnquote(spec.Path.Value)
			if err != nil {
				return err
			}
			if path == "C" {
				return fmt.Errorf("%w: %s", ErrCgo, file)
			}
			if strings.HasPrefix(path, ".") {
				return fmt.Errorf("%w: %s", ErrExternal, path)
			}
			imports[path] = struct{}{}
		}
	}
	asm, err := l.asmFiles(dir)
	if err != nil {
		return err
	}
	p := &Package{
		ImportPath: imp,
		Name:       name,
		Dir:        dir,
		Files:      abs,
		Asm:        len(asm) > 0,
		AsmFiles:   asm,
		Lang:       l.langFor(imp),
	}
	for path := range imports {
		p.Imports = append(p.Imports, path)
	}
	sort.Strings(p.Imports)
	_, err = l.finish(p)
	return err
}

func (l *loader) loadDir(dir, imp string) (*Package, error) {
	if p, ok := l.seen[imp]; ok {
		return p, nil
	}
	bp, ierr := l.bctx.ImportDir(dir, 0)
	if ierr != nil {
		var noGo *build.NoGoError
		if !errors.As(ierr, &noGo) {
			return nil, ierr
		}
	}
	if bp == nil {
		return nil, fmt.Errorf("%w: %s", ErrNoPackage, dir)
	}
	if len(bp.GoFiles) == 0 {
		cgo, err := l.importsC(dir)
		if err != nil {
			return nil, err
		}
		if cgo {
			return nil, fmt.Errorf("%w: %s", ErrCgo, dir)
		}
		if len(bp.SFiles) == 0 && len(bp.SysoFiles) == 0 {
			return nil, fmt.Errorf("%w: %s", ErrNoPackage, dir)
		}
	}
	files := make([]string, 0, len(bp.GoFiles))
	for _, name := range bp.GoFiles {
		files = append(files, filepath.Join(dir, name))
	}
	sort.Strings(files)
	emb, err := resolveEmbed(dir, bp.EmbedPatterns)
	if err != nil {
		return nil, err
	}
	imports := append([]string(nil), bp.Imports...)
	sort.Strings(imports)
	asmFiles := make([]string, 0, len(bp.SFiles)+len(bp.SysoFiles))
	for _, name := range bp.SFiles {
		asmFiles = append(asmFiles, filepath.Join(dir, name))
	}
	for _, name := range bp.SysoFiles {
		asmFiles = append(asmFiles, filepath.Join(dir, name))
	}
	sort.Strings(asmFiles)
	p := &Package{
		ImportPath: imp,
		Name:       bp.Name,
		Dir:        dir,
		Files:      files,
		Imports:    imports,
		Asm:        len(asmFiles) > 0,
		AsmFiles:   asmFiles,
		Lang:       l.langFor(imp),
		Embed:      emb,
	}
	return l.finish(p)
}

func (l *loader) finish(p *Package) (*Package, error) {
	if prev, ok := l.seen[p.ImportPath]; ok {
		return prev, nil
	}
	l.seen[p.ImportPath] = p
	for _, path := range p.Imports {
		if path == "unsafe" || isStd(path) {
			continue
		}
		dir, err := l.dirForImport(path)
		if err != nil {
			return nil, err
		}
		if _, err := l.loadDir(dir, path); err != nil {
			return nil, err
		}
		p.Deps = append(p.Deps, path)
	}
	l.order = append(l.order, p.ImportPath)
	return p, nil
}

func (l *loader) asmFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(name)
		if ext != ".s" && ext != ".syso" {
			continue
		}
		ok, err := l.bctx.MatchFile(dir, name)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, filepath.Join(dir, name))
		}
	}
	sort.Strings(out)
	return out, nil
}

func (l *loader) importsC(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		ok, err := l.bctx.MatchFile(dir, name)
		if err != nil {
			return false, err
		}
		if !ok {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ImportsOnly)
		if err != nil {
			return false, err
		}
		for _, spec := range f.Imports {
			path, err := strconvUnquote(spec.Path.Value)
			if err != nil {
				return false, err
			}
			if path == "C" {
				return true, nil
			}
		}
	}
	return false, nil
}

func (l *loader) pathForDir(dir string) (string, error) {
	if imp, err := l.importPath(dir); err == nil {
		return imp, nil
	}
	var best *modSrc
	for i := range l.mods {
		m := &l.mods[i]
		if m.dir == "" {
			continue
		}
		if dir == m.dir || strings.HasPrefix(dir, m.dir+string(filepath.Separator)) {
			if best == nil || len(m.dir) > len(best.dir) {
				best = m
			}
		}
	}
	if best == nil {
		return "", fmt.Errorf("%w: %s", ErrExternal, dir)
	}
	rel, err := filepath.Rel(best.dir, dir)
	if err != nil {
		return "", err
	}
	if rel == "." {
		return best.path, nil
	}
	return best.path + "/" + filepath.ToSlash(rel), nil
}

func (l *loader) importPath(dir string) (string, error) {
	rel, err := filepath.Rel(l.modDir, dir)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %s", ErrExternal, dir)
	}
	if rel == "." {
		return l.module, nil
	}
	return l.module + "/" + filepath.ToSlash(rel), nil
}

func isStd(path string) bool {
	first, _, _ := strings.Cut(path, "/")
	return !strings.Contains(first, ".")
}

func findModule(dir string) (string, *modfile.File, error) {
	for {
		path := filepath.Join(dir, "go.mod")
		body, err := os.ReadFile(path)
		if err == nil {
			f, err := modfile.Parse(path, body, nil)
			if err != nil {
				return "", nil, err
			}
			if f.Module == nil || f.Module.Mod.Path == "" {
				return "", nil, fmt.Errorf("%w: module path", ErrNoModule)
			}
			return dir, f, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", nil, err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil, ErrNoModule
		}
		dir = parent
	}
}

func strconvUnquote(s string) (string, error) {
	if len(s) < 2 {
		return "", fmt.Errorf("load: bad import %q", s)
	}
	quote := s[0]
	if quote != s[len(s)-1] || (quote != '"' && quote != '`') {
		return "", fmt.Errorf("load: bad import %q", s)
	}
	inner := s[1 : len(s)-1]
	if quote == '`' {
		return inner, nil
	}
	var b strings.Builder
	for i := 0; i < len(inner); i++ {
		if inner[i] != '\\' {
			b.WriteByte(inner[i])
			continue
		}
		if i+1 >= len(inner) {
			return "", fmt.Errorf("load: bad import %q", s)
		}
		i++
		switch inner[i] {
		case '\\', '"':
			b.WriteByte(inner[i])
		default:
			return "", fmt.Errorf("load: bad import %q", s)
		}
	}
	if strings.IndexFunc(b.String(), unicode.IsSpace) >= 0 {
		return "", fmt.Errorf("load: bad import %q", s)
	}
	return b.String(), nil
}
