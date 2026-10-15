package golang

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
)

func take(src []byte) (*File, error) {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, "source.go", src, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errParse, err)
	}
	return convertFile(fset, parsed)
}

func convertFile(fset *token.FileSet, parsed *ast.File) (*File, error) {
	if parsed.Name == nil {
		return nil, fmt.Errorf("%w: package name", errSyntax)
	}
	file := NewFile(parsed.Name.Name)
	sc := newScope(nil)
	var prev ast.Node
	groupN := 0
	for _, decl := range parsed.Decls {
		group, ok := decl.(*ast.GenDecl)
		if !ok || group.Tok != token.IMPORT {
			continue
		}
		for _, spec := range group.Specs {
			im, ok := spec.(*ast.ImportSpec)
			if !ok {
				return nil, fmt.Errorf("%w: import", errSyntax)
			}
			if prev != nil && fset.Position(im.Pos()).Line > fset.Position(prev.End()).Line+1 {
				groupN++
			}
			converted, err := convertImport(im)
			if err != nil {
				return nil, err
			}
			converted.Group = groupN
			file.Imports = append(file.Imports, converted)
			sc.defineImport(converted.Local(), converted.Path)
			prev = im
		}
	}
	file.Imports = normalizeImports(file.Imports)
	for _, decl := range parsed.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok == token.IMPORT {
				continue
			}
			converted, err := convertGen(d, sc)
			if err != nil {
				return nil, err
			}
			file.Decls = append(file.Decls, converted)
		case *ast.FuncDecl:
			fn, err := convertFunc(d, sc)
			if err != nil {
				return nil, err
			}
			file.Decls = append(file.Decls, fn)
			sc.defineValue(d.Name.Name)
		default:
			return nil, fmt.Errorf("%w: %T", errSyntax, decl)
		}
	}
	return file, nil
}

func convertImport(spec *ast.ImportSpec) (*Import, error) {
	if spec.Path == nil {
		return nil, fmt.Errorf("%w: import path", errSyntax)
	}
	path, err := strconv.Unquote(spec.Path.Value)
	if err != nil {
		return nil, fmt.Errorf("%w: import path", errSyntax)
	}
	im := &Import{Path: path}
	if spec.Name != nil {
		im.Name = spec.Name.Name
	}
	return im, nil
}

func convertGen(group *ast.GenDecl, sc *scope) (Decl, error) {
	switch group.Tok {
	case token.CONST:
		decl := &Const{}
		for _, spec := range group.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				return nil, fmt.Errorf("%w: const spec", errSyntax)
			}
			converted, err := convertValue(value, sc)
			if err != nil {
				return nil, err
			}
			decl.Specs = append(decl.Specs, converted)
			sc.defineValues(converted.Names)
		}
		return decl, nil
	case token.VAR:
		decl := &Var{}
		for _, spec := range group.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				return nil, fmt.Errorf("%w: var spec", errSyntax)
			}
			converted, err := convertValue(value, sc)
			if err != nil {
				return nil, err
			}
			decl.Specs = append(decl.Specs, converted)
			sc.defineValues(converted.Names)
		}
		return decl, nil
	case token.TYPE:
		decl := &TypeDecl{}
		for _, spec := range group.Specs {
			item, ok := spec.(*ast.TypeSpec)
			if !ok {
				return nil, fmt.Errorf("%w: type spec", errSyntax)
			}
			if item.TypeParams != nil && len(item.TypeParams.List) > 0 {
				return nil, fmt.Errorf("%w: type parameters", errSyntax)
			}
			typ, err := convertType(item.Type, sc)
			if err != nil {
				return nil, err
			}
			decl.Specs = append(decl.Specs, TypeSpec{
				Name:  item.Name.Name,
				Alias: item.Assign.IsValid(),
				Type:  typ,
			})
			sc.defineValue(item.Name.Name)
		}
		return decl, nil
	default:
		return nil, fmt.Errorf("%w: declaration %s", errSyntax, group.Tok)
	}
}

