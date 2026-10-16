package golang

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

type builder struct {
	file *File
}

func (b *builder) build(f *File) (*ast.File, error) {
	if f.Name == "" {
		return nil, fmt.Errorf("%w: package name", errSyntax)
	}
	out := &ast.File{Name: ast.NewIdent(f.Name)}
	if len(f.Imports) > 0 {
		specs := make([]ast.Spec, 0, len(f.Imports))
		for _, im := range f.Imports {
			if im == nil || im.Path == "" {
				return nil, fmt.Errorf("%w: import", errSyntax)
			}
			spec := &ast.ImportSpec{
				Path: &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(im.Path)},
			}
			if im.Name != "" {
				spec.Name = ast.NewIdent(im.Name)
			}
			specs = append(specs, spec)
		}
		out.Decls = append(out.Decls, &ast.GenDecl{Tok: token.IMPORT, Specs: specs})
	}
	for _, decl := range f.Decls {
		node, err := b.decl(decl)
		if err != nil {
			return nil, err
		}
		out.Decls = append(out.Decls, node)
	}
	return out, nil
}

func (b *builder) decl(decl Decl) (ast.Decl, error) {
	switch d := decl.(type) {
	case *Func:
		return b.funcDecl(d)
	case *Const:
		return b.values(token.CONST, d.Specs)
	case *Var:
		return b.values(token.VAR, d.Specs)
	case *TypeDecl:
		return b.typeDecl(d)
	default:
		return nil, fmt.Errorf("%w: declaration", errSyntax)
	}
}

func (b *builder) values(tok token.Token, specs []ValueSpec) (*ast.GenDecl, error) {
	if len(specs) == 0 {
		return nil, fmt.Errorf("%w: empty declaration", errSyntax)
	}
	group := &ast.GenDecl{Tok: tok}
	for _, spec := range specs {
		if len(spec.Names) == 0 {
			return nil, fmt.Errorf("%w: declaration names", errSyntax)
		}
		value := &ast.ValueSpec{}
		for _, name := range spec.Names {
			value.Names = append(value.Names, ast.NewIdent(name))
		}
		if spec.Type != nil {
			typ, err := b.typ(spec.Type)
			if err != nil {
				return nil, err
			}
			value.Type = typ
		}
		for _, expr := range spec.Values {
			node, err := b.expr(expr)
			if err != nil {
				return nil, err
			}
			value.Values = append(value.Values, node)
		}
		group.Specs = append(group.Specs, value)
	}
	return group, nil
}

func (b *builder) typeDecl(decl *TypeDecl) (*ast.GenDecl, error) {
	if decl == nil || len(decl.Specs) == 0 {
		return nil, fmt.Errorf("%w: type declaration", errSyntax)
	}
	group := &ast.GenDecl{Tok: token.TYPE}
	for _, spec := range decl.Specs {
		if spec.Name == "" || spec.Type == nil {
			return nil, fmt.Errorf("%w: type spec", errSyntax)
		}
		typ, err := b.typ(spec.Type)
		if err != nil {
			return nil, err
		}
		item := &ast.TypeSpec{Name: ast.NewIdent(spec.Name), Type: typ}
		if len(spec.Params) > 0 {
			params, err := b.fields(spec.Params)
			if err != nil {
				return nil, err
			}
			item.TypeParams = params
		}
		if spec.Alias {
			item.Assign = 1
		}
		group.Specs = append(group.Specs, item)
	}
	return group, nil
}

func (b *builder) funcDecl(fn *Func) (*ast.FuncDecl, error) {
	if fn == nil || fn.Name == "" {
		return nil, fmt.Errorf("%w: function name", errSyntax)
	}
	typ, err := b.funcType(fn.TypeParams, fn.Parameters, fn.Returns)
	if err != nil {
		return nil, err
	}
	decl := &ast.FuncDecl{Name: ast.NewIdent(fn.Name), Type: typ}
	if fn.Receiver != nil {
		recv, err := b.fields([]*Field{fn.Receiver})
		if err != nil {
			return nil, err
		}
		decl.Recv = recv
	}
	if fn.HasBody {
		body, err := b.stmts(fn.Body)
		if err != nil {
			return nil, err
		}
		decl.Body = &ast.BlockStmt{List: body}
	}
	return decl, nil
}

