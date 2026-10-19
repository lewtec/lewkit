package glsl

import (
	"fmt"
	"io"
	"strings"
)

// Take reads one GLSL shader.
// Stage is inferred from the text. Macros are expanded and not stored.
// A later [File.Make] prints the same file.
func Take(r io.Reader) (*File, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: reader", ErrValue)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: read: %w", ErrValue, err)
	}
	pre, err := prepare(string(data))
	if err != nil {
		return nil, err
	}
	file, err := parseSource(pre.source)
	if err != nil {
		return nil, err
	}
	file.Version = pre.version
	file.Profile = pre.profile
	file.Extensions = pre.extensions
	file.Stage = inferStage(file)
	return file, nil
}

func inferStage(file *File) Stage {
	if file == nil {
		return 0
	}
	seen := map[string]bool{}
	var discard bool
	var local bool
	var hasOut bool
	for _, decl := range file.Globals {
		if hasLocalSize(decl) {
			local = true
		}
		if decl != nil && decl.Storage == StorageOut && (decl.Name != "" || decl.Block != "") {
			hasOut = true
		}
		noteDecl(decl, seen, &discard)
	}
	for _, item := range file.Structs {
		if item == nil {
			continue
		}
		for _, field := range item.Fields {
			noteDecl(field, seen, &discard)
		}
	}
	for _, fn := range file.Functions {
		if fn == nil {
			continue
		}
		for _, stmt := range fn.Body {
			walkStmt(stmt, seen, &discard)
		}
	}
	if local {
		return Compute
	}
	if seen["gl_Position"] || seen["gl_VertexIndex"] || seen["gl_VertexID"] || seen["gl_InstanceIndex"] || seen["gl_InstanceID"] || seen["gl_PointSize"] {
		return Vertex
	}
	if discard || seen["gl_FragCoord"] || seen["gl_FrontFacing"] || seen["gl_FragDepth"] || seen["gl_PointCoord"] || seen["gl_SampleID"] {
		return Fragment
	}
	if hasOut {
		return Fragment
	}
	return 0
}

func hasLocalSize(decl *Decl) bool {
	if decl == nil {
		return false
	}
	for _, item := range decl.Layouts {
		if strings.HasPrefix(item.Name, "local_size") {
			return true
		}
	}
	for _, field := range decl.Fields {
		if hasLocalSize(field) {
			return true
		}
	}
	return false
}

func noteDecl(decl *Decl, seen map[string]bool, discard *bool) {
	if decl == nil {
		return
	}
	if decl.Name != "" {
		seen[decl.Name] = true
	}
	if decl.Instance != "" {
		seen[decl.Instance] = true
	}
	for _, item := range decl.Layouts {
		walkExpr(item.Value, seen, discard)
	}
	for _, size := range decl.Sizes {
		walkExpr(size.Expr, seen, discard)
	}
	for _, size := range decl.Type.Sizes {
		walkExpr(size.Expr, seen, discard)
	}
	walkExpr(decl.Init, seen, discard)
	for _, field := range decl.Fields {
		noteDecl(field, seen, discard)
	}
}

func walkStmt(stmt Stmt, seen map[string]bool, discard *bool) {
	if stmt == nil {
		return
	}
	switch stmt := stmt.(type) {
	case *Decl:
		noteDecl(stmt, seen, discard)
	case *Block:
		for _, item := range stmt.Body {
			walkStmt(item, seen, discard)
		}
	case *If:
		walkExpr(stmt.Cond, seen, discard)
		for _, item := range stmt.Body {
			walkStmt(item, seen, discard)
		}
		for _, item := range stmt.Else {
			walkStmt(item, seen, discard)
		}
	case *For:
		walkStmt(stmt.Init, seen, discard)
		walkExpr(stmt.Cond, seen, discard)
		walkExpr(stmt.Post, seen, discard)
		for _, item := range stmt.Body {
			walkStmt(item, seen, discard)
		}
	case *While:
		walkExpr(stmt.Cond, seen, discard)
		for _, item := range stmt.Body {
			walkStmt(item, seen, discard)
		}
	case *Do:
		for _, item := range stmt.Body {
			walkStmt(item, seen, discard)
		}
		walkExpr(stmt.Cond, seen, discard)
	case *Switch:
		walkExpr(stmt.Expr, seen, discard)
		for _, item := range stmt.Cases {
			if item == nil {
				continue
			}
			walkExpr(item.Expr, seen, discard)
			for _, body := range item.Body {
				walkStmt(body, seen, discard)
			}
		}
	case *Return:
		walkExpr(stmt.Expr, seen, discard)
	case *Discard:
		*discard = true
	case *ExprStmt:
		walkExpr(stmt.Expr, seen, discard)
	}
}

func walkExpr(expr Expr, seen map[string]bool, discard *bool) {
	if expr == nil {
		return
	}
	switch expr := expr.(type) {
	case *Ident:
		seen[expr.Name] = true
	case *Literal:
	case *Unary:
		walkExpr(expr.Expr, seen, discard)
	case *Binary:
		walkExpr(expr.Left, seen, discard)
		walkExpr(expr.Right, seen, discard)
	case *Ternary:
		walkExpr(expr.Cond, seen, discard)
		walkExpr(expr.Then, seen, discard)
		walkExpr(expr.Else, seen, discard)
	case *Call:
		walkExpr(expr.Callee, seen, discard)
		for _, arg := range expr.Args {
			walkExpr(arg, seen, discard)
		}
	case *Index:
		walkExpr(expr.Base, seen, discard)
		walkExpr(expr.Index, seen, discard)
	case *Select:
		walkExpr(expr.Expr, seen, discard)
		seen[expr.Name] = true
	case *Postfix:
		walkExpr(expr.Expr, seen, discard)
	case *List:
		for _, item := range expr.Exprs {
			walkExpr(item, seen, discard)
		}
	}
}