func convertValue(spec *ast.ValueSpec, sc *scope) (ValueSpec, error) {
	out := ValueSpec{}
	for _, name := range spec.Names {
		out.Names = append(out.Names, name.Name)
	}
	if spec.Type != nil {
		typ, err := convertType(spec.Type, sc)
		if err != nil {
			return ValueSpec{}, err
		}
		out.Type = typ
	}
	if spec.Values != nil {
		out.Values = make([]*Expr, 0, len(spec.Values))
		for _, value := range spec.Values {
			expr, err := convertExpr(value, sc)
			if err != nil {
				return ValueSpec{}, err
			}
			out.Values = append(out.Values, expr)
		}
	}
	return out, nil
}

func convertFunc(decl *ast.FuncDecl, sc *scope) (*Func, error) {
	if decl.Name == nil || decl.Type == nil {
		return nil, fmt.Errorf("%w: function", errSyntax)
	}
	if decl.Type.TypeParams != nil && len(decl.Type.TypeParams.List) > 0 {
		return nil, fmt.Errorf("%w: type parameters", errSyntax)
	}
	fn := &Func{Name: decl.Name.Name}
	body := sc.child()
	if decl.Recv != nil {
		if len(decl.Recv.List) != 1 {
			return nil, fmt.Errorf("%w: receiver", errSyntax)
		}
		field, err := convertField(decl.Recv.List[0], sc)
		if err != nil {
			return nil, err
		}
		fn.Receiver = field
		body.defineValues(field.Names)
	}
	params, err := convertFields(decl.Type.Params, sc)
	if err != nil {
		return nil, err
	}
	fn.Parameters = params
	for _, field := range params {
		body.defineValues(field.Names)
	}
	results, err := convertFields(decl.Type.Results, sc)
	if err != nil {
		return nil, err
	}
	fn.Returns = results
	for _, field := range results {
		body.defineValues(field.Names)
	}
	if decl.Body != nil {
		fn.HasBody = true
		fn.Body, err = convertStmtList(decl.Body.List, body)
		if err != nil {
			return nil, err
		}
	}
	return fn, nil
}

func convertFields(list *ast.FieldList, sc *scope) ([]*Field, error) {
	if list == nil {
		return []*Field{}, nil
	}
	out := make([]*Field, 0, len(list.List))
	for _, field := range list.List {
		converted, err := convertField(field, sc)
		if err != nil {
			return nil, err
		}
		out = append(out, converted)
	}
	return out, nil
}

func convertField(field *ast.Field, sc *scope) (*Field, error) {
	if field == nil {
		return nil, fmt.Errorf("%w: field", errSyntax)
	}
	out := &Field{Names: []string{}}
	for _, name := range field.Names {
		out.Names = append(out.Names, name.Name)
	}
	typ, variadic, err := convertFieldType(field.Type, sc)
	if err != nil {
		return nil, err
	}
	out.Type = typ
	out.Variadic = variadic
	if field.Tag != nil {
		tag, err := strconv.Unquote(field.Tag.Value)
		if err != nil {
			return nil, fmt.Errorf("%w: struct tag", errSyntax)
		}
		out.Tag = tag
	}
	return out, nil
}

func convertFieldType(expr ast.Expr, sc *scope) (*Type, bool, error) {
	if ellipsis, ok := expr.(*ast.Ellipsis); ok {
		typ, err := convertType(ellipsis.Elt, sc)
		if err != nil {
			return nil, false, err
		}
		return typ, true, nil
	}
	typ, err := convertType(expr, sc)
	if err != nil {
		return nil, false, err
	}
	return typ, false, nil
}

func convertType(expr ast.Expr, sc *scope) (*Type, error) {
	if expr == nil {
		return nil, fmt.Errorf("%w: nil type", errSyntax)
	}
	switch e := expr.(type) {
	case *ast.ParenExpr:
		return convertType(e.X, sc)
	case *ast.Ident:
		return Named(e.Name), nil
	case *ast.SelectorExpr:
		_, typ, err := convertSelector(e, sc, true)
		return typ, err
	case *ast.StarExpr:
		elem, err := convertType(e.X, sc)
		if err != nil {
			return nil, err
		}
		return Pointer(elem), nil
	case *ast.ArrayType:
		return convertArray(e, sc)
	case *ast.MapType:
		key, err := convertType(e.Key, sc)
		if err != nil {
			return nil, err
		}
		elem, err := convertType(e.Value, sc)
		if err != nil {
			return nil, err
		}
		return MapType(key, elem), nil
	case *ast.StructType:
		fields, err := convertFields(e.Fields, sc)
		if err != nil {
			return nil, err
		}
		return &Type{Kind: TypeStruct, Fields: fields}, nil
	case *ast.InterfaceType:
		fields, err := convertFields(e.Methods, sc)
		if err != nil {
			return nil, err
		}
		return &Type{Kind: TypeInterface, Fields: fields}, nil
	case *ast.FuncType:
		if e.TypeParams != nil && len(e.TypeParams.List) > 0 {
			return nil, fmt.Errorf("%w: type parameters", errSyntax)
		}
		params, err := convertFields(e.Params, sc)
		if err != nil {
			return nil, err
		}
		results, err := convertFields(e.Results, sc)
		if err != nil {
			return nil, err
		}
		return &Type{Kind: TypeFunc, Fields: params, Results: results}, nil
	case *ast.ChanType:
		elem, err := convertType(e.Value, sc)
		if err != nil {
			return nil, err
		}
		return Chan(chanFrom(e.Dir), elem), nil
	default:
		return nil, fmt.Errorf("%w: type %T", errSyntax, expr)
	}
}

