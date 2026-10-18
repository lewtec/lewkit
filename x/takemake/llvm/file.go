package llvm

import (
	"fmt"
	"io"
)

// TypeDef is one named type. IsOpaque prints `type opaque`.
type TypeDef struct {
	Name     string
	Body     *Type
	IsOpaque bool
}

// Struct sets the body to a literal struct.
func (d *TypeDef) Struct(fields ...*Type) *TypeDef {
	if d == nil {
		return nil
	}
	d.Body = Struct(fields...)
	d.IsOpaque = false
	return d
}

// Type sets the body.
func (d *TypeDef) Type(body *Type) *TypeDef {
	if d == nil {
		return nil
	}
	d.Body = body
	d.IsOpaque = false
	return d
}

// Opaque marks the named type opaque.
func (d *TypeDef) Opaque() *TypeDef {
	if d == nil {
		return nil
	}
	d.Body = nil
	d.IsOpaque = true
	return d
}

// Global is one global variable or constant.
type Global struct {
	Name        string
	Linkage     Linkage
	DSOLocal    bool
	Unnamed     Unnamed
	IsConstant  bool
	ValueType   *Type
	Initializer *Value
	Alignment   int
	Prefix      []string
	Suffix      []string
	Alias       bool
	IFunc       bool
	Aliasee     *Operand
}

// Link sets the linkage.
func (g *Global) Link(linkage Linkage) *Global {
	if g != nil {
		g.Linkage = linkage
	}
	return g
}

// Private sets private linkage.
func (g *Global) Private() *Global { return g.Link(LinkagePrivate) }

// Internal sets internal linkage.
func (g *Global) Internal() *Global { return g.Link(LinkageInternal) }

// External sets external linkage and expects no initializer.
func (g *Global) External() *Global { return g.Link(LinkageExternal) }

// SetDSOLocal marks the global dso_local.
func (g *Global) SetDSOLocal() *Global {
	if g != nil {
		g.DSOLocal = true
	}
	return g
}

// UnnamedAddr sets unnamed_addr.
func (g *Global) UnnamedAddr() *Global {
	if g != nil {
		g.Unnamed = UnnamedAddr
	}
	return g
}

// LocalUnnamedAddr sets local_unnamed_addr.
func (g *Global) LocalUnnamedAddr() *Global {
	if g != nil {
		g.Unnamed = UnnamedLocal
	}
	return g
}

// Constant marks the global as a constant.
func (g *Global) Constant() *Global {
	if g != nil {
		g.IsConstant = true
	}
	return g
}

// Type sets the global's type.
func (g *Global) Type(valueType *Type) *Global {
	if g != nil {
		g.ValueType = valueType
	}
	return g
}

// Init sets the initializer.
func (g *Global) Init(value *Value) *Global {
	if g != nil {
		g.Initializer = value
	}
	return g
}

// Align sets the alignment in bytes.
func (g *Global) Align(n int) *Global {
	if g != nil {
		g.Alignment = n
	}
	return g
}

// Param is one function parameter. An empty name is unnamed.
// Attrs are the source text of each parameter attribute.
type Param struct {
	Name  string
	Type  *Type
	Attrs []string
}

// Func is a declaration or a definition. Defined is false for a declaration.
type Func struct {
	Name        string
	Linkage     Linkage
	DSOLocal    bool
	Unnamed     Unnamed
	Result      *Type
	Parameters  []*Param
	Variadic    bool
	Defined     bool
	Blocks      []*Block
	Prefix      []string
	Conv        string
	ResultAttrs []string
	Suffix      []string
	Attach      []Attach
}

// Link sets the linkage.
func (fn *Func) Link(linkage Linkage) *Func {
	if fn != nil {
		fn.Linkage = linkage
	}
	return fn
}

// Private sets private linkage.
func (fn *Func) Private() *Func { return fn.Link(LinkagePrivate) }

