package c

import (
	"fmt"
	"strconv"
	"strings"
)

type printer struct {
	b      strings.Builder
	indent int
	open   map[*Type]bool
}

func (p *printer) file(f *File) {
	for i, decl := range f.Decls {
		if decl == nil {
			continue
		}
		if i > 0 {
			p.b.WriteByte('\n')
		}
		p.decl(decl)
	}
}

func (p *printer) decl(d *Decl) {
	switch d.Kind {
	case DeclInclude:
		p.include(d.Include)
	case DeclType:
		p.typeDecl(d.Type)
	case DeclConst:
		p.constDecl(d.Const)
	case DeclFunc:
		p.funcDecl(d.Func)
	case DeclObject:
		p.objectDecl(d.Object)
	}
}

func (p *printer) include(inc *Include) {
	if inc == nil {
		return
	}
	if inc.System {
		fmt.Fprintf(&p.b, "#include <%s>\n", inc.Path)
		return
	}
	fmt.Fprintf(&p.b, "#include %q\n", inc.Path)
}

func (p *printer) constDecl(c *Const) {
	if c == nil {
		return
	}
	fmt.Fprintf(&p.b, "#define %s %d\n", c.Name, c.Value)
}

func (p *printer) typeDecl(d *TypeDecl) {
	if d == nil {
		return
	}
	if d.Name != "" {
		p.b.WriteString("typedef ")
		p.declaratorDefine(d.Type, d.Name, d.Define)
		p.b.WriteString(";\n")
		return
	}
	p.typeSpec(d.Type, d.Define)
	p.b.WriteString(";\n")
}

func (p *printer) declaratorDefine(t *Type, name string, define bool) {
	if define && t != nil && t.Of != nil && isAggregate(t.Of) {
		savedConst, savedVolatile := t.Of.Const, t.Of.Volatile
		savedInline := t.Of.Inline
		t.Of.Const = t.Const
		t.Of.Volatile = t.Volatile
		t.Of.Inline = true
		p.declarator(t.Of, name)
		t.Of.Const = savedConst
		t.Of.Volatile = savedVolatile
		t.Of.Inline = savedInline
		return
	}
	if !define || t == nil || t.Of != nil {
		p.declarator(t, name)
		return
	}
	if t.Kind != KindStruct && t.Kind != KindUnion && t.Kind != KindEnum {
		p.declarator(t, name)
		return
	}
	saved := t.Inline
	t.Inline = true
	p.declarator(t, name)
	t.Inline = saved
}

func (p *printer) objectDecl(obj *Object) {
	if obj == nil {
		return
	}
	p.storage(obj.Storage)
	p.declarator(obj.Type, obj.Name)
	if obj.Init != nil {
		p.b.WriteString(" = ")
		p.expr(obj.Init)
	}
	p.b.WriteString(";\n")
}

func (p *printer) funcDecl(fn *Func) {
	if fn == nil {
		return
	}
	p.storage(fn.Storage)
	if fn.Inline {
		p.b.WriteString("inline ")
	}
	ft := &Type{
		Kind:     KindFunction,
		Elem:     fn.Result,
		Params:   fn.Params,
		Variadic: fn.Variadic,
	}
	p.declarator(ft, fn.Name)
	if fn.Body == nil {
		p.b.WriteString(";\n")
		return
	}
	p.b.WriteByte(' ')
	p.block(fn.Body, true)
}

func (p *printer) storage(s Storage) {
	switch s {
	case StorageStatic:
		p.b.WriteString("static ")
	case StorageExtern:
		p.b.WriteString("extern ")
	}
}

func (p *printer) declarator(t *Type, name string) {
	base, text := peel(t, name)
	p.typeSpec(base, false)
	if text != "" {
		p.b.WriteByte(' ')
		p.b.WriteString(text)
	}
}