func convertArray(expr *ast.ArrayType, sc *scope) (*Type, error) {
	elem, err := convertType(expr.Elt, sc)
	if err != nil {
		return nil, err
	}
	if expr.Len == nil {
		return Slice(elem), nil
	}
	if _, ok := expr.Len.(*ast.Ellipsis); ok {
		return &Type{Kind: TypeArray, Elem: elem, Ellipsis: true}, nil
	}
	length, err := convertExpr(expr.Len, sc)
	if err != nil {
		return nil, err
	}
	return Array(length, elem), nil
}

func convertStmtList(list []ast.Stmt, sc *scope) ([]*Stmt, error) {
	out := make([]*Stmt, 0, len(list))
	for _, stmt := range list {
		converted, err := convertStmt(stmt, sc)
		if err != nil {
			return nil, err
		}
		if converted != nil {
			out = append(out, converted)
		}
	}
	return out, nil
}

func convertStmt(stmt ast.Stmt, sc *scope) (*Stmt, error) {
	if stmt == nil {
		return nil, nil
	}
	switch s := stmt.(type) {
	case *ast.EmptyStmt:
		return nil, nil
	case *ast.ExprStmt:
		expr, err := convertExpr(s.X, sc)
		if err != nil {
			return nil, err
		}
		return &Stmt{Kind: StmtExpr, Expr: expr}, nil
	case *ast.ReturnStmt:
		vals, err := convertExprs(s.Results, sc)
		if err != nil {
			return nil, err
		}
		return &Stmt{Kind: StmtReturn, Exprs: vals}, nil
	case *ast.AssignStmt:
		return convertAssign(s, sc)
	case *ast.DeclStmt:
		group, ok := s.Decl.(*ast.GenDecl)
		if !ok {
			return nil, fmt.Errorf("%w: declaration statement", errSyntax)
		}
		decl, err := convertGen(group, sc)
		if err != nil {
			return nil, err
		}
		return &Stmt{Kind: StmtDecl, Decl: decl}, nil
	case *ast.IfStmt:
		return convertIf(s, sc)
	case *ast.ForStmt:
		return convertFor(s, sc)
	case *ast.RangeStmt:
		return convertRange(s, sc)
	case *ast.BlockStmt:
		body, err := convertStmtList(s.List, sc.child())
		if err != nil {
			return nil, err
		}
		return &Stmt{Kind: StmtBlock, Body: body}, nil
	case *ast.IncDecStmt:
		expr, err := convertExpr(s.X, sc)
		if err != nil {
			return nil, err
		}
		kind := StmtInc
		if s.Tok == token.DEC {
			kind = StmtDec
		} else if s.Tok != token.INC {
			return nil, fmt.Errorf("%w: incdec %s", errSyntax, s.Tok)
		}
		return &Stmt{Kind: kind, X: expr}, nil
	case *ast.DeferStmt:
		expr, err := convertExpr(s.Call, sc)
		if err != nil {
			return nil, err
		}
		return &Stmt{Kind: StmtDefer, Expr: expr}, nil
	case *ast.GoStmt:
		expr, err := convertExpr(s.Call, sc)
		if err != nil {
			return nil, err
		}
		return &Stmt{Kind: StmtGo, Expr: expr}, nil
	case *ast.BranchStmt:
		branch, err := branchFrom(s.Tok)
		if err != nil {
			return nil, err
		}
		out := &Stmt{Kind: StmtBranch, Branch: branch}
		if s.Label != nil {
			out.Label = s.Label.Name
		}
		return out, nil
	case *ast.SendStmt:
		ch, err := convertExpr(s.Chan, sc)
		if err != nil {
			return nil, err
		}
		value, err := convertExpr(s.Value, sc)
		if err != nil {
			return nil, err
		}
		return &Stmt{Kind: StmtSend, X: ch, Y: value}, nil
	case *ast.LabeledStmt:
		inner, err := convertStmt(s.Stmt, sc)
		if err != nil {
			return nil, err
		}
		return &Stmt{Kind: StmtLabel, Label: s.Label.Name, Stmt: inner}, nil
	default:
		return nil, fmt.Errorf("%w: statement %T", errSyntax, stmt)
	}
}

