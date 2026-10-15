package golang

import (
	"bytes"
	"fmt"
	"go/format"
	"go/printer"
	"go/token"
	"io"
	"strconv"
	"strings"
	"unicode"
)

// File is one Go source file: a package name, imports, and declarations.
type File struct {
	Name    string
	Imports []*Import
	Decls   []Decl
}

// Import is one import. Name is the explicit local name, including
// "." and "_". An empty Name uses the path's base name.
type Import struct {
	Name string
	Path string
}

// NewFile returns an empty file for package name.
func NewFile(name string) *File {
	return &File{
		Name:    name,
		Imports: []*Import{},
		Decls:   []Decl{},
	}
}

// Import adds an import and returns it. [Import.Alias] sets a local name.
func (f *File) Import(path string) *Import {
	im := &Import{Path: path}
	if f != nil {
		f.Imports = append(f.Imports, im)
	}
	return im
}

// Alias sets the import's local name.
func (im *Import) Alias(name string) *Import {
	if im != nil {
		im.Name = name
	}
	return im
}

// Local is the name used in selectors for this import.
func (im *Import) Local() string {
	if im == nil {
		return ""
	}
	if im.Name != "" {
		return im.Name
	}
	return defaultAlias(im.Path)
}

// Func appends a function and returns it.
func (f *File) Func() *Func {
	fn := &Func{file: f, Parameters: []*Field{}, Returns: []*Field{}}
	if f != nil {
		f.Decls = append(f.Decls, fn)
	}
	return fn
}

// Type appends a type declaration and returns it.
func (f *File) Type() *TypeDecl {
	d := &TypeDecl{file: f}
	if f != nil {
		f.Decls = append(f.Decls, d)
	}
	return d
}

// Var appends a var declaration and returns it.
func (f *File) Var() *Var {
	v := &Var{file: f}
	if f != nil {
		f.Decls = append(f.Decls, v)
	}
	return v
}

// Const appends a const declaration and returns it.
func (f *File) Const() *Const {
	c := &Const{file: f}
	if f != nil {
		f.Decls = append(f.Decls, c)
	}
	return c
}

// Take reads Go source into a [File].
func Take(r io.Reader) (*File, error) {
	if r == nil {
		return nil, errNil
	}
	src, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errRead, err)
	}
	return take(src)
}

// Make writes the file as Go source.
func (f *File) Make(w io.Writer) error {
	if f == nil || w == nil {
		return errNil
	}
	f.ensureImports()
	node, err := (&builder{file: f}).build(f)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, token.NewFileSet(), node); err != nil {
		return fmt.Errorf("%w: %w", errPrint, err)
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("%w: %w", errPrint, err)
	}
	if _, err := w.Write(formatted); err != nil {
		return fmt.Errorf("%w: %w", errPrint, err)
	}
	return nil
}

func (f *File) ensureImports() {
	if f == nil {
		return
	}
	for _, decl := range f.Decls {
		f.walkDecl(decl)
	}
}

func (f *File) use(path string) {
	if f == nil || path == "" {
		return
	}
	for _, im := range f.Imports {
		if im != nil && im.Path == path {
			return
		}
	}
	alias := defaultAlias(path)
	name := ""
	if !validIdent(alias) || f.localTaken(alias) {
		name = f.fresh(alias)
	}
	f.Imports = append(f.Imports, &Import{Name: name, Path: path})
}

func (f *File) local(path string) string {
	for _, im := range f.Imports {
		if im != nil && im.Path == path {
			return im.Local()
		}
	}
	return defaultAlias(path)
}

func (f *File) localTaken(name string) bool {
	if token.IsKeyword(name) {
		return true
	}
	for _, im := range f.Imports {
		if im != nil && im.Local() == name {
			return true
		}
	}
	return false
}

func (f *File) fresh(base string) string {
	if !validIdent(base) {
		base = "pkg"
	}
	if !f.localTaken(base) {
		return base
	}
	for n := 2; ; n++ {
		cand := base + strconv.Itoa(n)
		if !f.localTaken(cand) {
			return cand
		}
	}
}

func defaultAlias(path string) string {
	path = strings.Trim(path, `"`)
	if path == "" {
		return "pkg"
	}
	base := path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		base = path[i+1:]
		if major(base) && i > 0 {
			rest := path[:i]
			if j := strings.LastIndex(rest, "/"); j >= 0 {
				base = rest[j+1:]
			} else {
				base = rest
			}
		}
	}
	base = strings.Map(func(r rune) rune {
		if r == '-' || r == '.' {
			return '_'
		}
		return r
	}, base)
	if !validIdent(base) {
		return "pkg"
	}
	return base
}

func major(s string) bool {
	if len(s) < 2 || (s[0] != 'v' && s[0] != 'V') {
		return false
	}
	for _, r := range s[1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func validIdent(name string) bool {
	if name == "" || name == "_" || name == "." {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if !unicode.IsLetter(r) && r != '_' {
				return false
			}
			continue
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return !token.IsKeyword(name)
}
