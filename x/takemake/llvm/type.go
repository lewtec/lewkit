package llvm

// Type is one LLVM type. Named types refer to a [TypeDef] by name.
// An opaque pointer has kind ptr and a nil element. A typed pointer
// keeps the element and prints with a star.
type Type struct {
	Kind      Kind
	Bits      int
	Float     Float
	AddrSpace int
	Elem      *Type
	Len       int64
	Packed    bool
	Fields    []*Type
	Ret       *Type
	Variadic  bool
	Name      string
	Scalable  bool
}

// Void is the void type.
func Void() *Type { return &Type{Kind: KindVoid} }

// IntType is an integer of the given width.
func IntType(bits int) *Type { return &Type{Kind: KindInt, Bits: bits} }

// FP is a floating type.
func FP(kind Float) *Type { return &Type{Kind: KindFloat, Float: kind} }

// Ptr is an opaque pointer.
func Ptr() *Type { return &Type{Kind: KindPtr} }

// Pointer is a typed pointer.
func Pointer(elem *Type) *Type { return &Type{Kind: KindPtr, Elem: elem} }

// Space sets the address space and returns the type.
func (t *Type) Space(space int) *Type {
	if t != nil {
		t.AddrSpace = space
	}
	return t
}

// Array is [n x elem].
func Array(n int64, elem *Type) *Type {
	return &Type{Kind: KindArray, Len: n, Elem: elem}
}

// Vector is <n x elem>.
func Vector(n int64, elem *Type) *Type {
	return &Type{Kind: KindVector, Len: n, Elem: elem}
}

// Struct is a literal struct.
func Struct(fields ...*Type) *Type {
	return &Type{Kind: KindStruct, Fields: fields}
}

// PackedStruct is <{ fields }>.
func PackedStruct(fields ...*Type) *Type {
	return &Type{Kind: KindStruct, Packed: true, Fields: fields}
}

// FuncType is a function type. [Type.VarArg] marks the ellipsis.
func FuncType(ret *Type, params ...*Type) *Type {
	return &Type{Kind: KindFunc, Ret: ret, Fields: params}
}

// VarArg marks a function type as variadic.
func (t *Type) VarArg() *Type {
	if t != nil {
		t.Variadic = true
	}
	return t
}

// Named is a reference to a named type, without the % sigil.
func Named(name string) *Type { return &Type{Kind: KindNamed, Name: name} }
