package golang

func (f *File) walkDecl(decl Decl) {
	switch d := decl.(type) {
	case *Func:
		f.walkField(d.Receiver)
		f.walkFields(d.TypeParams)
		f.walkFields(d.Parameters)
		f.walkFields(d.Returns)
		f.walkStmts(d.Body)
	case *Const:
		f.walkSpecs(d.Specs)
	case *Var:
		f.walkSpecs(d.Specs)
	case *TypeDecl:
		for i := range d.Specs {
			f.walkFields(d.Specs[i].Params)
			f.walkType(d.Specs[i].Type)
		}
	}
}

func (f *File) walkSpecs(specs []ValueSpec) {
	for i := range specs {
		f.walkType(specs[i].Type)
		f.walkExprs(specs[i].Values)
	}
}

func (f *File) walkFields(fields []*Field) {
	for _, field := range fields {
		f.walkField(field)
	}
}

func (f *File) walkField(field *Field) {
	if field == nil {
		return
	}
	f.walkType(field.Type)
}

func (f *File) walkStmts(stmts []*Stmt) {
	for _, stmt := range stmts {
		f.walkStmt(stmt)
	}
}

func (f *File) walkStmt(stmt *Stmt) {
	if stmt == nil {
		return
	}
	f.walkExpr(stmt.Expr)
	f.walkExprs(stmt.Exprs)
	f.walkExprs(stmt.LHS)
	f.walkExprs(stmt.RHS)
	f.walkStmt(stmt.Init)
	f.walkExpr(stmt.Cond)
	f.walkStmt(stmt.Post)
	f.walkStmts(stmt.Body)
	f.walkStmt(stmt.ElseBranch)
	f.walkExpr(stmt.Key)
	f.walkExpr(stmt.Value)
	f.walkExpr(stmt.X)
	f.walkExpr(stmt.Y)
	f.walkDecl(stmt.Decl)
	f.walkStmt(stmt.Stmt)
}

func (f *File) walkExprs(exprs []*Expr) {
	for _, expr := range exprs {
		f.walkExpr(expr)
	}
}

func (f *File) walkExpr(expr *Expr) {
	if expr == nil {
		return
	}
	if expr.Path != "" {
		f.use(expr.Path)
	}
	f.walkExpr(expr.X)
	f.walkExpr(expr.Y)
	f.walkExpr(expr.Fun)
	f.walkExprs(expr.Args)
	f.walkExpr(expr.At)
	f.walkExprs(expr.Indexes)
	f.walkExpr(expr.Low)
	f.walkExpr(expr.High)
	f.walkExpr(expr.Max)
	f.walkType(expr.Type)
	for _, elt := range expr.Elts {
		if elt == nil {
			continue
		}
		f.walkExpr(elt.Key)
		f.walkExpr(elt.Value)
	}
	if expr.Fn != nil {
		f.walkField(expr.Fn.Receiver)
		f.walkFields(expr.Fn.TypeParams)
		f.walkFields(expr.Fn.Parameters)
		f.walkFields(expr.Fn.Returns)
		f.walkStmts(expr.Fn.Body)
	}
}

func (f *File) walkType(typ *Type) {
	if typ == nil {
		return
	}
	if typ.Path != "" {
		f.use(typ.Path)
	}
	f.walkExpr(typ.X)
	f.walkType(typ.Elem)
	f.walkType(typ.Key)
	f.walkType(typ.Left)
	f.walkType(typ.Right)
	f.walkExpr(typ.Len)
	f.walkFields(typ.Fields)
	f.walkFields(typ.Results)
	f.walkFields(typ.Params)
	for _, arg := range typ.Args {
		f.walkType(arg)
	}
}
