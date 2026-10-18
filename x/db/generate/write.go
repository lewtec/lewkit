package generate

import (
	"bytes"
	"path"
	"path/filepath"

	xgenerate "github.com/lewtec/lewkit/x/generate"
	"github.com/lewtec/lewkit/x/takemake/golang"
)

var generatedBy = xgenerate.Notice("db")

const dbPath = "github.com/lewtec/lewkit/x/db"

type out struct {
	dir, pkg, imp string
	engines       []engine
	methods       []method
	structs       []structType
}

func write(o out) error {
	if err := writeFS(o); err != nil {
		return err
	}
	return writeQueries(o)
}

func writeFS(o out) error {
	file := golang.NewFile(o.pkg)
	file.Header = "// " + generatedBy
	var dirs []string
	for _, e := range o.engines {
		dirs = append(dirs, e.dir)
	}
	file.Var().Comment(joinEmbed(dirs)).Id("FS").Type(golang.QualType("embed", "FS"))
	return save(file, filepath.Join(o.dir, "zz_fs.go"))
}

func writeQueries(o out) error {
	file := golang.NewFile(o.pkg)
	file.Header = "// " + generatedBy
	for _, e := range o.engines {
		file.Import(path.Join(dbPath, e.dir)).Alias("_")
	}

	need := map[string]bool{}
	for _, n := range localNames(o.methods) {
		need[n] = true
	}
	for _, st := range o.structs {
		if !need[st.name] {
			continue
		}
		fields := make([]*golang.Field, 0, len(st.fields))
		for _, fl := range st.fields {
			fields = append(fields, golang.NewField(goType(fl.typ), fl.name))
		}
		file.Type().Id(st.name).Struct(fields...)
	}

	methods := make([]*golang.Field, 0, len(o.methods))
	for _, m := range o.methods {
		methods = append(methods, golang.NewField(golang.FuncType(paramFields(m.params), paramFields(m.results)), m.name))
	}
	file.Type().Id("Queries").Interface(methods...)

	for _, e := range o.engines {
		writeWrap(file, o, e)
	}

	file.Func().Id("New").Params(
		golang.NewField(golang.Named("string"), "url"),
	).Results(
		golang.NewField(golang.FuncType(
			[]*golang.Field{golang.NewField(golang.QualType(dbPath, "DBTX"), "tx")},
			[]*golang.Field{golang.NewField(golang.Named("Queries"))},
		)),
	).Block(
		golang.Return(golang.FuncLit(
			[]*golang.Field{golang.NewField(golang.QualType(dbPath, "DBTX"), "tx")},
			[]*golang.Field{golang.NewField(golang.Named("Queries"))},
			golang.Switch(golang.Qual(dbPath, "Scheme").Call(golang.Id("url")), newCases(o.imp, o.engines)...),
		)),
	)

	file.Type().Id("DBArg").Struct(
		golang.NewField(golang.QualType(dbPath, "Arg").Inst(golang.Named("Queries"))),
	)
	file.Func().Recv("a", golang.Pointer(golang.Named("DBArg"))).Id("Open").Params(
		golang.NewField(golang.QualType("context", "Context"), "ctx"),
	).Results(
		golang.NewField(golang.Named("error")),
	).Block(
		golang.Define([]*golang.Expr{golang.Id("c")}, golang.Id("a").Sel("Value").Call()),
		golang.If(golang.Binary(golang.OpEql, golang.Id("c"), golang.Lit(nil)),
			golang.Return(golang.Qual("fmt", "Errorf").Call(golang.Lit("database url not set"))),
		),
		golang.Return(golang.Id("a").Sel("Arg").Sel("Open").Call(
			golang.Id("ctx"),
			golang.Id("FS"),
			golang.Id("New").Call(golang.Id("c").Sel("URL").Call()),
		)),
	)

	return save(file, filepath.Join(o.dir, "zz_queries.go"))
}

func save(file *golang.File, path string) error {
	var buf bytes.Buffer
	if err := file.Make(&buf); err != nil {
		return err
	}
	return xgenerate.WriteFile(path, buf.Bytes())
}

func newCases(imp string, engines []engine) []*golang.Stmt {
	cases := make([]*golang.Stmt, 0, len(engines)+1)
	for _, e := range engines {
		wrap := wrapName(e.dir)
		engImp := path.Join(imp, e.dir)
		cases = append(cases, golang.Case([]*golang.Expr{golang.Lit(e.dir)},
			golang.Return(golang.Composite(golang.Named(wrap),
				golang.KeyValue(golang.Id("q"), golang.Qual(engImp, "New").Call(golang.Id("tx"))),
			)),
		))
	}
	cases = append(cases, golang.Default(golang.Return(golang.Lit(nil))))
	return cases
}

