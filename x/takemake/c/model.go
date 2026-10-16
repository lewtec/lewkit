package c

import (
	"fmt"
	"io"
)

// File is one C translation unit. Includes, consts, types, functions,
// and objects share [File.Decls] so source order survives a round trip.
type File struct {
	Decls []*Decl
}

// Decl is one top-level item. The field that matches Kind is set.
type Decl struct {
	Kind    DeclKind
	Include *Include
	Type    *TypeDecl
	Const   *Const
	Func    *Func
	Object  *Object
}

// Include is a #include line. System selects angle brackets.
type Include struct {
	Path   string
	System bool
}

// TypeDecl is a typedef or a struct, union, or enum declaration.
// Define prints the body on this declaration.
type TypeDecl struct {
	Name   string
	Type   *Type
	Define bool
}

// Const is an integer #define.
type Const struct {
	Name  string
	Value int64
}

// Func is one function. A nil Body is a declaration.
type Func struct {
	Name     string
	Result   *Type
	Params   []*Param
	Variadic bool
	Storage  Storage
	Inline   bool
	Body     *Stmt
}

// Object is one variable.
type Object struct {
	Name    string
	Type    *Type
	Storage Storage
	Init    *Expr
}

// Param is one function parameter.
type Param struct {
	Name string
	Type *Type
}

// Type is one C type. Size and Align are the LP64 System V layout.
// Field offsets live on [Field.Offset]. Use is a typedef name to print.
// Of is the type Use or a qualifier refers to. Inline prints a body at
// this occurrence.
type Type struct {
	Kind     Kind
	Size     int
	Align    int
	Signed   bool
	Const    bool
	Volatile bool
	Spell    string
	Use      string
	Tag      string
	Of       *Type
	Elem     *Type
	Len      int
	Fields   []*Field
	Params   []*Param
	Variadic bool
	Enums    []*Enumerator
	Defined  bool
	Inline   bool

	home   *TypeDecl
	laying bool
}

// Field is one struct or union member.
type Field struct {
	Name   string
	Type   *Type
	Offset int
}

// Enumerator is one enum constant.
type Enumerator struct {
	Name  string
	Value int64
}

// Stmt is one statement.
type Stmt struct {
	Kind    StmtKind
	Expr    *Expr
	Body    *Stmt
	Else    *Stmt
	Init    *Stmt
	Post    *Expr
	Stmts   []*Stmt
	Objects []*Object
}

// Expr is one expression.
type Expr struct {
	Kind  ExprKind
	Op    Op
	Text  string
	Int   int64
	Float float64
	Type  *Type
	Left  *Expr
	Right *Expr
	Args  []*Expr
	Arrow bool
}

// New returns an empty translation unit.
func New() *File {
	return &File{}
}

// Take parses C from r into a [File].
func Take(r io.Reader) (*File, error) {
	if r == nil {
		return nil, errRead
	}
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errRead, err)
	}
	file, err := parseSource(buf)
	if err != nil {
		return nil, err
	}
	file.layout()
	return file, nil
}

// Make prints the translation unit as C.
func (f *File) Make(w io.Writer) error {
	if f == nil {
		return errFile
	}
	if w == nil {
		return errWrite
	}
	f.layout()
	var p printer
	p.file(f)
	if _, err := io.WriteString(w, p.b.String()); err != nil {
		return fmt.Errorf("%w: %w", errWrite, err)
	}
	return nil
}

// IncludeSystem stores #include <path> and returns the file.
func (f *File) IncludeSystem(path string) *File {
	return f.include(path, true)
}

// IncludeQuoted stores #include "path" and returns the file.
func (f *File) IncludeQuoted(path string) *File {
	return f.include(path, false)
}

func (f *File) include(path string, system bool) *File {
	if f == nil {
		return nil
	}
	f.Decls = append(f.Decls, &Decl{
		Kind:    DeclInclude,
		Include: &Include{Path: path, System: system},
	})
	return f
}

// Const stores #define name value and returns the file.
func (f *File) Const(name string, value int64) *File {
	if f == nil {
		return nil
	}
	f.Decls = append(f.Decls, &Decl{
		Kind:  DeclConst,
		Const: &Const{Name: name, Value: value},
	})
	return f
}