func (b *builder) funcType(typeParams, params, results []*Field) (*ast.FuncType, error) {
	list, err := b.fields(params)
	if err != nil {
		return nil, err
	}
	var tparams *ast.FieldList
	if len(typeParams) > 0 {
		tparams, err = b.fields(typeParams)
		if err != nil {
			return nil, err
		}
	}
	var out *ast.FieldList
	if len(results) > 0 {
		out, err = b.fields(results)
		if err != nil {
			return nil, err
		}
	}
	return &ast.FuncType{TypeParams: tparams, Params: list, Results: out}, nil
}

func (b *builder) fields(list []*Field) (*ast.FieldList, error) {
	fl := &ast.FieldList{}
	for _, field := range list {
		if field == nil || field.Type == nil {
			return nil, fmt.Errorf("%w: field type", errSyntax)
		}
		typ, err := b.typ(field.Type)
		if err != nil {
			return nil, err
		}
		if field.Variadic {
			typ = &ast.Ellipsis{Elt: typ}
		}
		node := &ast.Field{Type: typ}
		for _, name := range field.Names {
			node.Names = append(node.Names, ast.NewIdent(name))
		}
		if field.Tag != "" {
			node.Tag = &ast.BasicLit{Kind: token.STRING, Value: tagLiteral(field.Tag)}
		}
		fl.List = append(fl.List, node)
	}
	return fl, nil
}

func tagLiteral(tag string) string {
	if strings.Contains(tag, "`") {
		return strconv.Quote(tag)
	}
	return "`" + tag + "`"
}

func (b *builder) typ(t *Type) (ast.Expr, error) {
	if t == nil {
		return nil, fmt.Errorf("%w: nil type", errSyntax)
	}
	switch t.Kind {
	case TypeNamed:
		if t.Path != "" {
			return &ast.SelectorExpr{
				X:   ast.NewIdent(b.file.local(t.Path)),
				Sel: ast.NewIdent(t.Name),
			}, nil
		}
		if t.X != nil {
			x, err := b.expr(t.X)
			if err != nil {
				return nil, err
			}
			return &ast.SelectorExpr{X: x, Sel: ast.NewIdent(t.Name)}, nil
		}
		if t.Name == "" {
			return nil, fmt.Errorf("%w: type name", errSyntax)
		}
		return ast.NewIdent(t.Name), nil
	case TypePointer:
		elem, err := b.typ(t.Elem)
		if err != nil {
			return nil, err
		}
		return &ast.StarExpr{X: elem}, nil
	case TypeSlice:
		elem, err := b.typ(t.Elem)
		if err != nil {
			return nil, err
		}
		return &ast.ArrayType{Elt: elem}, nil
	case TypeArray:
		elem, err := b.typ(t.Elem)
		if err != nil {
			return nil, err
		}
		var length ast.Expr
		if t.Ellipsis {
			length = &ast.Ellipsis{}
		} else {
			length, err = b.expr(t.Len)
			if err != nil {
				return nil, err
			}
		}
		return &ast.ArrayType{Len: length, Elt: elem}, nil
	case TypeMap:
		key, err := b.typ(t.Key)
		if err != nil {
			return nil, err
		}
		elem, err := b.typ(t.Elem)
		if err != nil {
			return nil, err
		}
		return &ast.MapType{Key: key, Value: elem}, nil
	case TypeStruct:
		fields, err := b.fields(t.Fields)
		if err != nil {
			return nil, err
		}
		return &ast.StructType{Fields: fields}, nil
	case TypeInterface:
		fields, err := b.fields(t.Fields)
		if err != nil {
			return nil, err
		}
		return &ast.InterfaceType{Methods: fields}, nil
	case TypeFunc:
		return b.funcType(t.Params, t.Fields, t.Results)
	case TypeChan:
		elem, err := b.typ(t.Elem)
		if err != nil {
			return nil, err
		}
		return &ast.ChanType{Dir: t.Dir.ast(), Value: elem}, nil
	case TypeInst:
		return b.typeInst(t)
	case TypeBinary:
		return b.typeBinary(t)
	case TypeUnary:
		return b.typeUnary(t)
	default:
		return nil, fmt.Errorf("%w: type kind", errSyntax)
	}
}