// Internal sets internal linkage.
func (fn *Func) Internal() *Func { return fn.Link(LinkageInternal) }

// SetDSOLocal marks the function dso_local.
func (fn *Func) SetDSOLocal() *Func {
	if fn != nil {
		fn.DSOLocal = true
	}
	return fn
}

// UnnamedAddr sets unnamed_addr.
func (fn *Func) UnnamedAddr() *Func {
	if fn != nil {
		fn.Unnamed = UnnamedAddr
	}
	return fn
}

// Returns sets the result type.
func (fn *Func) Returns(valueType *Type) *Func {
	if fn != nil {
		fn.Result = valueType
	}
	return fn
}

// Param appends a parameter. An empty name is unnamed.
func (fn *Func) Param(name string, valueType *Type) *Func {
	if fn == nil {
		return nil
	}
	fn.Parameters = append(fn.Parameters, &Param{Name: name, Type: valueType})
	return fn
}

// VarArg marks the signature variadic.
func (fn *Func) VarArg() *Func {
	if fn != nil {
		fn.Variadic = true
	}
	return fn
}

// Block appends a basic block and returns it. The function becomes a definition.
func (fn *Func) Block(name string) *Block {
	block := &Block{Name: name}
	if fn != nil {
		fn.Defined = true
		fn.Blocks = append(fn.Blocks, block)
	}
	return block
}

// Block is one basic block.
type Block struct {
	Name         string
	Instructions []*Inst
}

func (b *Block) append(in *Inst) *Inst {
	if b != nil {
		b.Instructions = append(b.Instructions, in)
	}
	return in
}

// Alloca appends an alloca of valueType.
func (b *Block) Alloca(name string, valueType *Type) *Inst {
	return b.append(&Inst{Kind: InstAlloca, Name: name, Type: valueType})
}

// Load appends a load of the pointer.
func (b *Block) Load(name string, valueType, pointer *Type, value *Value) *Inst {
	return b.append(&Inst{Kind: InstLoad, Name: name, Type: valueType, Pointer: pointer, Left: value})
}

// Store appends a store.
func (b *Block) Store(valueType *Type, value *Value, pointer *Type, dest *Value) *Inst {
	return b.append(&Inst{Kind: InstStore, Type: valueType, Left: value, Pointer: pointer, Right: dest})
}

// GEP appends a getelementptr. The result type is an opaque pointer in the text.
func (b *Block) GEP(name string, source, pointer *Type, base *Value, index ...Operand) *Inst {
	return b.append(&Inst{
		Kind:    InstGEP,
		Name:    name,
		Source:  source,
		Pointer: pointer,
		Left:    base,
		Indices: index,
	})
}

// Bin appends a binary instruction.
func (b *Block) Bin(name string, op Op, valueType *Type, left, right *Value) *Inst {
	return b.append(&Inst{Kind: InstBin, Name: name, Op: op, Type: valueType, Left: left, Right: right})
}

// ICmp appends an integer comparison.
func (b *Block) ICmp(name string, pred IPred, valueType *Type, left, right *Value) *Inst {
	return b.append(&Inst{Kind: InstICmp, Name: name, IntegerPred: pred, Type: valueType, Left: left, Right: right})
}

// FCmp appends a floating comparison.
func (b *Block) FCmp(name string, pred FPred, valueType *Type, left, right *Value) *Inst {
	return b.append(&Inst{Kind: InstFCmp, Name: name, FloatPred: pred, Type: valueType, Left: left, Right: right})
}

// Cast appends a conversion from valueType to dest.
func (b *Block) Cast(name string, kind Cast, valueType *Type, value *Value, dest *Type) *Inst {
	return b.append(&Inst{Kind: InstCast, Name: name, Cast: kind, Type: valueType, Left: value, Dest: dest})
}

// Phi appends a phi of the incoming edges.
func (b *Block) Phi(name string, valueType *Type, incoming ...Incoming) *Inst {
	return b.append(&Inst{Kind: InstPhi, Name: name, Type: valueType, Incoming: incoming})
}

