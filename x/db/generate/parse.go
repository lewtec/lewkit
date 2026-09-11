package generate

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type method struct {
	name    string
	params  []field
	results []field
}

type field struct {
	name string
	typ  typeExpr
}

type typeExpr struct {
	kind string // ident, qual, pointer, slice, ellipsis, any
	pkg  string
	name string
	elem *typeExpr
}

func (t typeExpr) equal(o typeExpr) bool {
	if t.kind != o.kind || t.pkg != o.pkg || t.name != o.name {
		return false
	}
	if t.elem == nil && o.elem == nil {
		return true
	}
	if t.elem == nil || o.elem == nil {
		return false
	}
	return t.elem.equal(*o.elem)
}

func (t typeExpr) local() bool {
	switch t.kind {
	case "ident":
		return !builtin(t.name)
	case "pointer", "slice", "ellipsis":
		return t.elem != nil && t.elem.local()
	default:
		return false
	}
}

func (t typeExpr) baseName() string {
	if t.elem != nil {
		return t.elem.baseName()
	}
	return t.name
}

func builtin(name string) bool {
	switch name {
	case "bool", "byte", "rune", "string", "error", "any",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64", "complex64", "complex128":
		return true
	default:
		return false
	}
}

type structType struct {
	name   string
	fields []field
}

func loadGenerated(root string, engines []engine) (map[string][]method, map[string][]structType, error) {
	methods := map[string][]method{}
	types := map[string][]structType{}
	for _, e := range engines {
		dir := filepath.Join(root, e.dir)
		ms, ts, err := parsePkg(dir)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", e.dir, err)
		}
		methods[e.dir] = ms
		types[e.dir] = ts
	}
	return methods, types, nil
}

func parsePkg(dir string) ([]method, []structType, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(info os.FileInfo) bool {
		n := info.Name()
		return strings.HasSuffix(n, ".go") && !strings.HasSuffix(n, "_test.go") && !strings.HasPrefix(n, "zz_")
	}, 0)
	if err != nil {
		return nil, nil, err
	}
	var methods []method
	var types []structType
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.FuncDecl:
					if recvName(x) == "Queries" && x.Name.IsExported() && x.Name.Name != "WithTx" {
						methods = append(methods, parseMethod(x))
					}
				case *ast.GenDecl:
					if x.Tok != token.TYPE {
						return true
					}
					for _, spec := range x.Specs {
						ts, ok := spec.(*ast.TypeSpec)
						if !ok {
							continue
						}
						st, ok := ts.Type.(*ast.StructType)
						if !ok {
							continue
						}
						types = append(types, structType{name: ts.Name.Name, fields: structFields(st)})
					}
				}
				return true
			})
		}
	}
	sort.Slice(methods, func(i, j int) bool { return methods[i].name < methods[j].name })
	sort.Slice(types, func(i, j int) bool { return types[i].name < types[j].name })
	return methods, types, nil
}

func recvName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	t := fn.Recv.List[0].Type
	if s, ok := t.(*ast.StarExpr); ok {
		t = s.X
	}
	id, ok := t.(*ast.Ident)
	if !ok {
		return ""
	}
	return id.Name
}

func parseMethod(fn *ast.FuncDecl) method {
	m := method{name: fn.Name.Name}
	if fn.Type.Params != nil {
		for _, p := range fn.Type.Params.List {
			t := parseExpr(p.Type)
			if len(p.Names) == 0 {
				m.params = append(m.params, field{typ: t})
				continue
			}
			for _, n := range p.Names {
				m.params = append(m.params, field{name: n.Name, typ: t})
			}
		}
	}
	if fn.Type.Results != nil {
		for _, p := range fn.Type.Results.List {
			t := parseExpr(p.Type)
			if len(p.Names) == 0 {
				m.results = append(m.results, field{typ: t})
				continue
			}
			for _, n := range p.Names {
				m.results = append(m.results, field{name: n.Name, typ: t})
			}
		}
	}
	return m
}