func (b *builder) typeInst(t *Type) (ast.Expr, error) {
	base, err := b.expr(t.X)
	if err != nil {
		return nil, err
	}
	args := make([]ast.Expr, 0, len(t.Args))
	for _, arg := range t.Args {
		node, err := b.typ(arg)
		if err != nil {
			return nil, err
		}
		args = append(args, node)
	}
	// A one-argument list reparses as an index expression.
	if len(args) == 1 {
		return &ast.IndexExpr{X: base, Index: args[0]}, nil
	}
	return &ast.IndexListExpr{X: base, Indices: args}, nil
}

func (b *builder) typeBinary(t *Type) (ast.Expr, error) {
	op, err := t.Op.token()
	if err != nil {
		return nil, err
	}
	left, err := b.typ(t.Left)
	if err != nil {
		return nil, err
	}
	right, err := b.typ(t.Right)
	if err != nil {
		return nil, err
	}
	return &ast.BinaryExpr{X: left, Op: op, Y: right}, nil
}

func (b *builder) typeUnary(t *Type) (ast.Expr, error) {
	op, err := t.Op.token()
	if err != nil {
		return nil, err
	}
	elem, err := b.typ(t.Elem)
	if err != nil {
		return nil, err
	}
	return &ast.UnaryExpr{Op: op, X: elem}, nil
}

func (b *builder) stmts(list []*Stmt) ([]ast.Stmt, error) {
	out := make([]ast.Stmt, 0, len(list))
	for _, stmt := range list {
		node, err := b.stmt(stmt)
		if err != nil {
			return nil, err
		}
		if node != nil {
			out = append(out, node)
		}
	}
	return out, nil
}

func (b *builder) stmt(stmt *Stmt) (ast.Stmt, error) {
	if stmt == nil {
		return nil, fmt.Errorf("%w: nil statement", errSyntax)
	}
	switch stmt.Kind {
	case StmtExpr:
		expr, err := b.expr(stmt.Expr)
		if err != nil {
			return nil, err
		}
		return &ast.ExprStmt{X: expr}, nil
	case StmtReturn:
		vals, err := b.exprs(stmt.Exprs)
		if err != nil {
			return nil, err
		}
		return &ast.ReturnStmt{Results: vals}, nil
	case StmtAssign:
		return b.assign(stmt)
	case StmtIf:
		return b.ifStmt(stmt)
	case StmtFor:
		return b.forStmt(stmt)
	case StmtRange:
		return b.rangeStmt(stmt)
	case StmtBlock:
		body, err := b.stmts(stmt.Body)
		if err != nil {
			return nil, err
		}
		return &ast.BlockStmt{List: body}, nil
	case StmtDecl:
		decl, err := b.decl(stmt.Decl)
		if err != nil {
			return nil, err
		}
		return &ast.DeclStmt{Decl: decl}, nil
	case StmtInc, StmtDec:
		expr, err := b.expr(stmt.X)
		if err != nil {
			return nil, err
		}
		tok := token.INC
		if stmt.Kind == StmtDec {
			tok = token.DEC
		}
		return &ast.IncDecStmt{X: expr, Tok: tok}, nil
	case StmtDefer, StmtGo:
		call, err := b.call(stmt.Expr)
		if err != nil {
			return nil, err
		}
		if stmt.Kind == StmtDefer {
			return &ast.DeferStmt{Call: call}, nil
		}
		return &ast.GoStmt{Call: call}, nil
	case StmtBranch:
		tok, err := stmt.Branch.token()
		if err != nil {
			return nil, err
		}
		out := &ast.BranchStmt{Tok: tok}
		if stmt.Label != "" {
			out.Label = ast.NewIdent(stmt.Label)
		}
		return out, nil
	case StmtSend:
		ch, err := b.expr(stmt.X)
		if err != nil {
			return nil, err
		}
		value, err := b.expr(stmt.Y)
		if err != nil {
			return nil, err
		}
		return &ast.SendStmt{Chan: ch, Value: value}, nil
	case StmtLabel:
		if stmt.Label == "" {
			return nil, fmt.Errorf("%w: label", errSyntax)
		}
		var inner ast.Stmt = &ast.EmptyStmt{}
		if stmt.Stmt != nil {
			node, err := b.stmt(stmt.Stmt)
			if err != nil {
				return nil, err
			}
			inner = node
		}
		return &ast.LabeledStmt{Label: ast.NewIdent(stmt.Label), Stmt: inner}, nil
	case StmtSwitch:
		return b.switchStmt(stmt)
	case StmtTypeSwitch:
		return b.typeSwitchStmt(stmt)
	case StmtSelect:
		return b.selectStmt(stmt)
	case StmtCase:
		return b.caseClause(stmt)
	case StmtComm:
		return b.commClause(stmt)
	default:
		return nil, fmt.Errorf("%w: statement kind", errSyntax)
	}
}