// Extract appends extractvalue. path is the sequence of literal indices.
func (b *Block) Extract(name string, valueType *Type, value *Value, path ...int) *Inst {
	return b.append(&Inst{Kind: InstExtract, Name: name, Type: valueType, Left: value, Path: path})
}

// Insert appends insertvalue.
func (b *Block) Insert(name string, aggType *Type, agg *Value, elem Operand, path ...int) *Inst {
	return b.append(&Inst{
		Kind:     InstInsert,
		Name:     name,
		Type:     aggType,
		Left:     agg,
		Inserted: &elem,
		Path:     path,
	})
}

// Call appends a call. An empty name is a void call.
func (b *Block) Call(name string, ret *Type, callee *Value, arg ...Operand) *Inst {
	return b.append(&Inst{Kind: InstCall, Name: name, Type: ret, Callee: callee, Args: arg})
}

// CallSig appends a call that spells the callee signature.
func (b *Block) CallSig(name string, sig *Type, callee *Value, arg ...Operand) *Inst {
	in := b.Call(name, nil, callee, arg...)
	if in == nil {
		return nil
	}
	in.Signature = sig
	if sig != nil {
		in.Type = sig.Ret
	}
	return in
}

// Br appends an unconditional branch.
func (b *Block) Br(label string) *Inst {
	return b.append(&Inst{Kind: InstBr, Target: label})
}

// CondBr appends a conditional branch.
func (b *Block) CondBr(cond *Value, ifTrue, ifFalse string) *Inst {
	return b.append(&Inst{Kind: InstBr, Condition: cond, TrueLabel: ifTrue, FalseLabel: ifFalse})
}

// Ret appends a return of value.
func (b *Block) Ret(valueType *Type, value *Value) *Inst {
	return b.append(&Inst{Kind: InstRet, Type: valueType, Left: value})
}

// RetVoid appends `ret void`.
func (b *Block) RetVoid() *Inst {
	return b.append(&Inst{Kind: InstRet, Type: Void()})
}

// Unreachable appends unreachable.
func (b *Block) Unreachable() *Inst {
	return b.append(&Inst{Kind: InstUnreachable})
}

// Inst is one instruction. Which fields are set depends on Kind.
type Inst struct {
	Kind           InstKind
	Name           string
	Op             Op
	IntegerPred    IPred
	FloatPred      FPred
	Cast           Cast
	Tail           Tail
	NoSignedWrap   bool
	NoUnsignedWrap bool
	ExactDiv       bool
	InBounds       bool
	IsVolatile     bool
	Type           *Type
	Pointer        *Type
	Source         *Type
	Dest           *Type
	Left           *Value
	Right          *Value
	Indices        []Operand
	Alignment      int
	Count          *Operand
	Callee         *Value
	Signature      *Type
	Args           []Operand
	Incoming       []Incoming
	Inserted       *Operand
	Path           []int
	Condition      *Value
	TrueLabel      string
	FalseLabel     string
	Target         string
	Flags          []string
	Conv           string
	ResultAttrs    []string
	Groups         []string
	Bundles        []string
	Attach         []Attach
	Ordering       string
	FailureOrd     string
	AtomicOp       string
	SyncScope      string
	Weak           bool
	Atomic         bool
	CondType       *Type
	Index          *Operand
	Other          *Operand
	Labels         []string
	Clauses        []Clause
	Arms           []Arm
}

// Attach is one metadata attachment. Body is the metadata value, including its bang.
type Attach struct {
	Name string
	Body string
}

// Clause is one landingpad clause. Cleanup has an empty type.
type Clause struct {
	Kind  string
	Type  *Type
	Value *Value
}

// Arm is one switch case.
type Arm struct {
	Type  *Type
	Value *Value
	Label string
}

// Comdat is one comdat definition.
type Comdat struct {
	Name string
	Kind ComdatKind
}

