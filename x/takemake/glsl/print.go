package glsl

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	precComma = iota + 1
	precAssign
	precTernary
	precOr
	precXor
	precAnd
	precBitOr
	precBitXor
	precBitAnd
	precEq
	precRel
	precShift
	precAdd
	precMul
	precUnary
	precPost
)

// Make writes canonical GLSL.
// [Take] of that text returns the same file.
func (file *File) Make(w io.Writer) error {
	if file == nil || w == nil {
		return fmt.Errorf("%w: writer", ErrValue)
	}
	var printer printer
	printer.file(file)
	if printer.err != nil {
		return printer.err
	}
	if _, err := io.WriteString(w, printer.b.String()); err != nil {
		return fmt.Errorf("%w: write: %w", ErrValue, err)
	}
	return nil
}

type printer struct {
	b   strings.Builder
	err error
}

func (printer *printer) file(file *File) {
	header := false
	if file.Version != 0 {
		header = true
		printer.b.WriteString("#version ")
		printer.b.WriteString(strconv.Itoa(file.Version))
		if file.Profile.IsValid() {
			printer.b.WriteByte(' ')
			printer.b.WriteString(file.Profile.String())
		}
		printer.b.WriteByte('\n')
	}
	for _, ext := range file.Extensions {
		header = true
		printer.b.WriteString("#extension ")
		printer.b.WriteString(ext.Name)
		printer.b.WriteString(" : ")
		printer.b.WriteString(ext.Behavior.String())
		printer.b.WriteByte('\n')
	}
	body := len(file.Precisions) > 0 || len(file.Structs) > 0 || len(file.Globals) > 0 || len(file.Functions) > 0
	if header && body {
		printer.b.WriteByte('\n')
	}
	for _, item := range file.Precisions {
		printer.b.WriteString("precision ")
		printer.b.WriteString(item.Rank.String())
		printer.b.WriteByte(' ')
		printer.b.WriteString(item.Type)
		printer.b.WriteString(";\n")
	}
	for _, item := range file.Structs {
		printer.structOf(item)
	}
	for _, decl := range file.Globals {
		printer.decl(decl, 0, true)
	}
	for _, fn := range file.Functions {
		printer.function(fn)
	}
}

func (printer *printer) structOf(item *Struct) {
	if item == nil {
		printer.fail("nil struct")
		return
	}
	printer.b.WriteString("struct ")
	printer.b.WriteString(item.Name)
	printer.b.WriteString(" {\n")
	for _, field := range item.Fields {
		printer.decl(field, 1, true)
	}
	printer.b.WriteString("};\n")
}

func (printer *printer) function(fn *Func) {
	if fn == nil {
		printer.fail("nil function")
		return
	}
	if fn.Rank.IsValid() {
		printer.b.WriteString(fn.Rank.String())
		printer.b.WriteByte(' ')
	}
	printer.b.WriteString(fn.Return.Name)
	printer.b.WriteByte(' ')
	printer.b.WriteString(fn.Name)
	printer.b.WriteByte('(')
	for i, param := range fn.Params {
		if i > 0 {
			printer.b.WriteString(", ")
		}
		printer.param(param)
	}
	printer.b.WriteByte(')')
	printer.sizes(fn.Return.Sizes)
	if fn.Body == nil {
		printer.b.WriteString(";\n")
		return
	}
	printer.b.WriteString(" {\n")
	for _, stmt := range fn.Body {
		printer.stmt(stmt, 1)
	}
	printer.b.WriteString("}\n")
}

func (printer *printer) param(param *Param) {
	if param == nil {
		printer.fail("nil parameter")
		return
	}
	if param.Const {
		printer.b.WriteString("const ")
	}
	if param.Storage.IsValid() {
		printer.b.WriteString(param.Storage.String())
		printer.b.WriteByte(' ')
	}
	for _, mem := range param.Memory {
		printer.b.WriteString(mem.String())
		printer.b.WriteByte(' ')
	}
	if param.Rank.IsValid() {
		printer.b.WriteString(param.Rank.String())
		printer.b.WriteByte(' ')
	}
	printer.b.WriteString(param.Type.Name)
	if param.Name != "" {
		printer.b.WriteByte(' ')
		printer.b.WriteString(param.Name)
	}
	printer.sizes(param.Type.Sizes)
}

