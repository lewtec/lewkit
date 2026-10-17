package glsl

import "slices"

// File is one GLSL shader.
type File struct {
	Stage      Stage
	Version    int
	Profile    Profile
	Extensions []Extension
	Precisions []Precision
	Structs    []*Struct
	Globals    []*Decl
	Functions  []*Func
}

// Extension is one #extension line.
type Extension struct {
	Name     string
	Behavior Behavior
}

// Precision is one precision statement.
type Precision struct {
	Rank Rank
	Type string
}

// Struct is one struct declaration.
type Struct struct {
	Name   string
	Fields []*Decl
}

// Layout is one item inside layout(...). A nil Value is a bare identifier.
type Layout struct {
	Name  string
	Value Expr
}

// Size is one array dimension. A nil Expr is unsized.
type Size struct {
	Expr Expr
}

// Type is a type name and its array dimensions.
type Type struct {
	Name  string
	Sizes []Size
}

// Decl is a global, a block field, a local variable, or an interface block.
type Decl struct {
	Layouts   []Layout
	Invariant bool
	Precise   bool
	Interp    Interp
	Aux       Aux
	Memory    []Memory
	Rank      Rank
	Storage   Storage
	Block     string
	Fields    []*Decl
	Instance  string
	Sizes     []Size
	Type      Type
	Name      string
	Init      Expr
}

// Func is one function. A nil Body is a prototype.
type Func struct {
	Rank   Rank
	Return Type
	Name   string
	Params []*Param
	Body   []Stmt
}

// Param is one function parameter. An empty Name is unnamed.
type Param struct {
	Const   bool
	Storage Storage
	Rank    Rank
	Memory  []Memory
	Type    Type
	Name    string
}

// Expr is one expression.
type Expr interface{ expr() }

// Ident is a name.
type Ident struct{ Name string }

// Literal is a bool, int, uint, float, or double constant.
type Literal struct {
	Kind  Kind
	Bool  bool
	Int   int64
	Uint  uint64
	Float float64
}

// Unary is a prefix operator.
type Unary struct {
	Op   string
	Expr Expr
}

// Binary is an infix operator, including assignment.
type Binary struct {
	Op    string
	Left  Expr
	Right Expr
}

// Ternary is cond ? then : else.
type Ternary struct {
	Cond Expr
	Then Expr
	Else Expr
}

// Call is a function call or a constructor.
type Call struct {
	Callee Expr
	Args   []Expr
}

// Index is base[index].
type Index struct {
	Base  Expr
	Index Expr
}

// Select is expr.name, including a swizzle.
type Select struct {
	Expr Expr
	Name string
}

// Postfix is expr++ or expr--.
type Postfix struct {
	Expr Expr
	Op   string
}

// List is a brace initializer.
type List struct{ Exprs []Expr }

func (*Ident) expr()   {}
func (*Literal) expr() {}
func (*Unary) expr()   {}
func (*Binary) expr()  {}
func (*Ternary) expr() {}
func (*Call) expr()    {}
func (*Index) expr()   {}
func (*Select) expr()  {}
func (*Postfix) expr() {}
func (*List) expr()    {}

// Stmt is one statement.
type Stmt interface{ stmt() }

// Block is a brace list that is not a function body.
type Block struct{ Body []Stmt }

// If is a conditional. Else is nil when the source has no else.
// SetElse stores that branch.
type If struct {
	Cond Expr
	Body []Stmt
	Else []Stmt
}

// For is a for loop. Init is a [*Decl] or an [*ExprStmt].
type For struct {
	Init Stmt
	Cond Expr
	Post Expr
	Body []Stmt
}

// While is a while loop.
type While struct {
	Cond Expr
	Body []Stmt
}

// Do is a do-while loop.
type Do struct {
	Body []Stmt
	Cond Expr
}

// Switch is a switch.
type Switch struct {
	Expr  Expr
	Cases []*Case
}

// Case is one case or default label and the statements until the next label.
type Case struct {
	Expr    Expr
	Default bool
	Body    []Stmt
}

// Return is a return statement. A nil Expr returns nothing.
type Return struct{ Expr Expr }