func wrapName(dir string) string {
	return dir + "Wrap"
}

func writeWrap(file *golang.File, o out, e engine) {
	engImp := path.Join(o.imp, e.dir)
	wrap := wrapName(e.dir)
	file.Type().Id(wrap).Struct(golang.NewField(golang.Pointer(golang.QualType(engImp, "Queries")), "q"))
	for _, m := range o.methods {
		writeMethod(file, engImp, wrap, m)
	}
}

func writeMethod(file *golang.File, engImp, wrap string, m method) {
	args := make([]*golang.Expr, 0, len(m.params))
	for _, p := range m.params {
		arg := golang.Id(p.name)
		if p.typ.local() {
			arg = golang.Qual(engImp, p.typ.baseName()).Call(golang.Id(p.name))
		}
		args = append(args, arg)
	}
	call := golang.Id("w").Sel("q").Sel(m.name).Call(args...)
	file.Func().Recv("w", golang.Named(wrap)).Id(m.name).Params(paramFields(m.params)...).Results(paramFields(m.results)...).Block(wrapBody(m, call)...)
}

func wrapBody(m method, call *golang.Expr) []golang.Code {
	var locals []field
	for _, r := range m.results {
		if r.typ.local() {
			locals = append(locals, r)
		}
	}
	if len(locals) == 0 {
		return []golang.Code{golang.Return(call)}
	}
	lhs := make([]*golang.Expr, 0, len(m.results))
	for i, r := range m.results {
		name := r.name
		if name == "" {
			name = tmpName(i, r.typ)
		}
		lhs = append(lhs, golang.Id(name))
	}
	body := []golang.Code{golang.Define(lhs, call)}
	rets := make([]*golang.Expr, 0, len(m.results))
	for i, r := range m.results {
		id := r.name
		if id == "" {
			id = tmpName(i, r.typ)
		}
		if r.typ.kind == "slice" && r.typ.local() {
			item := r.typ.baseName()
			body = append(body,
				golang.Define([]*golang.Expr{golang.Id("out")}, golang.Id("make").Call(
					golang.Slice(golang.Named(item)).AsExpr(),
					golang.Id("len").Call(golang.Id(id)),
				)),
				golang.Range(golang.Id(id),
					golang.Assign(0, []*golang.Expr{golang.Id("out").Index(golang.Id("i"))}, golang.Named(item).AsExpr().Call(golang.Id(id).Index(golang.Id("i")))),
				).WithKey(true, golang.Id("i"), nil),
			)
			rets = append(rets, golang.Id("out"))
			continue
		}
		if r.typ.local() {
			rets = append(rets, golang.Named(r.typ.baseName()).AsExpr().Call(golang.Id(id)))
			continue
		}
		rets = append(rets, golang.Id(id))
	}
	body = append(body, golang.Return(rets...))
	return body
}

func tmpName(i int, t typeExpr) string {
	if t.kind == "ident" && t.name == "error" {
		return "err"
	}
	if t.local() {
		return "v"
	}
	return "r"
}

func paramFields(fs []field) []*golang.Field {
	out := make([]*golang.Field, 0, len(fs))
	for _, fl := range fs {
		typ := fl.typ
		variadic := typ.kind == "ellipsis"
		if variadic && typ.elem != nil {
			typ = *typ.elem
		}
		var field *golang.Field
		if fl.name == "" {
			field = golang.NewField(goType(typ))
		} else {
			field = golang.NewField(goType(typ), fl.name)
		}
		field.Variadic = variadic
		out = append(out, field)
	}
	return out
}

func goType(t typeExpr) *golang.Type {
	switch t.kind {
	case "ident":
		return golang.Named(t.name)
	case "qual":
		return golang.QualType(qualPath(t.pkg), t.name)
	case "pointer":
		return golang.Pointer(goType(*t.elem))
	case "slice":
		return golang.Slice(goType(*t.elem))
	case "ellipsis":
		return goType(*t.elem)
	case "any":
		return golang.Named("any")
	default:
		return golang.Named("any")
	}
}

func qualPath(pkg string) string {
	switch pkg {
	case "context":
		return "context"
	case "sql":
		return "database/sql"
	case "time":
		return "time"
	default:
		return pkg
	}
}

func joinEmbed(dirs []string) string {
	n := len("//go:embed")
	for _, d := range dirs {
		n += 1 + len(d)
	}
	b := make([]byte, 0, n)
	b = append(b, "//go:embed"...)
	for _, d := range dirs {
		b = append(b, ' ')
		b = append(b, d...)
	}
	return string(b)
}