func peel(t *Type, name string) (*Type, string) {
	for t != nil {
		if t.Use != "" {
			return t, name
		}
		switch t.Kind {
		case KindPointer:
			star := "*"
			if t.Const {
				star += "const "
			}
			if t.Volatile {
				star += "volatile "
			}
			inner := star + name
			if t.Elem != nil && (t.Elem.Kind == KindArray || t.Elem.Kind == KindFunction) {
				inner = "(" + strings.TrimRight(inner, " ") + ")"
			}
			name = inner
			t = t.Elem
		case KindArray:
			if t.Len < 0 {
				name += "[]"
			} else {
				name += fmt.Sprintf("[%d]", t.Len)
			}
			t = t.Elem
		case KindFunction:
			name += paramText(t.Params, t.Variadic)
			t = t.Elem
		default:
			return t, name
		}
	}
	return nil, name
}

func paramText(params []*Param, variadic bool) string {
	if len(params) == 0 && !variadic {
		return "(void)"
	}
	var b strings.Builder
	b.WriteByte('(')
	for i, param := range params {
		if i > 0 {
			b.WriteString(", ")
		}
		if param == nil {
			continue
		}
		var p printer
		p.declarator(param.Type, param.Name)
		b.WriteString(strings.TrimRight(p.b.String(), "\n"))
	}
	if variadic {
		if len(params) > 0 {
			b.WriteString(", ")
		}
		b.WriteString("...")
	}
	b.WriteByte(')')
	return b.String()
}

func (p *printer) typeSpec(t *Type, define bool) {
	if t == nil {
		p.b.WriteString("int")
		return
	}
	if t.Use != "" && !define {
		p.quals(t)
		p.b.WriteString(t.Use)
		return
	}
	switch t.Kind {
	case KindStruct, KindUnion, KindEnum:
		p.quals(t)
		p.b.WriteString(aggregateWord(t.Kind))
		if t.Tag != "" {
			p.b.WriteByte(' ')
			p.b.WriteString(t.Tag)
		}
		if (define || t.Inline) && !p.open[t] {
			if p.open == nil {
				p.open = map[*Type]bool{}
			}
			p.open[t] = true
			p.aggregateBody(t)
			delete(p.open, t)
		}
	default:
		p.quals(t)
		if t.Spell != "" {
			p.b.WriteString(t.Spell)
			return
		}
		p.b.WriteString("int")
	}
}

func aggregateWord(kind Kind) string {
	switch kind {
	case KindUnion:
		return "union"
	case KindEnum:
		return "enum"
	default:
		return "struct"
	}
}

func (p *printer) quals(t *Type) {
	if t.Const {
		p.b.WriteString("const ")
	}
	if t.Volatile {
		p.b.WriteString("volatile ")
	}
}

func (p *printer) aggregateBody(t *Type) {
	p.b.WriteString(" {\n")
	p.indent++
	if t.Kind == KindEnum {
		for _, item := range t.Enums {
			if item == nil {
				continue
			}
			p.writeIndent()
			fmt.Fprintf(&p.b, "%s = %d,\n", item.Name, item.Value)
		}
	} else {
		for _, field := range t.Fields {
			if field == nil {
				continue
			}
			p.writeIndent()
			p.declarator(field.Type, field.Name)
			p.b.WriteString(";\n")
		}
	}
	p.indent--
	p.writeIndent()
	p.b.WriteByte('}')
}

func (p *printer) block(s *Stmt, trailNL bool) {
	p.b.WriteString("{\n")
	p.indent++
	if s != nil {
		for _, child := range s.Stmts {
			p.stmt(child)
		}
	}
	p.indent--
	p.writeIndent()
	p.b.WriteByte('}')
	if trailNL {
		p.b.WriteByte('\n')
	}
}

