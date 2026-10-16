package llvm

// Operand is a typed value, as a call argument or an aggregate element.
type Operand struct {
	Type  *Type
	Value *Value
}

// Typed pairs a type with a value.
func Typed(typ *Type, value *Value) Operand {
	return Operand{Type: typ, Value: value}
}

// Incoming is one phi predecessor.
type Incoming struct {
	Value *Value
	Block string
}

// IncomingFrom builds one phi edge.
func IncomingFrom(value *Value, block string) Incoming {
	return Incoming{Value: value, Block: block}
}

// Value is an operand. Locals and globals store the name without the sigil.
// A constant integer or float stores the literal text and its type.
type Value struct {
	Kind        ValueKind
	Name        string
	Text        string
	Type        *Type
	Elems       []Operand
	InBounds    bool
	SourceType  *Type
	PointerType *Type
	Base        *Value
	Indices     []Operand
	Attrs       []string
	Flags       []string
	Body        string
	Dest        *Type
}

// Local is a %name reference.
func Local(name string) *Value { return &Value{Kind: ValLocal, Name: name} }

// GlobalRef is an @name reference.
func GlobalRef(name string) *Value { return &Value{Kind: ValGlobal, Name: name} }

// Int is a constant integer. text is the decimal literal, including a sign.
func Int(bits int, text string) *Value {
	return &Value{Kind: ValInt, Text: text, Type: IntType(bits)}
}

// FloatLit is a constant float. text is the LLVM literal.
func FloatLit(kind Float, text string) *Value {
	return &Value{Kind: ValFloat, Text: text, Type: FP(kind)}
}

// True is the i1 constant true.
func True() *Value { return &Value{Kind: ValTrue} }

// False is the i1 constant false.
func False() *Value { return &Value{Kind: ValFalse} }

// Null is a typed null.
func Null(typ *Type) *Value { return &Value{Kind: ValNull, Type: typ} }

// Zero is a typed zeroinitializer.
func Zero(typ *Type) *Value { return &Value{Kind: ValZero, Type: typ} }

// Undef is a typed undef.
func Undef(typ *Type) *Value { return &Value{Kind: ValUndef, Type: typ} }

// Poison is a typed poison.
func Poison(typ *Type) *Value { return &Value{Kind: ValPoison, Type: typ} }

// CString is a c"..." constant. text is the decoded bytes.
func CString(text string) *Value { return &Value{Kind: ValString, Text: text} }

// StructLit is { elem, elem }.
func StructLit(elems ...Operand) *Value {
	return &Value{Kind: ValStruct, Elems: elems}
}

// PackedLit is <{ elem, elem }>.
func PackedLit(elems ...Operand) *Value {
	return &Value{Kind: ValPacked, Elems: elems}
}

// ArrayLit is [ elem, elem ].
func ArrayLit(elems ...Operand) *Value {
	return &Value{Kind: ValArray, Elems: elems}
}

// VectorLit is < elem, elem >.
func VectorLit(elems ...Operand) *Value {
	return &Value{Kind: ValVector, Elems: elems}
}

// ConstGEP is a constant getelementptr. [Value.Inbounds] sets the flag.
func ConstGEP(source, pointer *Type, base *Value, index ...Operand) *Value {
	return &Value{
		Kind:        ValGEP,
		SourceType:  source,
		PointerType: pointer,
		Base:        base,
		Indices:     index,
	}
}

// Inbounds marks a constant getelementptr inbounds.
func (v *Value) Inbounds() *Value {
	if v != nil {
		v.InBounds = true
	}
	return v
}