// AttrGroup is one attribute group. Body includes the braces.
type AttrGroup struct {
	ID   string
	Body string
}

// MetaDef is one metadata definition. RHS is the text after '='.
type MetaDef struct {
	Name string
	RHS  string
}

// NSW sets the no-signed-wrap flag.
func (in *Inst) NSW() *Inst {
	if in != nil {
		in.NoSignedWrap = true
	}
	return in
}

// NUW sets the no-unsigned-wrap flag.
func (in *Inst) NUW() *Inst {
	if in != nil {
		in.NoUnsignedWrap = true
	}
	return in
}

// Exact sets the exact flag.
func (in *Inst) Exact() *Inst {
	if in != nil {
		in.ExactDiv = true
	}
	return in
}

// Inbounds marks getelementptr inbounds.
func (in *Inst) Inbounds() *Inst {
	if in != nil {
		in.InBounds = true
	}
	return in
}

// Volatile marks a volatile load or store.
func (in *Inst) Volatile() *Inst {
	if in != nil {
		in.IsVolatile = true
	}
	return in
}

// Align sets the alignment in bytes.
func (in *Inst) Align(n int) *Inst {
	if in != nil {
		in.Alignment = n
	}
	return in
}

// Elements sets the alloca element count.
func (in *Inst) Elements(count Operand) *Inst {
	if in != nil {
		in.Count = &count
	}
	return in
}

// TailCall marks the call tail.
func (in *Inst) TailCall() *Inst {
	if in != nil {
		in.Tail = TailTail
	}
	return in
}

// MustTail marks the call musttail.
func (in *Inst) MustTail() *Inst {
	if in != nil {
		in.Tail = TailMust
	}
	return in
}

// NoTail marks the call notail.
func (in *Inst) NoTail() *Inst {
	if in != nil {
		in.Tail = TailNo
	}
	return in
}

// File is one LLVM IR module.
type File struct {
	SourceFilename string
	DataLayout     string
	TargetTriple   string
	Asms           []string
	Types          []*TypeDef
	Comdats        []*Comdat
	Globals        []*Global
	Funcs          []*Func
	AttrGroups     []*AttrGroup
	Metadata       []*MetaDef
}

// New returns an empty module.
func New() *File { return &File{} }

// Source sets the source filename.
func (f *File) Source(name string) *File {
	if f != nil {
		f.SourceFilename = name
	}
	return f
}

// Layout sets the data layout.
func (f *File) Layout(layout string) *File {
	if f != nil {
		f.DataLayout = layout
	}
	return f
}

// Triple sets the target triple.
func (f *File) Triple(triple string) *File {
	if f != nil {
		f.TargetTriple = triple
	}
	return f
}

// Type appends a named type and returns it.
func (f *File) Type(name string) *TypeDef {
	def := &TypeDef{Name: name}
	if f != nil {
		f.Types = append(f.Types, def)
	}
	return def
}

// Global appends a global and returns it.
func (f *File) Global(name string) *Global {
	g := &Global{Name: name}
	if f != nil {
		f.Globals = append(f.Globals, g)
	}
	return g
}

// Func appends a function and returns it. With no blocks it is a declaration.
func (f *File) Func(name string) *Func {
	fn := &Func{Name: name}
	if f != nil {
		f.Funcs = append(f.Funcs, fn)
	}
	return fn
}

// Take reads LLVM IR assembly into a [File].
func Take(r io.Reader) (*File, error) {
	if r == nil {
		return nil, ErrNilReader
	}
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRead, err)
	}
	return parse(string(raw))
}

// Make writes the module as LLVM IR assembly.
func (f *File) Make(w io.Writer) error {
	if f == nil {
		return ErrNilFile
	}
	if w == nil {
		return ErrNilWriter
	}
	if err := f.validate(); err != nil {
		return err
	}
	s := &sink{w: w}
	f.write(s)
	return s.err
}