// Break is a break statement.
type Break struct{}

// Continue is a continue statement.
type Continue struct{}

// Discard is a discard statement.
type Discard struct{}

// Empty is a bare semicolon.
type Empty struct{}

// ExprStmt is an expression used as a statement.
type ExprStmt struct{ Expr Expr }

func (*Decl) stmt()     {}
func (*Block) stmt()    {}
func (*If) stmt()       {}
func (*For) stmt()      {}
func (*While) stmt()    {}
func (*Do) stmt()       {}
func (*Switch) stmt()   {}
func (*Return) stmt()   {}
func (*Break) stmt()    {}
func (*Continue) stmt() {}
func (*Discard) stmt()  {}
func (*Empty) stmt()    {}
func (*ExprStmt) stmt() {}

// New returns an empty shader for stage.
func New(stage Stage) *File {
	return &File{Stage: stage}
}

// SetVersion sets the #version number.
func (file *File) SetVersion(version int) *File {
	if file == nil {
		return nil
	}
	file.Version = version
	return file
}

// SetProfile sets the #version profile.
func (file *File) SetProfile(profile Profile) *File {
	if file == nil {
		return nil
	}
	file.Profile = profile
	return file
}

// Extension stores one #extension line.
func (file *File) Extension(name string, behavior Behavior) *File {
	if file == nil {
		return nil
	}
	file.Extensions = append(file.Extensions, Extension{Name: name, Behavior: behavior})
	return file
}

// Precision stores one precision statement.
func (file *File) Precision(rank Rank, typeName string) *File {
	if file == nil {
		return nil
	}
	file.Precisions = append(file.Precisions, Precision{Rank: rank, Type: typeName})
	return file
}

// Struct stores a struct and returns it.
func (file *File) Struct(name string) *Struct {
	item := &Struct{Name: name}
	if file != nil {
		file.Structs = append(file.Structs, item)
	}
	return item
}

// Field stores a struct field and returns the struct.
func (item *Struct) Field(name string, typ Type) *Struct {
	if item == nil {
		return nil
	}
	item.Fields = append(item.Fields, &Decl{Type: typ, Name: name})
	return item
}

// LocalSize stores layout(local_size_*) in. A zero component is omitted.
func (file *File) LocalSize(x, y, z int) *File {
	if file == nil {
		return nil
	}
	decl := &Decl{Storage: StorageIn}
	decl = decl.layoutDim("local_size_x", x)
	decl = decl.layoutDim("local_size_y", y)
	decl = decl.layoutDim("local_size_z", z)
	file.Globals = append(file.Globals, decl)
	return file
}

func (decl *Decl) layoutDim(name string, value int) *Decl {
	if value == 0 {
		return decl
	}
	return decl.Layout(name, IntLit(int64(value)))
}

// Uniform stores a uniform variable and returns it.
func (file *File) Uniform(name string, typ Type) *Decl {
	return file.decl(StorageUniform, name, typ)
}

// In stores an input variable and returns it.
func (file *File) In(name string, typ Type) *Decl {
	return file.decl(StorageIn, name, typ)
}

// Out stores an output variable and returns it.
func (file *File) Out(name string, typ Type) *Decl {
	return file.decl(StorageOut, name, typ)
}

// Buffer stores a buffer variable and returns it.
func (file *File) Buffer(name string, typ Type) *Decl {
	return file.decl(StorageBuffer, name, typ)
}

// Shared stores a shared variable and returns it.
func (file *File) Shared(name string, typ Type) *Decl {
	return file.decl(StorageShared, name, typ)
}

// Const stores a const and returns it.
func (file *File) Const(name string, typ Type, value Expr) *Decl {
	decl := file.decl(StorageConst, name, typ)
	if decl != nil {
		decl.Init = value
	}
	return decl
}

func (file *File) decl(storage Storage, name string, typ Type) *Decl {
	decl := &Decl{Storage: storage, Type: typ, Name: name}
	if file != nil {
		file.Globals = append(file.Globals, decl)
	}
	return decl
}