func structFields(st *ast.StructType) []field {
	var out []field
	if st.Fields == nil {
		return out
	}
	for _, f := range st.Fields.List {
		t := parseExpr(f.Type)
		for _, n := range f.Names {
			out = append(out, field{name: n.Name, typ: t})
		}
	}
	return out
}

func parseExpr(e ast.Expr) typeExpr {
	switch t := e.(type) {
	case *ast.Ident:
		return typeExpr{kind: "ident", name: t.Name}
	case *ast.SelectorExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return typeExpr{kind: "qual", pkg: id.Name, name: t.Sel.Name}
		}
	case *ast.StarExpr:
		inner := parseExpr(t.X)
		return typeExpr{kind: "pointer", elem: &inner}
	case *ast.ArrayType:
		inner := parseExpr(t.Elt)
		return typeExpr{kind: "slice", elem: &inner}
	case *ast.Ellipsis:
		inner := parseExpr(t.Elt)
		return typeExpr{kind: "ellipsis", elem: &inner}
	case *ast.InterfaceType:
		return typeExpr{kind: "any"}
	}
	return typeExpr{kind: "ident", name: "any"}
}

func compareMethods(all map[string][]method) error {
	if len(all) < 2 {
		return nil
	}
	var names []string
	for k := range all {
		names = append(names, k)
	}
	sort.Strings(names)
	idx := map[string]method{}
	for _, m := range all[names[0]] {
		idx[m.name] = m
	}
	for _, eng := range names[1:] {
		seen := map[string]bool{}
		for _, m := range all[eng] {
			seen[m.name] = true
			b, ok := idx[m.name]
			if !ok {
				return fmt.Errorf("%w: %s in %s not %s", errMethodMismatch, m.name, eng, names[0])
			}
			if !methodEqual(b, m) {
				return fmt.Errorf("%w: %s in %s and %s", errMethodMismatch, m.name, names[0], eng)
			}
		}
		for n := range idx {
			if !seen[n] {
				return fmt.Errorf("%w: %s in %s not %s", errMethodMismatch, n, names[0], eng)
			}
		}
	}
	return nil
}

func methodEqual(a, b method) bool {
	if a.name != b.name || len(a.params) != len(b.params) || len(a.results) != len(b.results) {
		return false
	}
	for i := range a.params {
		if !a.params[i].typ.equal(b.params[i].typ) {
			return false
		}
	}
	for i := range a.results {
		if !a.results[i].typ.equal(b.results[i].typ) {
			return false
		}
	}
	return true
}

func compareStructs(all map[string][]structType, methods map[string][]method) error {
	need := localNames(firstMethods(methods))
	if len(all) < 2 {
		return nil
	}
	var names []string
	for k := range all {
		names = append(names, k)
	}
	sort.Strings(names)
	by := map[string]map[string]structType{}
	for eng, list := range all {
		by[eng] = map[string]structType{}
		for _, s := range list {
			by[eng][s.name] = s
		}
	}
	base := by[names[0]]
	for _, name := range need {
		a, ok := base[name]
		if !ok {
			continue
		}
		for _, eng := range names[1:] {
			b, ok := by[eng][name]
			if !ok {
				return fmt.Errorf("%w: %s in %s not %s", errTypeMismatch, name, names[0], eng)
			}
			if !fieldsEqual(a.fields, b.fields) {
				return fmt.Errorf("%w: %s in %s and %s", errTypeMismatch, name, names[0], eng)
			}
		}
	}
	return nil
}

func fieldsEqual(a, b []field) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].name != b[i].name || !a[i].typ.equal(b[i].typ) {
			return false
		}
	}
	return true
}

func localNames(methods []method) []string {
	seen := map[string]bool{}
	var out []string
	add := func(t typeExpr) {
		if !t.local() {
			return
		}
		n := t.baseName()
		if seen[n] {
			return
		}
		seen[n] = true
		out = append(out, n)
	}
	for _, m := range methods {
		for _, p := range m.params {
			add(p.typ)
		}
		for _, r := range m.results {
			add(r.typ)
		}
	}
	sort.Strings(out)
	return out
}