// Struct stores a struct definition and returns it.
func (f *File) Struct(tag string) *Type {
	return f.aggregate(KindStruct, tag)
}

// Union stores a union definition and returns it.
func (f *File) Union(tag string) *Type {
	return f.aggregate(KindUnion, tag)
}

// Enum stores an enum definition and returns it.
func (f *File) Enum(tag string) *Type {
	t := f.aggregate(KindEnum, tag)
	t.Size = intSize
	t.Align = intSize
	return t
}

func (f *File) aggregate(kind Kind, tag string) *Type {
	t := &Type{Kind: kind, Tag: tag, Defined: true}
	decl := &TypeDecl{Type: t, Define: true}
	t.home = decl
	if f != nil {
		f.Decls = append(f.Decls, &Decl{Kind: DeclType, Type: decl})
	}
	layout(t)
	return t
}

// Typedef stores a typedef and returns a type that prints as name.
func (f *File) Typedef(name string, typ *Type) *Type {
	if typ != nil && typ.home != nil && typ.home.Name == "" && typ.Defined {
		return typ.Alias(name)
	}
	decl := &TypeDecl{Name: name, Type: typ}
	if f != nil {
		f.Decls = append(f.Decls, &Decl{Kind: DeclType, Type: decl})
	}
	return useType(name, typ)
}

// Alias names this struct, union, or enum as a typedef and returns the use.
func (t *Type) Alias(name string) *Type {
	if t != nil && t.home != nil {
		t.home.Name = name
	}
	return useType(name, t)
}

// Field stores a member and returns the aggregate.
func (t *Type) Field(name string, typ *Type) *Type {
	if t == nil {
		return nil
	}
	t.Fields = append(t.Fields, &Field{Name: name, Type: typ})
	t.Defined = true
	layout(t)
	return t
}

// Enumerator stores an enumerator and returns the enum.
func (t *Type) Enumerator(name string, value int64) *Type {
	if t == nil {
		return nil
	}
	t.Enums = append(t.Enums, &Enumerator{Name: name, Value: value})
	t.Defined = true
	layout(t)
	return t
}

// Object stores a variable and returns it.
func (f *File) Object(name string, typ *Type) *Object {
	obj := &Object{Name: name, Type: typ}
	if f != nil {
		f.Decls = append(f.Decls, &Decl{Kind: DeclObject, Object: obj})
	}
	return obj
}

// WithInit stores an initializer and returns the object.
func (o *Object) WithInit(e *Expr) *Object {
	if o == nil {
		return nil
	}
	o.Init = e
	return o
}

// Static stores static and returns the object.
func (o *Object) Static() *Object {
	if o == nil {
		return nil
	}
	o.Storage = StorageStatic
	return o
}

// Extern stores extern and returns the object.
func (o *Object) Extern() *Object {
	if o == nil {
		return nil
	}
	o.Storage = StorageExtern
	return o
}

// Func stores a function and returns it.
func (f *File) Func(name string) *Func {
	fn := &Func{Name: name, Result: Void()}
	if f != nil {
		f.Decls = append(f.Decls, &Decl{Kind: DeclFunc, Func: fn})
	}
	return fn
}

// Returns stores the result type and returns the function.
func (fn *Func) Returns(t *Type) *Func {
	if fn == nil {
		return nil
	}
	fn.Result = t
	return fn
}

// Param stores a parameter and returns the function.
func (fn *Func) Param(name string, t *Type) *Func {
	if fn == nil {
		return nil
	}
	fn.Params = append(fn.Params, &Param{Name: name, Type: t})
	return fn
}

// WithVariadic marks the parameter list and returns the function.
func (fn *Func) WithVariadic() *Func {
	if fn == nil {
		return nil
	}
	fn.Variadic = true
	return fn
}

// Static stores static and returns the function.
func (fn *Func) Static() *Func {
	if fn == nil {
		return nil
	}
	fn.Storage = StorageStatic
	return fn
}