// UniformBlock stores a named uniform block and returns it.
func (file *File) UniformBlock(name string) *Decl {
	return file.block(StorageUniform, name)
}

// BufferBlock stores a named buffer block and returns it.
func (file *File) BufferBlock(name string) *Decl {
	return file.block(StorageBuffer, name)
}

func (file *File) block(storage Storage, name string) *Decl {
	decl := &Decl{Storage: storage, Block: name}
	if file != nil {
		file.Globals = append(file.Globals, decl)
	}
	return decl
}

// Layout appends one layout item.
func (decl *Decl) Layout(name string, value Expr) *Decl {
	if decl == nil {
		return nil
	}
	decl.Layouts = append(decl.Layouts, Layout{Name: name, Value: value})
	return decl
}

// Flat sets the flat interpolation qualifier.
func (decl *Decl) Flat() *Decl {
	if decl == nil {
		return nil
	}
	decl.Interp = Flat
	return decl
}

// Readonly adds the readonly memory qualifier.
func (decl *Decl) Readonly() *Decl {
	return decl.memory(Readonly)
}

// Writeonly adds the writeonly memory qualifier.
func (decl *Decl) Writeonly() *Decl {
	return decl.memory(Writeonly)
}

func (decl *Decl) memory(kind Memory) *Decl {
	if decl == nil {
		return nil
	}
	decl.Memory = addMemory(decl.Memory, kind)
	return decl
}

// Field stores a block field and returns the block.
func (decl *Decl) Field(name string, typ Type) *Decl {
	if decl == nil {
		return nil
	}
	decl.Fields = append(decl.Fields, &Decl{Type: typ, Name: name})
	return decl
}

// SetInstance sets the block instance name.
func (decl *Decl) SetInstance(name string) *Decl {
	if decl == nil {
		return nil
	}
	decl.Instance = name
	return decl
}

// Value sets a variable initializer.
func (decl *Decl) Value(expr Expr) *Decl {
	if decl == nil {
		return nil
	}
	decl.Init = expr
	return decl
}

// Func stores a function and returns it.
func (file *File) Func(name string) *Func {
	fn := &Func{Name: name}
	if file != nil {
		file.Functions = append(file.Functions, fn)
	}
	return fn
}

// Returns sets the function result type.
func (fn *Func) Returns(typ Type) *Func {
	if fn == nil {
		return nil
	}
	fn.Return = typ
	return fn
}

// Param stores a parameter and returns the function.
func (fn *Func) Param(name string, typ Type) *Func {
	if fn == nil {
		return nil
	}
	fn.Params = append(fn.Params, &Param{Type: typ, Name: name})
	return fn
}

// Block sets the function body. A nil list is an empty body, not a prototype.
func (fn *Func) Block(stmts ...Stmt) *Func {
	if fn == nil {
		return nil
	}
	if stmts == nil {
		fn.Body = []Stmt{}
		return fn
	}
	fn.Body = stmts
	return fn
}

// Declare builds a local or field declaration.
func Declare(name string, typ Type, value Expr) *Decl {
	return &Decl{Type: typ, Name: name, Init: value}
}

// Exprs builds an expression statement.
func Exprs(expr Expr) *ExprStmt {
	return &ExprStmt{Expr: expr}
}

// IfThen builds a conditional. Else and ElseIf add the other branch.
func IfThen(cond Expr, body ...Stmt) *If {
	return &If{Cond: cond, Body: bodyList(body)}
}

// SetElse sets the else branch.
func (item *If) SetElse(body ...Stmt) *If {
	if item == nil {
		return nil
	}
	item.Else = bodyList(body)
	return item
}

// ElseIf sets the else branch to another conditional.
func (item *If) ElseIf(cond Expr, body ...Stmt) *If {
	if item == nil {
		return nil
	}
	return item.SetElse(IfThen(cond, body...))
}

// ForLoop builds a for loop.
func ForLoop(init Stmt, cond Expr, post Expr, body ...Stmt) *For {
	return &For{Init: init, Cond: cond, Post: post, Body: bodyList(body)}
}

// WhileLoop builds a while loop.
func WhileLoop(cond Expr, body ...Stmt) *While {
	return &While{Cond: cond, Body: bodyList(body)}
}