func convertAssign(stmt *ast.AssignStmt, sc *scope) (*Stmt, error) {
	lhs, err := convertExprs(stmt.Lhs, sc)
	if err != nil {
		return nil, err
	}
	rhs, err := convertExprs(stmt.Rhs, sc)
	if err != nil {
		return nil, err
	}
	op, define, err := opFromAssign(stmt.Tok)
	if err != nil {
		return nil, err
	}
	if define {
		for _, expr := range stmt.Lhs {
			if id, ok := expr.(*ast.Ident); ok {
				sc.defineValue(id.Name)
			}
		}
	}
	return &Stmt{Kind: StmtAssign, LHS: lhs, RHS: rhs, Op: op, Define: define}, nil
}

func convertIf(stmt *ast.IfStmt, sc *scope) (*Stmt, error) {
	block := sc.child()
	out := &Stmt{Kind: StmtIf}
	if stmt.Init != nil {
		init, err := convertStmt(stmt.Init, block)
		if err != nil {
			return nil, err
		}
		out.Init = init
	}
	cond, err := convertExpr(stmt.Cond, block)
	if err != nil {
		return nil, err
	}
	out.Cond = cond
	if stmt.Body == nil {
		return nil, fmt.Errorf("%w: if body", errSyntax)
	}
	out.Body, err = convertStmtList(stmt.Body.List, block.child())
	if err != nil {
		return nil, err
	}
	if stmt.Else != nil {
		out.ElseBranch, err = convertStmt(stmt.Else, block)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func convertFor(stmt *ast.ForStmt, sc *scope) (*Stmt, error) {
	block := sc.child()
	out := &Stmt{Kind: StmtFor}
	if stmt.Init != nil {
		init, err := convertStmt(stmt.Init, block)
		if err != nil {
			return nil, err
		}
		out.Init = init
	}
	if stmt.Cond != nil {
		cond, err := convertExpr(stmt.Cond, block)
		if err != nil {
			return nil, err
		}
		out.Cond = cond
	}
	if stmt.Post != nil {
		post, err := convertStmt(stmt.Post, block)
		if err != nil {
			return nil, err
		}
		out.Post = post
	}
	if stmt.Body == nil {
		return nil, fmt.Errorf("%w: for body", errSyntax)
	}
	body, err := convertStmtList(stmt.Body.List, block.child())
	if err != nil {
		return nil, err
	}
	out.Body = body
	return out, nil
}

func convertRange(stmt *ast.RangeStmt, sc *scope) (*Stmt, error) {
	x, err := convertExpr(stmt.X, sc)
	if err != nil {
		return nil, err
	}
	bodyScope := sc.child()
	out := &Stmt{Kind: StmtRange, X: x, Define: stmt.Tok == token.DEFINE}
	if stmt.Tok != token.DEFINE && stmt.Tok != token.ASSIGN && stmt.Tok != token.ILLEGAL {
		return nil, fmt.Errorf("%w: range %s", errSyntax, stmt.Tok)
	}
	if stmt.Key != nil {
		out.Key, err = convertExpr(stmt.Key, sc)
		if err != nil {
			return nil, err
		}
		if out.Define {
			if id, ok := stmt.Key.(*ast.Ident); ok {
				bodyScope.defineValue(id.Name)
			}
		}
	}
	if stmt.Value != nil {
		out.Value, err = convertExpr(stmt.Value, sc)
		if err != nil {
			return nil, err
		}
		if out.Define {
			if id, ok := stmt.Value.(*ast.Ident); ok {
				bodyScope.defineValue(id.Name)
			}
		}
	}
	if stmt.Body == nil {
		return nil, fmt.Errorf("%w: range body", errSyntax)
	}
	out.Body, err = convertStmtList(stmt.Body.List, bodyScope)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func convertExprs(list []ast.Expr, sc *scope) ([]*Expr, error) {
	out := make([]*Expr, 0, len(list))
	for _, expr := range list {
		converted, err := convertExpr(expr, sc)
		if err != nil {
			return nil, err
		}
		out = append(out, converted)
	}
	return out, nil
}

func convertExpr(expr ast.Expr, sc *scope) (*Expr, error) {
	if expr == nil {
		return nil, fmt.Errorf("%w: nil expression", errSyntax)
	}
	switch e := expr.(type) {
	case *ast.Ident:
		return convertIdent(e), nil
	case *ast.SelectorExpr:
		out, _, err := convertSelector(e, sc, false)
		return out, err
	case *ast.CallExpr:
		fun, err := convertExpr(e.Fun, sc)
		if err != nil {
			return nil, err
		}
		args, err := convertExprs(e.Args, sc)
		if err != nil {
			return nil, err
		}
		return &Expr{Kind: ExprCall, Fun: fun, Args: args, Ellipsis: e.Ellipsis.IsValid()}, nil
	case *ast.BasicLit:
		kind, err := litFrom(e.Kind)
		if err != nil {
			return nil, err
		}
		return &Expr{Kind: ExprLit, Lit: kind, Text: e.Value}, nil
	case *ast.CompositeLit:
		var typ *Type
		if e.Type != nil {
			typ, err := convertType(e.Type, sc)
			if err != nil {
				return nil, err
			}
			return composite(typ, e.Elts, sc)
		}
		return composite(typ, e.Elts, sc)
	case *ast.UnaryExpr:
		op, err := opFrom(e.Op)
		if err != nil {
			return nil, err
		}
		x, err := convertExpr(e.X, sc)
		if err != nil {
			return nil, err
		}
		return Unary(op, x), nil
	case *ast.StarExpr:
		x, err := convertExpr(e.X, sc)
		if err != nil {
			return nil, err
		}
		return Unary(OpMul, x), nil
	case *ast.BinaryExpr:
		op, err := opFrom(e.Op)
		if err != nil {
			return nil, err
		}
		x, err := convertExpr(e.X, sc)
		if err != nil {
			return nil, err
		}
		y, err := convertExpr(e.Y, sc)
		if err != nil {
			return nil, err
		}
		return Binary(op, x, y), nil
	case *ast.ParenExpr:
		x, err := convertExpr(e.X, sc)
		if err != nil {
			return nil, err
		}
		return Paren(x), nil
	case *ast.IndexExpr:
		x, err := convertExpr(e.X, sc)
		if err != nil {
			return nil, err
		}
		index, err := convertExpr(e.Index, sc)
		if err != nil {
			return nil, err
		}
		return &Expr{Kind: ExprIndex, X: x, At: index}, nil
	case *ast.IndexListExpr:
		x, err := convertExpr(e.X, sc)
		if err != nil {
			return nil, err
		}
		indexes, err := convertExprs(e.Indices, sc)
		if err != nil {
			return nil, err
		}
		return &Expr{Kind: ExprIndexList, X: x, Indexes: indexes}, nil
	case *ast.SliceExpr:
		x, err := convertExpr(e.X, sc)
		if err != nil {
			return nil, err
		}
		out := &Expr{Kind: ExprSlice, X: x}
		if e.Low != nil {
			out.Low, err = convertExpr(e.Low, sc)
			if err != nil {
				return nil, err
			}
		}
		if e.High != nil {
			out.High, err = convertExpr(e.High, sc)
			if err != nil {
				return nil, err
			}
		}
		if e.Max != nil {
			out.Max, err = convertExpr(e.Max, sc)
			if err != nil {
				return nil, err
			}
		}
		return out, nil
	case *ast.TypeAssertExpr:
		x, err := convertExpr(e.X, sc)
		if err != nil {
			return nil, err
		}
		out := &Expr{Kind: ExprAssert, X: x}
		if e.Type != nil {
			out.Type, err = convertType(e.Type, sc)
			if err != nil {
				return nil, err
			}
		}
		return out, nil
	case *ast.FuncLit:
		return convertFuncLit(e, sc)
	case *ast.ArrayType, *ast.StructType, *ast.FuncType, *ast.InterfaceType, *ast.MapType, *ast.ChanType:
		typ, err := convertType(e, sc)
		if err != nil {
			return nil, err
		}
		return typ.AsExpr(), nil
	default:
		return nil, fmt.Errorf("%w: expression %T", errSyntax, expr)
	}
}

func composite(typ *Type, elts []ast.Expr, sc *scope) (*Expr, error) {
	out := Composite(typ)
	out.Elts = make([]*Element, 0, len(elts))
	for _, elt := range elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			key, err := convertExpr(kv.Key, sc)
			if err != nil {
				return nil, err
			}
			value, err := convertExpr(kv.Value, sc)
			if err != nil {
				return nil, err
			}
			out.Elts = append(out.Elts, KeyValue(key, value))
			continue
		}
		value, err := convertExpr(elt, sc)
		if err != nil {
			return nil, err
		}
		out.Elts = append(out.Elts, Elem(value))
	}
	return out, nil
}

func convertIdent(id *ast.Ident) *Expr {
	switch id.Name {
	case "true":
		return Lit(true)
	case "false":
		return Lit(false)
	case "nil":
		return Lit(nil)
	default:
		return Id(id.Name)
	}
}

func convertSelector(expr *ast.SelectorExpr, sc *scope, asType bool) (*Expr, *Type, error) {
	if id, ok := expr.X.(*ast.Ident); ok {
		if path, ok := sc.lookupImport(id.Name); ok {
			if asType {
				return nil, QualType(path, expr.Sel.Name), nil
			}
			return Qual(path, expr.Sel.Name), nil, nil
		}
	}
	x, err := convertExpr(expr.X, sc)
	if err != nil {
		return nil, nil, err
	}
	if asType {
		return nil, &Type{Kind: TypeNamed, Name: expr.Sel.Name, X: x}, nil
	}
	return &Expr{Kind: ExprSelector, X: x, Name: expr.Sel.Name}, nil, nil
}

func convertFuncLit(expr *ast.FuncLit, sc *scope) (*Expr, error) {
	if expr.Type == nil {
		return nil, fmt.Errorf("%w: function literal", errSyntax)
	}
	if expr.Type.TypeParams != nil && len(expr.Type.TypeParams.List) > 0 {
		return nil, fmt.Errorf("%w: type parameters", errSyntax)
	}
	params, err := convertFields(expr.Type.Params, sc)
	if err != nil {
		return nil, err
	}
	results, err := convertFields(expr.Type.Results, sc)
	if err != nil {
		return nil, err
	}
	bodyScope := sc.child()
	for _, field := range params {
		bodyScope.defineValues(field.Names)
	}
	for _, field := range results {
		bodyScope.defineValues(field.Names)
	}
	fn := &Func{Parameters: params, Returns: results, HasBody: true}
	if expr.Body != nil {
		fn.Body, err = convertStmtList(expr.Body.List, bodyScope)
		if err != nil {
			return nil, err
		}
	} else {
		fn.Body = []*Stmt{}
	}
	return &Expr{Kind: ExprFunc, Fn: fn}, nil
}

type scope struct {
	parent  *scope
	imports map[string]string
	values  map[string]bool
}

func newScope(parent *scope) *scope {
	return &scope{parent: parent}
}

func (s *scope) child() *scope {
	return newScope(s)
}

func (s *scope) defineImport(name, path string) {
	if name == "" || name == "_" || name == "." || path == "" {
		return
	}
	if s.imports == nil {
		s.imports = map[string]string{}
	}
	s.imports[name] = path
}

func (s *scope) defineValue(name string) {
	if name == "" || name == "_" {
		return
	}
	if s.values == nil {
		s.values = map[string]bool{}
	}
	s.values[name] = true
}

func (s *scope) defineValues(names []string) {
	for _, name := range names {
		s.defineValue(name)
	}
}

func (s *scope) lookupImport(name string) (string, bool) {
	for cur := s; cur != nil; cur = cur.parent {
		if cur.values[name] {
			return "", false
		}
		if path, ok := cur.imports[name]; ok {
			return path, true
		}
	}
	return "", false
}