func (printer *printer) decl(decl *Decl, indent int, semi bool) {
	if decl == nil {
		printer.fail("nil decl")
		return
	}
	printer.indent(indent)
	printer.qualifiers(decl)
	if decl.Block != "" {
		printer.b.WriteString(decl.Block)
		printer.b.WriteString(" {\n")
		for _, field := range decl.Fields {
			printer.decl(field, indent+1, true)
		}
		printer.indent(indent)
		printer.b.WriteByte('}')
		if decl.Instance != "" {
			printer.b.WriteByte(' ')
			printer.b.WriteString(decl.Instance)
			printer.sizes(decl.Sizes)
		}
		if semi {
			printer.b.WriteString(";\n")
		}
		return
	}
	if decl.Type.Name == "" && decl.Name == "" {
		printer.trimSpace()
	}
	if decl.Type.Name != "" {
		printer.b.WriteString(decl.Type.Name)
		if decl.Name != "" {
			printer.b.WriteByte(' ')
		}
	}
	if decl.Name != "" {
		printer.b.WriteString(decl.Name)
	}
	if decl.Type.Name != "" || decl.Name != "" {
		printer.sizes(decl.Type.Sizes)
	}
	if decl.Init != nil {
		printer.b.WriteString(" = ")
		printer.expr(decl.Init, 0)
	}
	if semi {
		printer.b.WriteString(";\n")
	}
}

func (printer *printer) qualifiers(decl *Decl) {
	if len(decl.Layouts) > 0 {
		printer.b.WriteString("layout(")
		for i, item := range decl.Layouts {
			if i > 0 {
				printer.b.WriteString(", ")
			}
			printer.b.WriteString(item.Name)
			if item.Value != nil {
				printer.b.WriteString(" = ")
				printer.expr(item.Value, 0)
			}
		}
		printer.b.WriteString(") ")
	}
	if decl.Invariant {
		printer.b.WriteString("invariant ")
	}
	if decl.Precise {
		printer.b.WriteString("precise ")
	}
	if decl.Interp.IsValid() {
		printer.b.WriteString(decl.Interp.String())
		printer.b.WriteByte(' ')
	}
	if decl.Aux.IsValid() {
		printer.b.WriteString(decl.Aux.String())
		printer.b.WriteByte(' ')
	}
	for _, mem := range decl.Memory {
		printer.b.WriteString(mem.String())
		printer.b.WriteByte(' ')
	}
	if decl.Rank.IsValid() {
		printer.b.WriteString(decl.Rank.String())
		printer.b.WriteByte(' ')
	}
	if decl.Storage.IsValid() {
		printer.b.WriteString(decl.Storage.String())
		printer.b.WriteByte(' ')
	}
}

func (printer *printer) stmt(stmt Stmt, indent int) {
	if printer.err != nil {
		return
	}
	switch stmt := stmt.(type) {
	case nil:
		printer.fail("nil statement")
	case *Decl:
		printer.decl(stmt, indent, true)
	case *Empty:
		printer.indent(indent)
		printer.b.WriteString(";\n")
	case *ExprStmt:
		printer.indent(indent)
		printer.expr(stmt.Expr, 0)
		printer.b.WriteString(";\n")
	case *Return:
		printer.indent(indent)
		printer.b.WriteString("return")
		if stmt.Expr != nil {
			printer.b.WriteByte(' ')
			printer.expr(stmt.Expr, 0)
		}
		printer.b.WriteString(";\n")
	case *Break:
		printer.indent(indent)
		printer.b.WriteString("break;\n")
	case *Continue:
		printer.indent(indent)
		printer.b.WriteString("continue;\n")
	case *Discard:
		printer.indent(indent)
		printer.b.WriteString("discard;\n")
	case *Block:
		printer.indent(indent)
		printer.b.WriteString("{\n")
		for _, item := range stmt.Body {
			printer.stmt(item, indent+1)
		}
		printer.indent(indent)
		printer.b.WriteString("}\n")
	case *If:
		printer.ifStmt(stmt, indent, true)
	case *For:
		printer.indent(indent)
		printer.b.WriteString("for (")
		printer.forInit(stmt.Init)
		printer.b.WriteString("; ")
		if stmt.Cond != nil {
			printer.expr(stmt.Cond, 0)
		}
		printer.b.WriteString("; ")
		if stmt.Post != nil {
			printer.expr(stmt.Post, 0)
		}
		printer.b.WriteString(") {\n")
		for _, item := range stmt.Body {
			printer.stmt(item, indent+1)
		}
		printer.indent(indent)
		printer.b.WriteString("}\n")
	case *While:
		printer.indent(indent)
		printer.b.WriteString("while (")
		printer.expr(stmt.Cond, 0)
		printer.b.WriteString(") {\n")
		for _, item := range stmt.Body {
			printer.stmt(item, indent+1)
		}
		printer.indent(indent)
		printer.b.WriteString("}\n")
	case *Do:
		printer.indent(indent)
		printer.b.WriteString("do {\n")
		for _, item := range stmt.Body {
			printer.stmt(item, indent+1)
		}
		printer.indent(indent)
		printer.b.WriteString("} while (")
		printer.expr(stmt.Cond, 0)
		printer.b.WriteString(");\n")
	case *Switch:
		printer.indent(indent)
		printer.b.WriteString("switch (")
		printer.expr(stmt.Expr, 0)
		printer.b.WriteString(") {\n")
		for _, item := range stmt.Cases {
			printer.indent(indent + 1)
			if item == nil {
				printer.fail("nil case")
				return
			}
			if item.Default {
				printer.b.WriteString("default:\n")
			} else {
				printer.b.WriteString("case ")
				printer.expr(item.Expr, 0)
				printer.b.WriteString(":\n")
			}
			for _, body := range item.Body {
				printer.stmt(body, indent+2)
			}
		}
		printer.indent(indent)
		printer.b.WriteString("}\n")
	default:
		printer.fail("statement")
	}
}