// ReturnOf builds a return statement. A nil expr returns nothing.
func ReturnOf(expr Expr) *Return {
	return &Return{Expr: expr}
}

func bodyList(stmts []Stmt) []Stmt {
	if stmts == nil {
		return []Stmt{}
	}
	return stmts
}

func addMemory(list []Memory, kind Memory) []Memory {
	for _, item := range list {
		if item == kind {
			return list
		}
	}
	list = append(list, kind)
	slices.Sort(list)
	return list
}

func namedType(name string) Type {
	return Type{Name: name}
}

// Void is the void type.
func Void() Type { return namedType("void") }

// Bool is the bool type.
func Bool() Type { return namedType("bool") }

// Int is the int type.
func Int() Type { return namedType("int") }

// Uint is the uint type.
func Uint() Type { return namedType("uint") }

// Float is the float type.
func Float() Type { return namedType("float") }

// Double is the double type.
func Double() Type { return namedType("double") }

// Vec2 is vec2.
func Vec2() Type { return namedType("vec2") }

// Vec3 is vec3.
func Vec3() Type { return namedType("vec3") }

// Vec4 is vec4.
func Vec4() Type { return namedType("vec4") }

// IVec2 is ivec2.
func IVec2() Type { return namedType("ivec2") }

// IVec3 is ivec3.
func IVec3() Type { return namedType("ivec3") }

// IVec4 is ivec4.
func IVec4() Type { return namedType("ivec4") }

// UVec2 is uvec2.
func UVec2() Type { return namedType("uvec2") }

// UVec3 is uvec3.
func UVec3() Type { return namedType("uvec3") }

// UVec4 is uvec4.
func UVec4() Type { return namedType("uvec4") }

// BVec4 is bvec4.
func BVec4() Type { return namedType("bvec4") }

// Mat3 is mat3.
func Mat3() Type { return namedType("mat3") }

// Mat4 is mat4.
func Mat4() Type { return namedType("mat4") }

// Sampler2D is sampler2D.
func Sampler2D() Type { return namedType("sampler2D") }

// Image2D is image2D.
func Image2D() Type { return namedType("image2D") }

// Array adds a sized dimension.
func (typ Type) Array(size Expr) Type {
	typ.Sizes = append(append([]Size{}, typ.Sizes...), Size{Expr: size})
	return typ
}

// Unsized adds a dimension with no length.
func (typ Type) Unsized() Type {
	typ.Sizes = append(append([]Size{}, typ.Sizes...), Size{})
	return typ
}

// Id builds a name expression.
func Id(name string) Expr { return &Ident{Name: name} }

// IntLit builds a signed integer literal.
func IntLit(value int64) Expr {
	return &Literal{Kind: KindInt, Int: value}
}

// UintLit builds an unsigned integer literal.
func UintLit(value uint64) Expr {
	return &Literal{Kind: KindUint, Uint: value}
}

// FloatLit builds a float literal.
func FloatLit(value float64) Expr {
	return &Literal{Kind: KindFloat, Float: value}
}

// True builds the true literal.
func True() Expr { return &Literal{Kind: KindBool, Bool: true} }

// False builds the false literal.
func False() Expr { return &Literal{Kind: KindBool, Bool: false} }

// CallOf builds a call or a constructor.
func CallOf(callee Expr, args ...Expr) Expr {
	return &Call{Callee: callee, Args: args}
}

// Dot builds a field or swizzle selection.
func Dot(expr Expr, name string) Expr {
	return &Select{Expr: expr, Name: name}
}

// At builds an index expression.
func At(base Expr, index Expr) Expr {
	return &Index{Base: base, Index: index}
}

// Op builds an infix expression, including assignment.
func Op(op string, left Expr, right Expr) Expr {
	return &Binary{Op: op, Left: left, Right: right}
}

// Not builds a logical negation.
func Not(expr Expr) Expr { return &Unary{Op: "!", Expr: expr} }

// Neg builds an arithmetic negation.
func Neg(expr Expr) Expr { return &Unary{Op: "-", Expr: expr} }