func (p *printer) stmt(s *Stmt) {
	if s == nil {
		return
	}
	switch s.Kind {
	case StmtBlock:
		p.writeIndent()
		p.block(s, true)
	case StmtDecl:
		p.writeIndent()
		p.localDecl(s)
	case StmtExpr:
		p.writeIndent()
		p.expr(s.Expr)
		p.b.WriteString(";\n")
	case StmtReturn:
		p.writeIndent()
		p.b.WriteString("return")
		if s.Expr != nil {
			p.b.WriteByte(' ')
			p.expr(s.Expr)
		}
		p.b.WriteString(";\n")
	case StmtIf:
		p.writeIf(s)
	case StmtWhile:
		p.writeIndent()
		p.b.WriteString("while (")
		p.expr(s.Expr)
		p.b.WriteByte(')')
		p.controlled(s.Body, false)
	case StmtFor:
		p.writeFor(s)
	case StmtBreak:
		p.writeIndent()
		p.b.WriteString("break;\n")
	case StmtContinue:
		p.writeIndent()
		p.b.WriteString("continue;\n")
	}
}

func (p *printer) localDecl(s *Stmt) {
	p.writeObjects(s.Objects)
	p.b.WriteString(";\n")
}

func (p *printer) writeIf(s *Stmt) {
	p.writeIndent()
	p.writeIfTail(s)
}

func (p *printer) writeIfTail(s *Stmt) {
	p.b.WriteString("if (")
	p.expr(s.Expr)
	p.b.WriteByte(')')
	p.controlled(s.Body, s.Else != nil)
	if s.Else == nil {
		return
	}
	if s.Body != nil && s.Body.Kind == StmtBlock {
		p.b.WriteString(" else")
	} else {
		p.writeIndent()
		p.b.WriteString("else")
	}
	if s.Else.Kind == StmtIf {
		p.b.WriteByte(' ')
		p.writeIfTail(s.Else)
		return
	}
	p.controlled(s.Else, false)
}

func (p *printer) writeFor(s *Stmt) {
	p.writeIndent()
	p.b.WriteString("for (")
	p.forInit(s.Init)
	p.b.WriteString("; ")
	if s.Expr != nil {
		p.expr(s.Expr)
	}
	p.b.WriteString("; ")
	if s.Post != nil {
		p.expr(s.Post)
	}
	p.b.WriteByte(')')
	p.controlled(s.Body, false)
}

func (p *printer) forInit(s *Stmt) {
	if s == nil {
		return
	}
	switch s.Kind {
	case StmtDecl:
		p.localInline(s)
	case StmtExpr:
		p.expr(s.Expr)
	}
}

func (p *printer) controlled(s *Stmt, elseFollows bool) {
	if s != nil && s.Kind == StmtBlock {
		p.b.WriteByte(' ')
		p.block(s, !elseFollows)
		return
	}
	p.b.WriteByte('\n')
	p.indent++
	p.stmt(s)
	p.indent--
}

func (p *printer) localInline(s *Stmt) {
	if len(s.Objects) == 0 {
		return
	}
	p.writeObjects(s.Objects)
}

func (p *printer) writeObjects(objs []*Object) {
	if len(objs) == 0 || objs[0] == nil {
		return
	}
	p.storage(objs[0].Storage)
	base, _ := peel(objs[0].Type, "")
	p.typeSpec(base, false)
	for i, obj := range objs {
		if obj == nil {
			continue
		}
		if i > 0 {
			p.b.WriteString(", ")
		} else {
			p.b.WriteByte(' ')
		}
		_, text := peel(obj.Type, obj.Name)
		p.b.WriteString(text)
		if obj.Init != nil {
			p.b.WriteString(" = ")
			p.expr(obj.Init)
		}
	}
}

func (p *printer) writeIndent() {
	for range p.indent {
		p.b.WriteByte('\t')
	}
}