func (printer *printer) ifStmt(stmt *If, indent int, lead bool) {
	if stmt == nil {
		printer.fail("nil if")
		return
	}
	if lead {
		printer.indent(indent)
	}
	printer.b.WriteString("if (")
	printer.expr(stmt.Cond, 0)
	printer.b.WriteString(") {\n")
	for _, item := range stmt.Body {
		printer.stmt(item, indent+1)
	}
	printer.indent(indent)
	printer.b.WriteByte('}')
	if stmt.Else == nil {
		printer.b.WriteByte('\n')
		return
	}
	if len(stmt.Else) == 1 {
		if next, ok := stmt.Else[0].(*If); ok {
			printer.b.WriteString(" else ")
			printer.ifStmt(next, indent, false)
			return
		}
	}
	printer.b.WriteString(" else {\n")
	for _, item := range stmt.Else {
		printer.stmt(item, indent+1)
	}
	printer.indent(indent)
	printer.b.WriteString("}\n")
}

func (printer *printer) forInit(stmt Stmt) {
	switch stmt := stmt.(type) {
	case nil:
	case *Decl:
		printer.decl(stmt, 0, false)
	case *ExprStmt:
		printer.expr(stmt.Expr, 0)
	case *Block:
		printer.forDecls(stmt.Body)
	default:
		printer.fail("for init")
	}
}

func (printer *printer) forDecls(body []Stmt) {
	if len(body) == 0 {
		return
	}
	first, ok := body[0].(*Decl)
	if !ok {
		printer.fail("for init")
		return
	}
	printer.qualifiers(first)
	printer.b.WriteString(first.Type.Name)
	for i, stmt := range body {
		decl, ok := stmt.(*Decl)
		if !ok {
			printer.fail("for init")
			return
		}
		if i > 0 {
			printer.b.WriteString(", ")
		} else {
			printer.b.WriteByte(' ')
		}
		printer.b.WriteString(decl.Name)
		printer.sizes(decl.Type.Sizes)
		if decl.Init != nil {
			printer.b.WriteString(" = ")
			printer.expr(decl.Init, 0)
		}
	}
}

func (printer *printer) expr(expr Expr, limit int) {
	if printer.err != nil {
		return
	}
	if expr == nil {
		printer.fail("nil expression")
		printer.b.WriteByte('0')
		return
	}
	switch expr := expr.(type) {
	case *Ident:
		printer.b.WriteString(expr.Name)
	case *Literal:
		printer.b.WriteString(expr.text())
	case *Binary:
		printer.binary(expr, limit)
	case *Unary:
		printer.unary(expr, limit)
	case *Ternary:
		printer.ternary(expr, limit)
	case *Call:
		printer.wrap(precPost, limit, func() {
			printer.expr(expr.Callee, precPost)
			printer.b.WriteByte('(')
			for i, arg := range expr.Args {
				if i > 0 {
					printer.b.WriteString(", ")
				}
				printer.expr(arg, 0)
			}
			printer.b.WriteByte(')')
		})
	case *Index:
		printer.wrap(precPost, limit, func() {
			printer.expr(expr.Base, precPost)
			printer.b.WriteByte('[')
			printer.expr(expr.Index, 0)
			printer.b.WriteByte(']')
		})
	case *Select:
		printer.wrap(precPost, limit, func() {
			printer.expr(expr.Expr, precPost)
			printer.b.WriteByte('.')
			printer.b.WriteString(expr.Name)
		})
	case *Postfix:
		printer.wrap(precPost, limit, func() {
			printer.expr(expr.Expr, precPost)
			printer.b.WriteString(expr.Op)
		})
	case *List:
		printer.b.WriteByte('{')
		for i, item := range expr.Exprs {
			if i > 0 {
				printer.b.WriteString(", ")
			}
			printer.expr(item, 0)
		}
		printer.b.WriteByte('}')
	default:
		printer.fail("expression")
	}
}

