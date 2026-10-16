package golang

// Type is one Go type. The kind selects the fields in use.
//
//	Named      Name. Path is the import path of a package qualifier.
//	           X is the qualifier when it is not an import.
//	Pointer    Elem
//	Slice      Elem
//	Array      Elem, Len, Ellipsis when the length is [...]
//	Map        Key, Elem
//	Struct     Fields
//	Interface  Fields
//	Func       Params, Fields, Results
//	Chan       Elem, Dir
//	Inst       X, Args. Listed is an index-list instantiation.
//	Binary     Op, Left, Right
//	Unary      Op, Elem
type Type struct {
	Kind     TypeKind
	Name     string
	Path     string
	X        *Expr
	Elem     *Type
	Key      *Type
	Left     *Type
	Right    *Type
	Len      *Expr
	Ellipsis bool
	Listed   bool
	Op       Op
	Fields   []*Field
	Results  []*Field
	Params   []*Field
	Args     []*Type
	Dir      ChanDir
}

// Named returns a named type, including predeclared names.
func Named(name string) *Type {
	return &Type{Kind: TypeNamed, Name: name}
}

// QualType returns a package-qualified named type.
func QualType(path, name string) *Type {
	return &Type{Kind: TypeNamed, Path: path, Name: name}
}

// Pointer returns a pointer type.
func Pointer(elem *Type) *Type {
	return &Type{Kind: TypePointer, Elem: elem}
}

// Slice returns a slice type.
func Slice(elem *Type) *Type {
	return &Type{Kind: TypeSlice, Elem: elem}
}

// Array returns an array type. Set [Type.Ellipsis] for [...].
func Array(len *Expr, elem *Type) *Type {
	return &Type{Kind: TypeArray, Len: len, Elem: elem}
}

// MapType returns a map type.
func MapType(key, elem *Type) *Type {
	return &Type{Kind: TypeMap, Key: key, Elem: elem}
}

// Struct returns a struct type.
func Struct(fields ...*Field) *Type {
	if fields == nil {
		fields = []*Field{}
	}
	return &Type{Kind: TypeStruct, Fields: fields}
}

// Interface returns an interface type.
func Interface(fields ...*Field) *Type {
	if fields == nil {
		fields = []*Field{}
	}
	return &Type{Kind: TypeInterface, Fields: fields}
}

// FuncType returns a function type.
func FuncType(params, results []*Field) *Type {
	if params == nil {
		params = []*Field{}
	}
	if results == nil {
		results = []*Field{}
	}
	return &Type{Kind: TypeFunc, Params: []*Field{}, Fields: params, Results: results}
}

// Chan returns a channel type.
func Chan(dir ChanDir, elem *Type) *Type {
	if dir == 0 {
		dir = ChanBoth
	}
	return &Type{Kind: TypeChan, Dir: dir, Elem: elem}
}

// AsExpr returns the type where an expression needs a type node,
// such as a conversion.
func (t *Type) AsExpr() *Expr {
	return &Expr{Kind: ExprType, Type: t}
}