func (b *builder) switchStmt(stmt *Stmt) (ast.Stmt, error) {
	body, err := b.stmts(stmt.Body)
	if err != nil {
		return nil, err
	}
	out := &ast.SwitchStmt{Body: &ast.BlockStmt{List: body}}
	if stmt.Init != nil {
		out.Init, err = b.stmt(stmt.Init)
		if err != nil {
			return nil, err
		}
	}
	if stmt.Cond != nil {
		out.Tag, err = b.expr(stmt.Cond)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (b *builder) typeSwitchStmt(stmt *Stmt) (ast.Stmt, error) {
	if stmt.Stmt == nil {
		return nil, fmt.Errorf("%w: type switch", errSyntax)
	}
	assign, err := b.stmt(stmt.Stmt)
	if err != nil {
		return nil, err
	}
	body, err := b.stmts(stmt.Body)
	if err != nil {
		return nil, err
	}
	out := &ast.TypeSwitchStmt{Assign: assign, Body: &ast.BlockStmt{List: body}}
	if stmt.Init != nil {
		out.Init, err = b.stmt(stmt.Init)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (b *builder) selectStmt(stmt *Stmt) (ast.Stmt, error) {
	body, err := b.stmts(stmt.Body)
	if err != nil {
		return nil, err
	}
	return &ast.SelectStmt{Body: &ast.BlockStmt{List: body}}, nil
}

func (b *builder) caseClause(stmt *Stmt) (ast.Stmt, error) {
	body, err := b.stmts(stmt.Body)
	if err != nil {
		return nil, err
	}
	out := &ast.CaseClause{Body: body}
	if stmt.Exprs != nil {
		out.List, err = b.exprs(stmt.Exprs)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (b *builder) commClause(stmt *Stmt) (ast.Stmt, error) {
	body, err := b.stmts(stmt.Body)
	if err != nil {
		return nil, err
	}
	out := &ast.CommClause{Body: body}
	if stmt.Stmt != nil {
		out.Comm, err = b.stmt(stmt.Stmt)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (b *builder) assign(stmt *Stmt) (ast.Stmt, error) {
	lhs, err := b.exprs(stmt.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := b.exprs(stmt.RHS)
	if err != nil {
		return nil, err
	}
	tok := token.DEFINE
	if !stmt.Define {
		tok, err = assignToken(stmt.Op)
		if err != nil {
			return nil, err
		}
	}
	return &ast.AssignStmt{Lhs: lhs, Tok: tok, Rhs: rhs}, nil
}

func (b *builder) ifStmt(stmt *Stmt) (ast.Stmt, error) {
	body, err := b.stmts(stmt.Body)
	if err != nil {
		return nil, err
	}
	out := &ast.IfStmt{Body: &ast.BlockStmt{List: body}}
	if stmt.Init != nil {
		out.Init, err = b.stmt(stmt.Init)
		if err != nil {
			return nil, err
		}
	}
	out.Cond, err = b.expr(stmt.Cond)
	if err != nil {
		return nil, err
	}
	if stmt.ElseBranch != nil {
		out.Else, err = b.stmt(stmt.ElseBranch)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (b *builder) forStmt(stmt *Stmt) (ast.Stmt, error) {
	body, err := b.stmts(stmt.Body)
	if err != nil {
		return nil, err
	}
	out := &ast.ForStmt{Body: &ast.BlockStmt{List: body}}
	if stmt.Init != nil {
		out.Init, err = b.stmt(stmt.Init)
		if err != nil {
			return nil, err
		}
	}
	if stmt.Cond != nil {
		out.Cond, err = b.expr(stmt.Cond)
		if err != nil {
			return nil, err
		}
	}
	if stmt.Post != nil {
		out.Post, err = b.stmt(stmt.Post)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (b *builder) rangeStmt(stmt *Stmt) (ast.Stmt, error) {
	x, err := b.expr(stmt.X)
	if err != nil {
		return nil, err
	}
	body, err := b.stmts(stmt.Body)
	if err != nil {
		return nil, err
	}
	out := &ast.RangeStmt{
		Tok:  token.ILLEGAL,
		X:    x,
		Body: &ast.BlockStmt{List: body},
	}
	if stmt.Key != nil || stmt.Value != nil {
		if stmt.Define {
			out.Tok = token.DEFINE
		} else {
			out.Tok = token.ASSIGN
		}
	}
	if stmt.Key != nil {
		out.Key, err = b.expr(stmt.Key)
		if err != nil {
			return nil, err
		}
	}
	if stmt.Value != nil {
		out.Value, err = b.expr(stmt.Value)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (b *builder) exprs(list []*Expr) ([]ast.Expr, error) {
	out := make([]ast.Expr, 0, len(list))
	for _, expr := range list {
		node, err := b.expr(expr)
		if err != nil {
			return nil, err
		}
		out = append(out, node)
	}
	return out, nil
}

func (b *builder) expr(expr *Expr) (ast.Expr, error) {
	if expr == nil {
		return nil, fmt.Errorf("%w: nil expression", errSyntax)
	}
	switch expr.Kind {
	case ExprIdent:
		if expr.Name == "" {
			return nil, fmt.Errorf("%w: identifier", errSyntax)
		}
		return ast.NewIdent(expr.Name), nil
	case ExprSelector:
		sel := ast.NewIdent(expr.Name)
		if expr.Path != "" {
			return &ast.SelectorExpr{X: ast.NewIdent(b.file.local(expr.Path)), Sel: sel}, nil
		}
		x, err := b.expr(expr.X)
		if err != nil {
			return nil, err
		}
		return &ast.SelectorExpr{X: x, Sel: sel}, nil
	case ExprCall:
		fun, err := b.expr(expr.Fun)
		if err != nil {
			return nil, err
		}
		args, err := b.exprs(expr.Args)
		if err != nil {
			return nil, err
		}
		call := &ast.CallExpr{Fun: fun, Args: args}
		if expr.Ellipsis {
			call.Ellipsis = 1
		}
		return call, nil
	case ExprLit:
		return b.lit(expr)
	case ExprComposite:
		var typ ast.Expr
		if expr.Type != nil {
			node, err := b.typ(expr.Type)
			if err != nil {
				return nil, err
			}
			typ = node
		}
		elts := make([]ast.Expr, 0, len(expr.Elts))
		for _, elt := range expr.Elts {
			if elt == nil {
				return nil, fmt.Errorf("%w: composite element", errSyntax)
			}
			value, err := b.expr(elt.Value)
			if err != nil {
				return nil, err
			}
			if elt.Key == nil {
				elts = append(elts, value)
				continue
			}
			key, err := b.expr(elt.Key)
			if err != nil {
				return nil, err
			}
			elts = append(elts, &ast.KeyValueExpr{Key: key, Value: value})
		}
		return &ast.CompositeLit{Type: typ, Elts: elts}, nil
	case ExprUnary:
		op, err := expr.Op.token()
		if err != nil {
			return nil, err
		}
		x, err := b.expr(expr.X)
		if err != nil {
			return nil, err
		}
		if op == token.MUL {
			return &ast.StarExpr{X: x}, nil
		}
		return &ast.UnaryExpr{Op: op, X: x}, nil
	case ExprBinary:
		op, err := expr.Op.token()
		if err != nil {
			return nil, err
		}
		x, err := b.expr(expr.X)
		if err != nil {
			return nil, err
		}
		y, err := b.expr(expr.Y)
		if err != nil {
			return nil, err
		}
		return &ast.BinaryExpr{X: x, Op: op, Y: y}, nil
	case ExprIndex:
		x, err := b.expr(expr.X)
		if err != nil {
			return nil, err
		}
		index, err := b.expr(expr.At)
		if err != nil {
			return nil, err
		}
		return &ast.IndexExpr{X: x, Index: index}, nil
	case ExprIndexList:
		x, err := b.expr(expr.X)
		if err != nil {
			return nil, err
		}
		indexes, err := b.exprs(expr.Indexes)
		if err != nil {
			return nil, err
		}
		return &ast.IndexListExpr{X: x, Indices: indexes}, nil
	case ExprSlice:
		x, err := b.expr(expr.X)
		if err != nil {
			return nil, err
		}
		out := &ast.SliceExpr{X: x, Slice3: expr.Max != nil}
		if expr.Low != nil {
			out.Low, err = b.expr(expr.Low)
			if err != nil {
				return nil, err
			}
		}
		if expr.High != nil {
			out.High, err = b.expr(expr.High)
			if err != nil {
				return nil, err
			}
		}
		if expr.Max != nil {
			out.Max, err = b.expr(expr.Max)
			if err != nil {
				return nil, err
			}
		}
		return out, nil
	case ExprParen:
		x, err := b.expr(expr.X)
		if err != nil {
			return nil, err
		}
		return &ast.ParenExpr{X: x}, nil
	case ExprType:
		return b.typ(expr.Type)
	case ExprFunc:
		if expr.Fn == nil {
			return nil, fmt.Errorf("%w: function literal", errSyntax)
		}
		typ, err := b.funcType(expr.Fn.TypeParams, expr.Fn.Parameters, expr.Fn.Returns)
		if err != nil {
			return nil, err
		}
		body, err := b.stmts(expr.Fn.Body)
		if err != nil {
			return nil, err
		}
		return &ast.FuncLit{Type: typ, Body: &ast.BlockStmt{List: body}}, nil
	case ExprAssert:
		x, err := b.expr(expr.X)
		if err != nil {
			return nil, err
		}
		out := &ast.TypeAssertExpr{X: x}
		if expr.Type != nil {
			out.Type, err = b.typ(expr.Type)
			if err != nil {
				return nil, err
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("%w: expression kind", errSyntax)
	}
}

func (b *builder) lit(expr *Expr) (ast.Expr, error) {
	switch expr.Lit {
	case LitBool, LitNil:
		if expr.Text == "" {
			return nil, fmt.Errorf("%w: literal", errSyntax)
		}
		return ast.NewIdent(expr.Text), nil
	case LitInt, LitFloat, LitImag, LitRune, LitString:
		if expr.Text == "" {
			return nil, fmt.Errorf("%w: literal", errSyntax)
		}
		return &ast.BasicLit{Kind: expr.Lit.token(), Value: expr.Text}, nil
	default:
		return nil, fmt.Errorf("%w: literal kind", errSyntax)
	}
}

func (b *builder) call(expr *Expr) (*ast.CallExpr, error) {
	node, err := b.expr(expr)
	if err != nil {
		return nil, err
	}
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return nil, fmt.Errorf("%w: call", errSyntax)
	}
	return call, nil
}
