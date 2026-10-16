package golang

// Decl is a const, var, type, or function declaration.
type Decl interface {
	decl()
}

// Field is a parameter, result, struct field, or interface method.
// Variadic is the "..." form, and Type is the element type in that case.
// Tag is the struct tag text without its surrounding quotes.
type Field struct {
	Names    []string
	Type     *Type
	Tag      string
	Variadic bool
}

// NewField returns a field of typ with the given names. No names is an
// unnamed parameter, an embedded field, or an unnamed result.
func NewField(typ *Type, names ...string) *Field {
	if names == nil {
		names = []string{}
	}
	return &Field{Names: names, Type: typ}
}

// WithTag sets the struct tag text.
func (field *Field) WithTag(tag string) *Field {
	if field != nil {
		field.Tag = tag
	}
	return field
}

// ValueSpec is one const or var spec inside a declaration group.
// Values is nil when a later const in an iota group omits its expression.
type ValueSpec struct {
	Names  []string
	Type   *Type
	Values []*Expr
}

// TypeSpec is one type spec inside a type declaration.
// Params are the type parameters.
type TypeSpec struct {
	Name   string
	Alias  bool
	Params []*Field
	Type   *Type
}

// Const is a const declaration. Specs preserves a parenthesized group.
type Const struct {
	file  *File
	Specs []ValueSpec
}

// Var is a var declaration. Specs preserves a parenthesized group.
type Var struct {
	file  *File
	Specs []ValueSpec
}

// TypeDecl is a type declaration.
type TypeDecl struct {
	file  *File
	Specs []TypeSpec
}

// Func is a function or method declaration. HasBody distinguishes
// `func f()` from `func f() {}`. Body is nil when HasBody is false.
type Func struct {
	file       *File
	Receiver   *Field
	Name       string
	TypeParams []*Field
	Parameters []*Field
	Returns    []*Field
	HasBody    bool
	Body       []*Stmt
}

func (*Const) decl()    {}
func (*Var) decl()      {}
func (*TypeDecl) decl() {}
func (*Func) decl()     {}

// Id sets the function name.
func (fn *Func) Id(name string) *Func {
	if fn != nil {
		fn.Name = name
	}
	return fn
}

// Recv sets the method receiver. An empty name is an unnamed receiver.
func (fn *Func) Recv(name string, typ *Type) *Func {
	if fn == nil {
		return nil
	}
	field := &Field{Type: typ}
	if name != "" {
		field.Names = []string{name}
	} else {
		field.Names = []string{}
	}
	fn.Receiver = field
	fn.note()
	return fn
}

// Params sets the parameter list.
func (fn *Func) Params(params ...*Field) *Func {
	if fn == nil {
		return nil
	}
	if params == nil {
		params = []*Field{}
	}
	fn.Parameters = params
	fn.note()
	return fn
}

// Results sets the result list.
func (fn *Func) Results(results ...*Field) *Func {
	if fn == nil {
		return nil
	}
	if results == nil {
		results = []*Field{}
	}
	fn.Returns = results
	fn.note()
	return fn
}

// Block sets the function body. An expression becomes an expression
// statement.
func (fn *Func) Block(lines ...Code) *Func {
	if fn == nil {
		return nil
	}
	fn.HasBody = true
	fn.Body = statements(lines)
	fn.note()
	return fn
}

func (fn *Func) note() {
	if fn != nil && fn.file != nil {
		fn.file.ensureImports()
	}
}

// Id adds a name to the current const spec, or starts a spec.
func (c *Const) Id(name string) *Const {
	if c != nil {
		addSpecName(&c.Specs, name)
	}
	return c
}

// Type sets the type of the current const spec.
func (c *Const) Type(typ *Type) *Const {
	if c != nil {
		setSpecType(&c.Specs, typ)
		c.note()
	}
	return c
}

// Values sets the expressions of the current const spec.
func (c *Const) Values(exprs ...*Expr) *Const {
	if c != nil {
		setSpecValues(&c.Specs, exprs)
		c.note()
	}
	return c
}

func (c *Const) note() {
	if c != nil && c.file != nil {
		c.file.ensureImports()
	}
}

// Id adds a name to the current var spec, or starts a spec.
func (v *Var) Id(name string) *Var {
	if v != nil {
		addSpecName(&v.Specs, name)
	}
	return v
}

// Type sets the type of the current var spec.
func (v *Var) Type(typ *Type) *Var {
	if v != nil {
		setSpecType(&v.Specs, typ)
		v.note()
	}
	return v
}

// Values sets the expressions of the current var spec.
func (v *Var) Values(exprs ...*Expr) *Var {
	if v != nil {
		setSpecValues(&v.Specs, exprs)
		v.note()
	}
	return v
}

func (v *Var) note() {
	if v != nil && v.file != nil {
		v.file.ensureImports()
	}
}

func addSpecName(specs *[]ValueSpec, name string) {
	if len(*specs) == 0 {
		*specs = append(*specs, ValueSpec{})
	}
	cur := &(*specs)[len(*specs)-1]
	if cur.Type != nil || len(cur.Values) > 0 {
		*specs = append(*specs, ValueSpec{})
		cur = &(*specs)[len(*specs)-1]
	}
	cur.Names = append(cur.Names, name)
}

func setSpecType(specs *[]ValueSpec, typ *Type) {
	if len(*specs) == 0 {
		*specs = append(*specs, ValueSpec{})
	}
	(*specs)[len(*specs)-1].Type = typ
}

func setSpecValues(specs *[]ValueSpec, exprs []*Expr) {
	if len(*specs) == 0 {
		*specs = append(*specs, ValueSpec{})
	}
	if exprs == nil {
		exprs = []*Expr{}
	}
	(*specs)[len(*specs)-1].Values = exprs
}

// Id starts a type spec with that name.
func (d *TypeDecl) Id(name string) *TypeDecl {
	if d == nil {
		return nil
	}
	if len(d.Specs) == 0 || d.Specs[len(d.Specs)-1].Name != "" {
		d.Specs = append(d.Specs, TypeSpec{})
	}
	d.Specs[len(d.Specs)-1].Name = name
	return d
}

// Of sets the type of the current spec.
func (d *TypeDecl) Of(typ *Type) *TypeDecl {
	return d.set(typ, false)
}

// Alias sets the current spec to a type alias.
func (d *TypeDecl) Alias(typ *Type) *TypeDecl {
	return d.set(typ, true)
}

// Struct sets the current spec to a struct type.
func (d *TypeDecl) Struct(fields ...*Field) *TypeDecl {
	return d.Of(Struct(fields...))
}

// Interface sets the current spec to an interface type.
func (d *TypeDecl) Interface(fields ...*Field) *TypeDecl {
	return d.Of(Interface(fields...))
}

func (d *TypeDecl) set(typ *Type, alias bool) *TypeDecl {
	if d == nil {
		return nil
	}
	if len(d.Specs) == 0 {
		d.Specs = append(d.Specs, TypeSpec{})
	}
	cur := &d.Specs[len(d.Specs)-1]
	cur.Type = typ
	cur.Alias = alias
	if d.file != nil {
		d.file.ensureImports()
	}
	return d
}
