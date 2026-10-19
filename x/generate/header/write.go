package header

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/lewtec/lewkit/x/generate"
	"github.com/lewtec/lewkit/x/takemake/golang"
)

var generatedBy = generate.Notice("header")

const nativePath = "github.com/lewtec/lewkit/x/ffi/native"

// Write renders unit as a Go source file for package pkg.
// libs are the sonames Open asks native.Bind to dlopen. Function symbols
// are fields of the struct named by BindName.
// BindSoft marks every field soft so a missing symbol stays nil.
func Write(unit *Unit, pkg string, libs []string) ([]byte, error) {
	if unit == nil {
		return nil, errHeader
	}
	if pkg == "" {
		return nil, errPackage
	}
	if err := bindFuncName(unit); err != nil {
		return nil, err
	}
	assignNames(unit)
	w := &writer{unit: unit, ptr: unit.Target.Ptr, libs: libs}
	if w.ptr < 1 {
		w.ptr = 8
	}
	file := golang.NewFile(pkg)
	file.Header = w.banner()
	w.emit(file)
	var buf bytes.Buffer
	if err := file.Make(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type writer struct {
	unit *Unit
	ptr  int
	libs []string
	raw  bool
}

func (w *writer) banner() string {
	var b strings.Builder
	fmt.Fprintf(&b, "//go:build %s\n\n", w.unit.Target.Constraint())
	b.WriteString("// ")
	b.WriteString(generatedBy)
	b.WriteByte('\n')
	fmt.Fprintf(&b, "// ABI %s. Field offsets follow that C ABI.\n", w.unit.Target.ID)
	fmt.Fprintf(&b, "// native.Bind[%s] loads a library and fills that struct.", w.unit.BindName)
	if w.unit.BindSoft {
		b.WriteString(" Missing symbols stay nil.")
	}
	if len(w.unit.Skipped) > 0 {
		b.WriteString("\n// Skipped variadic: ")
		b.WriteString(strings.Join(w.unit.Skipped, ", "))
	}
	return b.String()
}

func (w *writer) emit(file *golang.File) {
	for _, t := range w.unit.Types {
		if t.Kind == kindEnum {
			w.emitEnum(file, t)
		}
	}
	for _, c := range w.unit.Consts {
		file.Const().Id(c.Go).Values(golang.Lit(c.Value))
	}
	for _, t := range w.unit.Types {
		switch t.Kind {
		case kindStruct, kindUnion:
			w.emitAggregate(file, t)
		case kindAlias:
			file.Type().Id(t.Go).Alias(w.goType(t.Elem))
		}
	}
	if len(w.unit.Funcs) > 0 {
		w.emitSymbols(file)
	}
	if len(w.libs) > 0 && len(w.unit.Funcs) > 0 {
		w.emitOpen(file)
	}
	for i := range w.unit.Objects {
		w.emitObject(file, &w.unit.Objects[i])
	}
	for _, t := range w.unit.Types {
		if t.Kind != kindStruct && t.Kind != kindUnion {
			continue
		}
		if t.Incomplete || t.natural(w.ptr) {
			continue
		}
		w.raw = true
		w.emitMethods(file, t)
	}
	if w.raw {
		file.Func().Id("copyAt").Params(
			golang.NewField(golang.Slice(golang.Named("byte")), "dst"),
			golang.NewField(golang.Slice(golang.Named("byte")), "src"),
		).Block(
			golang.Id("copy").Call(golang.Id("dst"), golang.Id("src")),
		)
	}
}

func (w *writer) emitEnum(file *golang.File, t *Type) {
	file.Type().Id(t.Go).Of(intCode(t.Size))
	for _, e := range t.Enums {
		file.Const().Id(e.Go).Type(golang.Named(t.Go)).Values(golang.Lit(e.Value))
	}
}

func intCode(size int) *golang.Type {
	switch size {
	case 1:
		return golang.Named("int8")
	case 2:
		return golang.Named("int16")
	case 8:
		return golang.Named("int64")
	default:
		return golang.Named("int32")
	}
}

func (w *writer) emitAggregate(file *golang.File, t *Type) {
	if t.Incomplete {
		file.Type().Id(t.Go).Struct()
		return
	}
	var fields []*golang.Field
	if t.natural(w.ptr) {
		fields = w.naturalFields(t)
	} else {
		fields = rawFields(t, w.ptr)
	}
	file.Type().Id(t.Go).Struct(fields...)
}

func (w *writer) naturalFields(t *Type) []*golang.Field {
	fields := []*golang.Field{golang.NewField(golang.QualType("structs", "HostLayout"), "_")}
	off := 0
	for _, f := range t.Fields {
		if f == nil || f.Go == "" {
			continue
		}
		if f.Offset > off {
			fields = append(fields, golang.NewField(golang.Array(golang.Lit(f.Offset-off), golang.Named("byte")), "_"))
			off = f.Offset
		}
		fields = append(fields, golang.NewField(w.goType(f.Type), f.Go))
		off += f.CSize
	}
	if t.Size > off {
		fields = append(fields, golang.NewField(golang.Array(golang.Lit(t.Size-off), golang.Named("byte")), "_"))
	}
	return fields
}

func rawFields(t *Type, ptr int) []*golang.Field {
	fields := []*golang.Field{golang.NewField(golang.QualType("structs", "HostLayout"), "_")}
	if t.Size <= 0 {
		return fields
	}
	word, n := anchorWord(anchorAlign(t, ptr), t.Size)
	fields = append(fields, golang.NewField(word, "_"))
	if tail := t.Size - n; tail > 0 {
		fields = append(fields, golang.NewField(golang.Array(golang.Lit(tail), golang.Named("byte")), "_"))
	}
	return fields
}

func anchorWord(align, size int) (*golang.Type, int) {
	switch {
	case align >= 8 && size >= 8:
		return golang.Named("uint64"), 8
	case align >= 4 && size >= 4:
		return golang.Named("uint32"), 4
	case align >= 2 && size >= 2:
		return golang.Named("uint16"), 2
	default:
		return golang.Named("uint8"), 1
	}
}

func anchorAlign(t *Type, ptr int) int {
	a := t.Align
	if a > ptr {
		a = ptr
	}
	if a < 1 {
		a = 1
	}
	if t.Size <= 0 {
		return 1
	}
	for a > 1 && t.Size%a != 0 {
		a >>= 1
	}
	return a
}

func goAlign(t *Type, ptr int) int {
	if t == nil {
		return 1
	}
	switch t.Kind {
	case kindAlias, kindArray:
		return goAlign(t.Elem, ptr)
	case kindStruct:
		if !t.natural(ptr) {
			return anchorAlign(t, ptr)
		}
		m := 1
		for _, f := range t.Fields {
			if f.Type == nil || f.Type.flex() {
				continue
			}
			if a := goAlign(f.Type, ptr); a > m {
				m = a
			}
		}
		if m > ptr {
			m = ptr
		}
		return m
	case kindUnion:
		return anchorAlign(t, ptr)
	default:
		a := t.Align
		if a < 1 {
			a = 1
		}
		if a > ptr {
			a = ptr
		}
		return a
	}
}

func (t *Type) natural(ptr int) bool {
	if t == nil {
		return true
	}
	if t.natKnown {
		return t.nat
	}
	t.natKnown = true
	switch t.Kind {
	case kindStruct:
		if t.Packed || t.Incomplete {
			return false
		}
		for _, f := range t.Fields {
			if f == nil || f.Type == nil || f.Type.flex() {
				continue
			}
			if f.IsBit || f.Packed || !f.Type.natural(ptr) {
				return false
			}
			ga := goAlign(f.Type, ptr)
			if ga < 1 || f.Offset%ga != 0 {
				return false
			}
		}
		t.nat = true
		return true
	case kindUnion:
		return false
	case kindArray, kindAlias:
		t.nat = t.Elem.natural(ptr)
		return t.nat
	default:
		t.nat = true
		return true
	}
}

func (w *writer) goType(t *Type) *golang.Type {
	if t == nil {
		return golang.Named("byte")
	}
	switch t.Kind {
	case kindAlias, kindStruct, kindUnion, kindEnum:
		if t.Go != "" {
			return golang.Named(t.Go)
		}
		if t.Kind == kindAlias {
			return w.goType(t.Elem)
		}
		return golang.Named("byte")
	case kindBool:
		return golang.Named("bool")
	case kindSInt:
		if t.Spell == "intptr_t" || t.Spell == "uintptr_t" {
			return golang.Named("uintptr")
		}
		return intCode(t.Size)
	case kindUInt:
		if t.Spell == "uintptr_t" || t.Spell == "intptr_t" {
			return golang.Named("uintptr")
		}
		switch t.Size {
		case 1:
			return golang.Named("byte")
		case 2:
			return golang.Named("uint16")
		case 4:
			return golang.Named("uint32")
		case 8:
			return golang.Named("uint64")
		default:
			return golang.Named("uint32")
		}
	case kindFloat:
		switch t.Size {
		case 4:
			return golang.Named("float32")
		case 8:
			return golang.Named("float64")
		case 12:
			return golang.Array(golang.Lit(3), golang.Named("uint32"))
		default:
			return golang.Array(golang.Lit(2), golang.Named("uint64"))
		}
	case kindPtr:
		if t.Elem == nil || t.Elem.Kind == kindVoid {
			return golang.QualType("unsafe", "Pointer")
		}
		if t.Elem.Kind == kindFunc {
			return golang.Named("uintptr")
		}
		return golang.Pointer(w.goType(t.Elem))
	case kindArray:
		n := t.Len
		if n < 0 {
			n = 0
		}
		return golang.Array(golang.Lit(n), w.goType(t.Elem))
	case kindFunc:
		return golang.Named("uintptr")
	default:
		return golang.Named("byte")
	}
}

func (w *writer) funcType(fn *Func) *golang.Type {
	params := make([]*golang.Field, 0, len(fn.Params))
	for _, param := range fn.Params {
		if param.Name == "" {
			params = append(params, golang.NewField(w.goType(param.Type)))
			continue
		}
		params = append(params, golang.NewField(w.goType(param.Type), param.Name))
	}
	var results []*golang.Field
	if fn.Ret != nil && fn.Ret.Kind != kindVoid {
		results = []*golang.Field{golang.NewField(w.goType(fn.Ret))}
	}
	return golang.FuncType(params, results)
}

func (w *writer) emitSymbols(file *golang.File) {
	name := w.unit.BindName
	fields := make([]*golang.Field, 0, len(w.unit.Funcs))
	for i := range w.unit.Funcs {
		fn := &w.unit.Funcs[i]
		tag := fn.Name
		if w.unit.BindSoft {
			tag += ",soft"
		}
		fields = append(fields, golang.NewField(w.funcType(fn), fn.Go).WithTag(`sym:"`+tag+`"`))
	}
	file.Type().Id(name).Struct(fields...)
}

func (w *writer) emitOpen(file *golang.File) {
	args := []*golang.Expr{
		golang.Binary(golang.OpOr, golang.Qual(nativePath, "Now"), golang.Qual(nativePath, "Global")),
	}
	for _, lib := range w.libs {
		args = append(args, golang.Lit(lib))
	}
	name := w.unit.BindName
	file.Func().Id("Open").Results(
		golang.NewField(golang.Named(name)),
		golang.NewField(golang.Named("error")),
	).Block(
		golang.Return(golang.QualType(nativePath, "Bind").Inst(golang.Named(name)).AsExpr().Call(args...)),
	)
}

func (w *writer) emitObject(file *golang.File, obj *Object) {
	ret := golang.Pointer(w.goType(obj.Type))
	file.Func().Id(obj.Go).Params(
		golang.NewField(golang.Named("uintptr"), "lib"),
	).Results(
		golang.NewField(ret),
		golang.NewField(golang.Named("error")),
	).Block(
		golang.Define([]*golang.Expr{golang.Id("addr"), golang.Id("err")}, golang.Qual(nativePath, "Symbol").Call(golang.Id("lib"), golang.Lit(obj.Name))),
		golang.If(golang.Binary(golang.OpNeq, golang.Id("err"), golang.Lit(nil)), golang.Return(golang.Lit(nil), golang.Id("err"))),
		golang.Return(ret.AsExpr().Call(golang.Qual("unsafe", "Pointer").Call(golang.Id("addr"))), golang.Lit(nil)),
	)
}

func (w *writer) emitMethods(file *golang.File, t *Type) {
	for _, field := range t.Fields {
		if field == nil || field.Go == "" {
			continue
		}
		if field.IsBit {
			w.emitBitMethod(file, t, field)
			continue
		}
		w.emitCopyMethod(file, t, field)
	}
}

func loadLen(t *Type, f *Field) int {
	n := f.CSize
	off := f.Offset
	if f.IsBit {
		off = f.Container
	}
	if n < 1 {
		n = 1
	}
	if off < 0 {
		off = 0
	}
	if off+n > t.Size {
		n = t.Size - off
	}
	if n < 1 {
		n = 1
	}
	if n > 8 {
		n = 8
	}
	return n
}

func (w *writer) sliceSrc(size *golang.Expr) *golang.Stmt {
	return golang.Define([]*golang.Expr{golang.Id("src")}, unsafeSlice(golang.Id("v"), size))
}

func sizeofStarV() *golang.Expr {
	return golang.Named("int").AsExpr().Call(golang.Qual("unsafe", "Sizeof").Call(golang.Unary(golang.OpMul, golang.Id("v"))))
}

func unsafeSlice(ptr, n *golang.Expr) *golang.Expr {
	return golang.Qual("unsafe", "Slice").Call(
		golang.Pointer(golang.Named("byte")).AsExpr().Call(golang.Qual("unsafe", "Pointer").Call(ptr)),
		n,
	)
}

func declareVar(name string, typ *golang.Type) *golang.Stmt {
	group := &golang.Var{}
	group.Id(name).Type(typ)
	return golang.Declare(group)
}

func (w *writer) emitBitMethod(file *golang.File, t *Type, field *Field) {
	n := loadLen(t, field)
	off := field.Container
	width := field.BitWidth
	if width < 0 {
		width = 0
	}
	signed := field.Type != nil && field.Type.Kind == kindSInt
	mask := bitMask(width)
	recv := golang.Pointer(golang.Named(t.Go))
	body := w.bitLoad(n, off)
	if field.BitOff > 0 {
		body = append(body, golang.Assign(golang.OpShr, []*golang.Expr{golang.Id("out")}, golang.Lit(field.BitOff)))
	}
	if width > 0 && width < 64 {
		body = append(body, golang.Assign(golang.OpAnd, []*golang.Expr{golang.Id("out")}, golang.Lit(mask)))
	}
	body = append(body, golang.Return(w.bitReturn(field, signed, width)))
	file.Func().Recv("v", recv).Id(field.Go).Results(golang.NewField(w.bitGo(field))).Block(body...)

	set := w.bitLoad(n, off)
	cleared, stored := bitStore(field, width, mask)
	set = append(set,
		cleared,
		golang.Assign(golang.OpOr, []*golang.Expr{golang.Id("out")}, stored),
		declareVar("buf", golang.Array(golang.Lit(8), golang.Named("byte"))),
		golang.Id("copyAt").Call(
			golang.Id("buf").Slice(nil, golang.Lit(n), nil),
			unsafeSlice(golang.Unary(golang.OpAnd, golang.Id("out")), golang.Lit(n)),
		),
		golang.Id("copyAt").Call(
			golang.Id("src").Slice(golang.Lit(off), golang.Lit(off+n), nil),
			golang.Id("buf").Slice(nil, golang.Lit(n), nil),
		),
	)
	file.Func().Recv("v", recv).Id("Set" + field.Go).Params(
		golang.NewField(w.bitGo(field), "x"),
	).Block(set...)
}

func (w *writer) bitLoad(n, off int) []golang.Code {
	return []golang.Code{
		declareVar("out", golang.Named("uint64")),
		golang.Define([]*golang.Expr{golang.Id("dst")}, unsafeSlice(golang.Unary(golang.OpAnd, golang.Id("out")), golang.Lit(n))),
		w.sliceSrc(sizeofStarV()),
		golang.Id("copyAt").Call(golang.Id("dst"), golang.Id("src").Slice(golang.Lit(off), golang.Lit(off+n), nil)),
	}
}

func bitStore(field *Field, width int, mask uint64) (*golang.Stmt, *golang.Expr) {
	if width >= 64 {
		return golang.Assign(0, []*golang.Expr{golang.Id("out")}, golang.Lit(0)),
			golang.Named("uint64").AsExpr().Call(golang.Id("x"))
	}
	stored := golang.Paren(golang.Binary(golang.OpAnd, golang.Named("uint64").AsExpr().Call(golang.Id("x")), golang.Lit(mask)))
	shift := field.BitOff
	if shift > 0 {
		stored = golang.Binary(golang.OpShl, golang.Paren(stored), golang.Lit(shift))
		return golang.Assign(golang.OpAndNot, []*golang.Expr{golang.Id("out")}, golang.Paren(golang.Binary(golang.OpShl, golang.Lit(mask), golang.Lit(shift)))), stored
	}
	return golang.Assign(golang.OpAndNot, []*golang.Expr{golang.Id("out")}, golang.Lit(mask)), stored
}

func (w *writer) bitGo(field *Field) *golang.Type {
	if field.Type == nil {
		return golang.Named("uint32")
	}
	if field.Type.Kind == kindSInt || field.Type.Kind == kindUInt || field.Type.Kind == kindBool {
		return w.goType(field.Type)
	}
	return golang.Named("uint32")
}

func (w *writer) bitReturn(field *Field, signed bool, width int) *golang.Expr {
	if !signed || width <= 0 || width >= 64 {
		return w.bitGo(field).AsExpr().Call(golang.Id("out"))
	}
	shift := 64 - width
	return w.bitGo(field).AsExpr().Call(golang.Binary(golang.OpShr,
		golang.Named("int64").AsExpr().Call(golang.Binary(golang.OpShl, golang.Id("out"), golang.Lit(shift))),
		golang.Lit(shift),
	))
}

func bitMask(width int) uint64 {
	if width <= 0 {
		return 0
	}
	if width >= 64 {
		return ^uint64(0)
	}
	return (uint64(1) << uint(width)) - 1
}

func (w *writer) emitCopyMethod(file *golang.File, t *Type, field *Field) {
	off := field.Offset
	n := field.CSize
	if off+n > t.Size {
		n = t.Size - off
	}
	if n < 0 {
		n = 0
	}
	typ := w.goType(field.Type)
	recv := golang.Pointer(golang.Named(t.Go))
	file.Func().Recv("v", recv).Id(field.Go).Results(golang.NewField(typ)).Block(
		declareVar("out", typ),
		golang.Define([]*golang.Expr{golang.Id("dst")}, unsafeSlice(golang.Unary(golang.OpAnd, golang.Id("out")), golang.Lit(n))),
		w.sliceSrc(sizeofStarV()),
		golang.Id("copyAt").Call(golang.Id("dst"), golang.Id("src").Slice(golang.Lit(off), golang.Lit(off+n), nil)),
		golang.Return(golang.Id("out")),
	)
	file.Func().Recv("v", recv).Id("Set"+field.Go).Params(
		golang.NewField(typ, "x"),
	).Block(
		w.sliceSrc(sizeofStarV()),
		golang.Id("copyAt").Call(
			golang.Id("src").Slice(golang.Lit(off), golang.Lit(off+n), nil),
			unsafeSlice(golang.Unary(golang.OpAnd, golang.Id("x")), golang.Lit(n)),
		),
	)
}