// Extern stores extern and returns the function.
func (fn *Func) Extern() *Func {
	if fn == nil {
		return nil
	}
	fn.Storage = StorageExtern
	return fn
}

// WithBody stores the function body and returns the function.
func (fn *Func) WithBody(stmts ...*Stmt) *Func {
	if fn == nil {
		return nil
	}
	fn.Body = Block(stmts...)
	return fn
}

// Constant returns a const-qualified type. The receiver stays unqualified.
func (t *Type) Constant() *Type {
	if t == nil {
		return nil
	}
	if t.Const {
		return t
	}
	n := *t
	n.Const = true
	n.Of = t
	n.Fields = nil
	n.Enums = nil
	n.Defined = false
	n.Inline = false
	n.home = nil
	n.laying = false
	layout(&n)
	return &n
}

func useType(name string, of *Type) *Type {
	if of == nil {
		return &Type{Use: name}
	}
	t := &Type{
		Kind:   of.Kind,
		Signed: of.Signed,
		Spell:  of.Spell,
		Use:    name,
		Tag:    of.Tag,
		Of:     of,
	}
	layout(t)
	return t
}

// Void returns void.
func Void() *Type { return scalar(KindVoid, "void", 0, 1, false) }

// Bool returns bool.
func Bool() *Type { return scalar(KindBool, "bool", 1, 1, false) }

// Char returns char.
func Char() *Type { return integer("char", 1, true) }

// SignedChar returns signed char.
func SignedChar() *Type { return integer("signed char", 1, true) }

// UnsignedChar returns unsigned char.
func UnsignedChar() *Type { return integer("unsigned char", 1, false) }

// Short returns short.
func Short() *Type { return integer("short", 2, true) }

// UnsignedShort returns unsigned short.
func UnsignedShort() *Type { return integer("unsigned short", 2, false) }

// Int returns int.
func Int() *Type { return integer("int", intSize, true) }

// UnsignedInt returns unsigned int.
func UnsignedInt() *Type { return integer("unsigned int", intSize, false) }

// Long returns long.
func Long() *Type { return integer("long", longSize, true) }

// UnsignedLong returns unsigned long.
func UnsignedLong() *Type { return integer("unsigned long", longSize, false) }

// LongLong returns long long.
func LongLong() *Type { return integer("long long", 8, true) }

// UnsignedLongLong returns unsigned long long.
func UnsignedLongLong() *Type { return integer("unsigned long long", 8, false) }

// Float returns float.
func Float() *Type { return scalar(KindFloat, "float", 4, 4, false) }

// Double returns double.
func Double() *Type { return scalar(KindFloat, "double", 8, 8, false) }

// LongDouble returns long double.
func LongDouble() *Type {
	return scalar(KindFloat, "long double", longDoubleSize, longDoubleSize, false)
}

func integer(spell string, size int, signed bool) *Type {
	return scalar(KindInteger, spell, size, size, signed)
}

func scalar(kind Kind, spell string, size, align int, signed bool) *Type {
	if align < 1 {
		align = 1
	}
	return &Type{Kind: kind, Spell: spell, Size: size, Align: align, Signed: signed}
}

// Pointer returns a pointer to elem.
func Pointer(elem *Type) *Type {
	t := &Type{Kind: KindPointer, Elem: elem, Size: ptrSize, Align: ptrSize}
	layout(t)
	return t
}

// Array returns an array of n elements. A negative n is an incomplete array.
func Array(elem *Type, n int) *Type {
	t := &Type{Kind: KindArray, Elem: elem, Len: n}
	layout(t)
	return t
}

// FuncType returns a function type.
func FuncType(result *Type, params ...*Param) *Type {
	t := &Type{Kind: KindFunction, Elem: result, Params: params, Size: 0, Align: 1}
	return t
}

// ParamOf returns a parameter.
func ParamOf(name string, typ *Type) *Param {
	return &Param{Name: name, Type: typ}
}

// Ident returns an identifier.
func Ident(name string) *Expr {
	return &Expr{Kind: ExprIdent, Text: name}
}