func (printer *printer) binary(expr *Binary, limit int) {
	prec, right := binPrec(expr.Op)
	if prec == 0 {
		printer.fail("operator %s", expr.Op)
		return
	}
	printer.wrap(prec, limit, func() {
		leftLimit := prec
		rightLimit := prec + 1
		if right {
			leftLimit = prec + 1
			rightLimit = prec
		}
		printer.expr(expr.Left, leftLimit)
		printer.b.WriteByte(' ')
		printer.b.WriteString(expr.Op)
		printer.b.WriteByte(' ')
		printer.expr(expr.Right, rightLimit)
	})
}

func (printer *printer) unary(expr *Unary, limit int) {
	printer.wrap(precUnary, limit, func() {
		printer.b.WriteString(expr.Op)
		switch expr.Expr.(type) {
		case *Unary, *Binary, *Ternary:
			printer.b.WriteByte('(')
			printer.expr(expr.Expr, 0)
			printer.b.WriteByte(')')
		default:
			printer.expr(expr.Expr, precUnary)
		}
	})
}

func (printer *printer) ternary(expr *Ternary, limit int) {
	printer.wrap(precTernary, limit, func() {
		printer.expr(expr.Cond, precTernary+1)
		printer.b.WriteString(" ? ")
		printer.expr(expr.Then, 0)
		printer.b.WriteString(" : ")
		printer.expr(expr.Else, precTernary)
	})
}

func (printer *printer) wrap(prec, limit int, write func()) {
	if prec < limit {
		printer.b.WriteByte('(')
		write()
		printer.b.WriteByte(')')
		return
	}
	write()
}

func (printer *printer) sizes(sizes []Size) {
	for _, size := range sizes {
		printer.b.WriteByte('[')
		if size.Expr != nil {
			printer.expr(size.Expr, 0)
		}
		printer.b.WriteByte(']')
	}
}

func (printer *printer) indent(level int) {
	for range level {
		printer.b.WriteString("    ")
	}
}

func (printer *printer) trimSpace() {
	text := printer.b.String()
	if strings.HasSuffix(text, " ") {
		printer.b.Reset()
		printer.b.WriteString(text[:len(text)-1])
	}
}

func (printer *printer) fail(format string, args ...any) {
	if printer.err != nil {
		return
	}
	printer.err = fmt.Errorf("%w: %s", ErrValue, fmt.Sprintf(format, args...))
}

func binPrec(op string) (int, bool) {
	switch op {
	case ",":
		return precComma, false
	case "=", "+=", "-=", "*=", "/=", "%=", "<<=", ">>=", "&=", "^=", "|=":
		return precAssign, true
	case "||":
		return precOr, false
	case "^^":
		return precXor, false
	case "&&":
		return precAnd, false
	case "|":
		return precBitOr, false
	case "^":
		return precBitXor, false
	case "&":
		return precBitAnd, false
	case "==", "!=":
		return precEq, false
	case "<", ">", "<=", ">=":
		return precRel, false
	case "<<", ">>":
		return precShift, false
	case "+", "-":
		return precAdd, false
	case "*", "/", "%":
		return precMul, false
	default:
		return 0, false
	}
}

func (lit *Literal) text() string {
	if lit == nil {
		return "0"
	}
	switch lit.Kind {
	case KindBool:
		if lit.Bool {
			return "true"
		}
		return "false"
	case KindInt:
		return strconv.FormatInt(lit.Int, 10)
	case KindUint:
		return strconv.FormatUint(lit.Uint, 10) + "u"
	case KindFloat:
		return formatFloat(lit.Float)
	case KindDouble:
		return formatFloat(lit.Float) + "lf"
	default:
		return "0"
	}
}

func formatFloat(value float64) string {
	text := strconv.FormatFloat(value, 'g', -1, 64)
	if !strings.ContainsAny(text, ".e") {
		text += ".0"
	}
	return text
}