func (p *printer) expr(e *Expr) {
	if e == nil {
		return
	}
	switch e.Kind {
	case ExprIdent:
		p.b.WriteString(e.Text)
	case ExprInt:
		fmt.Fprintf(&p.b, "%d", e.Int)
	case ExprFloat:
		p.b.WriteString(formatFloat(e.Float))
	case ExprString:
		p.b.WriteString(quoteC(e.Text))
	case ExprChar:
		p.b.WriteString(quoteChar(rune(e.Int)))
	case ExprCall:
		p.expr(e.Left)
		p.b.WriteByte('(')
		for i, arg := range e.Args {
			if i > 0 {
				p.b.WriteString(", ")
			}
			p.expr(arg)
		}
		p.b.WriteByte(')')
	case ExprUnary:
		p.b.WriteByte('(')
		p.b.WriteString(opSpell(e.Op))
		// A cast starts with '(', and a prefix operator does not take a cast.
		// Parentheses keep -(T)x readable as -((T) x).
		if e.Left != nil && e.Left.Kind == ExprCast {
			p.b.WriteByte('(')
			p.expr(e.Left)
			p.b.WriteByte(')')
		} else {
			p.expr(e.Left)
		}
		p.b.WriteByte(')')
	case ExprBinary:
		p.b.WriteByte('(')
		p.expr(e.Left)
		p.b.WriteByte(' ')
		p.b.WriteString(opSpell(e.Op))
		p.b.WriteByte(' ')
		p.expr(e.Right)
		p.b.WriteByte(')')
	case ExprAssign:
		p.b.WriteByte('(')
		p.expr(e.Left)
		p.b.WriteString(" = ")
		p.expr(e.Right)
		p.b.WriteByte(')')
	case ExprField:
		p.primary(e.Left)
		if e.Arrow {
			p.b.WriteString("->")
		} else {
			p.b.WriteByte('.')
		}
		p.b.WriteString(e.Text)
	case ExprIndex:
		p.primary(e.Left)
		p.b.WriteByte('[')
		p.expr(e.Right)
		p.b.WriteByte(']')
	case ExprCast:
		p.b.WriteByte('(')
		p.declarator(e.Type, "")
		p.b.WriteString(") ")
		p.expr(e.Left)
	case ExprSizeof:
		p.b.WriteString("sizeof(")
		if e.Type != nil {
			p.declarator(e.Type, "")
		} else {
			p.expr(e.Left)
		}
		p.b.WriteByte(')')
	}
}

func (p *printer) primary(e *Expr) {
	if e == nil {
		return
	}
	switch e.Kind {
	case ExprIdent, ExprInt, ExprFloat, ExprString, ExprChar, ExprCall, ExprField, ExprIndex, ExprSizeof:
		p.expr(e)
	default:
		p.b.WriteByte('(')
		p.expr(e)
		p.b.WriteByte(')')
	}
}

func opSpell(op Op) string {
	switch op {
	case OpAdd:
		return "+"
	case OpSub:
		return "-"
	case OpMul:
		return "*"
	case OpDiv:
		return "/"
	case OpMod:
		return "%"
	case OpEq:
		return "=="
	case OpNe:
		return "!="
	case OpLt:
		return "<"
	case OpGt:
		return ">"
	case OpLe:
		return "<="
	case OpGe:
		return ">="
	case OpAnd:
		return "&&"
	case OpOr:
		return "||"
	case OpBitAnd:
		return "&"
	case OpBitOr:
		return "|"
	case OpBitXor:
		return "^"
	case OpShl:
		return "<<"
	case OpShr:
		return ">>"
	case OpNot:
		return "!"
	case OpBitNot:
		return "~"
	case OpNeg:
		return "-"
	case OpPos:
		return "+"
	case OpAddr:
		return "&"
	case OpDeref:
		return "*"
	default:
		return ""
	}
}

func formatFloat(v float64) string {
	s := strconv.FormatFloat(v, 'g', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}

func quoteC(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\', '"':
			b.WriteByte('\\')
			b.WriteByte(c)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\a':
			b.WriteString(`\a`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\v':
			b.WriteString(`\v`)
		default:
			if c < 0x20 || c >= 0x7f {
				fmt.Fprintf(&b, `\x%02x`, c)
				continue
			}
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func quoteChar(r rune) string {
	switch r {
	case '\\':
		return `'\\'`
	case '\'':
		return `'\''`
	case '\n':
		return `'\n'`
	case '\r':
		return `'\r'`
	case '\t':
		return `'\t'`
	case 0:
		return `'\0'`
	}
	if r < 0x20 || r >= 0x7f {
		return fmt.Sprintf(`'\x%02x'`, r)
	}
	return "'" + string(r) + "'"
}