// IntLit returns an integer literal.
func IntLit(v int64) *Expr {
	return &Expr{Kind: ExprInt, Int: v}
}

// FloatLit returns a floating literal.
func FloatLit(v float64) *Expr {
	return &Expr{Kind: ExprFloat, Float: v}
}

// StringLit returns a string literal.
func StringLit(v string) *Expr {
	return &Expr{Kind: ExprString, Text: v}
}

// CharLit returns a character literal.
func CharLit(v rune) *Expr {
	return &Expr{Kind: ExprChar, Int: int64(v)}
}

// Call returns a call.
func Call(fn *Expr, args ...*Expr) *Expr {
	return &Expr{Kind: ExprCall, Left: fn, Args: args}
}

// Unary returns a unary expression.
func Unary(op Op, x *Expr) *Expr {
	return &Expr{Kind: ExprUnary, Op: op, Left: x}
}

// Binary returns a binary expression.
func Binary(op Op, left, right *Expr) *Expr {
	return &Expr{Kind: ExprBinary, Op: op, Left: left, Right: right}
}

// Assign returns an assignment.
func Assign(left, right *Expr) *Expr {
	return &Expr{Kind: ExprAssign, Left: left, Right: right}
}

// Dot returns a member expression.
func Dot(recv *Expr, name string) *Expr {
	return &Expr{Kind: ExprField, Left: recv, Text: name}
}

// Arrow returns a pointer member expression.
func Arrow(recv *Expr, name string) *Expr {
	return &Expr{Kind: ExprField, Left: recv, Text: name, Arrow: true}
}

// Index returns a subscript expression.
func Index(recv, idx *Expr) *Expr {
	return &Expr{Kind: ExprIndex, Left: recv, Right: idx}
}

// Cast returns a cast.
func Cast(typ *Type, x *Expr) *Expr {
	return &Expr{Kind: ExprCast, Type: typ, Left: x}
}

// SizeofType returns sizeof(type).
func SizeofType(typ *Type) *Expr {
	return &Expr{Kind: ExprSizeof, Type: typ}
}

// SizeofExpr returns sizeof expr.
func SizeofExpr(x *Expr) *Expr {
	return &Expr{Kind: ExprSizeof, Left: x}
}

// Block returns a compound statement.
func Block(stmts ...*Stmt) *Stmt {
	return &Stmt{Kind: StmtBlock, Stmts: stmts}
}

// Return returns a return statement. A nil expression prints return.
func Return(e *Expr) *Stmt {
	return &Stmt{Kind: StmtReturn, Expr: e}
}

// If returns an if statement.
func If(cond *Expr, body *Stmt) *Stmt {
	return &Stmt{Kind: StmtIf, Expr: cond, Body: body}
}

// WithElse stores the alternative and returns the if statement.
func (s *Stmt) WithElse(alt *Stmt) *Stmt {
	if s == nil {
		return nil
	}
	s.Else = alt
	return s
}

// While returns a while statement.
func While(cond *Expr, body *Stmt) *Stmt {
	return &Stmt{Kind: StmtWhile, Expr: cond, Body: body}
}

// For returns a for statement.
func For(init *Stmt, cond *Expr, post *Expr, body *Stmt) *Stmt {
	return &Stmt{Kind: StmtFor, Init: init, Expr: cond, Post: post, Body: body}
}

// Break returns a break statement.
func Break() *Stmt { return &Stmt{Kind: StmtBreak} }

// Continue returns a continue statement.
func Continue() *Stmt { return &Stmt{Kind: StmtContinue} }

// ExprStmt returns an expression statement.
func ExprStmt(e *Expr) *Stmt {
	return &Stmt{Kind: StmtExpr, Expr: e}
}

// Declare returns a local declaration.
func Declare(name string, typ *Type) *Stmt {
	return &Stmt{Kind: StmtDecl, Objects: []*Object{{Name: name, Type: typ}}}
}

// WithInit stores an initializer on the first object and returns the statement.
func (s *Stmt) WithInit(e *Expr) *Stmt {
	if s == nil || len(s.Objects) == 0 {
		return s
	}
	s.Objects[0].Init = e
	return s
}
