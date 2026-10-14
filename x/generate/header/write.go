package header

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/lewtec/lewkit/x/generate"
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
	file := jen.NewFile(pkg)
	file.HeaderComment(w.banner())
	w.emit(file)
	var buf bytes.Buffer
	if err := file.Render(&buf); err != nil {
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

func (w *writer) emit(f *jen.File) {
	for _, t := range w.unit.Types {
		if t.Kind == kindEnum {
			w.emitEnum(f, t)
		}
	}
	for _, c := range w.unit.Consts {
		f.Const().Id(c.Go).Op("=").Add(litInt(c.Value))
	}
	for _, t := range w.unit.Types {
		switch t.Kind {
		case kindStruct, kindUnion:
			w.emitAggregate(f, t)
		case kindAlias:
			f.Type().Id(t.Go).Op("=").Add(w.goType(t.Elem))
		}
	}
	if len(w.unit.Funcs) > 0 {
		w.emitSymbols(f)
	}
	if len(w.libs) > 0 && len(w.unit.Funcs) > 0 {
		w.emitOpen(f)
	}
	for i := range w.unit.Objects {
		w.emitObject(f, &w.unit.Objects[i])
	}
	for _, t := range w.unit.Types {
		if t.Kind != kindStruct && t.Kind != kindUnion {
			continue
		}
		if t.Incomplete || t.natural(w.ptr) {
			continue
		}
		w.raw = true
		w.emitMethods(f, t)
	}
	if w.raw {
		f.Func().Id("copyAt").Params(jen.Id("dst").Index().Byte(), jen.Id("src").Index().Byte()).Block(
			jen.Id("copy").Call(jen.Id("dst"), jen.Id("src")),
		)
	}
}

func (w *writer) emitEnum(f *jen.File, t *Type) {
	f.Type().Id(t.Go).Add(intCode(t.Size))
	for _, e := range t.Enums {
		f.Const().Id(e.Go).Id(t.Go).Op("=").Add(litInt(e.Value))
	}
}

func litInt(v int64) jen.Code {
	return jen.Op(strconv.FormatInt(v, 10))
}

func intCode(size int) *jen.Statement {
	switch size {
	case 1:
		return jen.Int8()
	case 2:
		return jen.Int16()
	case 8:
		return jen.Int64()
	default:
		return jen.Int32()
	}
}

func (w *writer) emitAggregate(f *jen.File, t *Type) {
	if t.Incomplete {
		f.Type().Id(t.Go).Struct()
		return
	}
	var fields []jen.Code
	if t.natural(w.ptr) {
		fields = w.naturalFields(t)
	} else {
		fields = rawFields(t, w.ptr)
	}
	f.Type().Id(t.Go).Struct(fields...)
}

func (w *writer) naturalFields(t *Type) []jen.Code {
	fields := []jen.Code{jen.Id("_").Qual("structs", "HostLayout")}
	off := 0
	for _, f := range t.Fields {
		if f == nil || f.Go == "" {
			continue
		}
		if f.Offset > off {
			fields = append(fields, jen.Id("_").Index(jen.Lit(f.Offset-off)).Byte())
			off = f.Offset
		}
		fields = append(fields, jen.Id(f.Go).Add(w.goType(f.Type)))
		off += f.CSize
	}
	if t.Size > off {
		fields = append(fields, jen.Id("_").Index(jen.Lit(t.Size-off)).Byte())
	}
	return fields
}

func rawFields(t *Type, ptr int) []jen.Code {
	fields := []jen.Code{jen.Id("_").Qual("structs", "HostLayout")}
	if t.Size <= 0 {
		return fields
	}
	word, n := anchorWord(anchorAlign(t, ptr), t.Size)
	fields = append(fields, jen.Id("_").Add(word))
	if tail := t.Size - n; tail > 0 {
		fields = append(fields, jen.Id("_").Index(jen.Lit(tail)).Byte())
	}
	return fields
}

func anchorWord(align, size int) (jen.Code, int) {
	switch {
	case align >= 8 && size >= 8:
		return jen.Uint64(), 8
	case align >= 4 && size >= 4:
		return jen.Uint32(), 4
	case align >= 2 && size >= 2:
		return jen.Uint16(), 2
	default:
		return jen.Uint8(), 1
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

func (w *writer) goType(t *Type) *jen.Statement {
	if t == nil {
		return jen.Byte()
	}
	switch t.Kind {
	case kindAlias, kindStruct, kindUnion, kindEnum:
		if t.Go != "" {
			return jen.Id(t.Go)
		}
		if t.Kind == kindAlias {
			return w.goType(t.Elem)
		}
		return jen.Byte()
	case kindBool:
		return jen.Bool()
	case kindSInt:
		if t.Spell == "intptr_t" || t.Spell == "uintptr_t" {
			return jen.Uintptr()
		}
		return intCode(t.Size)
	case kindUInt:
		if t.Spell == "uintptr_t" || t.Spell == "intptr_t" {
			return jen.Uintptr()
		}
		switch t.Size {
		case 1:
			return jen.Byte()
		case 2:
			return jen.Uint16()
		case 4:
			return jen.Uint32()
		case 8:
			return jen.Uint64()
		default:
			return jen.Uint32()
		}
	case kindFloat:
		switch t.Size {
		case 4:
			return jen.Float32()
		case 8:
			return jen.Float64()
		case 12:
			return jen.Index(jen.Lit(3)).Uint32()
		default:
			return jen.Index(jen.Lit(2)).Uint64()
		}
	case kindPtr:
		if t.Elem == nil || t.Elem.Kind == kindVoid {
			return jen.Qual("unsafe", "Pointer")
		}
		if t.Elem.Kind == kindFunc {
			return jen.Uintptr()
		}
		return jen.Op("*").Add(w.goType(t.Elem))
	case kindArray:
		n := t.Len
		if n < 0 {
			n = 0
		}
		return jen.Index(jen.Lit(n)).Add(w.goType(t.Elem))
	case kindFunc:
		return jen.Uintptr()
	default:
		return jen.Byte()
	}
}

func (w *writer) funcType(fn *Func) *jen.Statement {
	params := make([]jen.Code, 0, len(fn.Params))
	for _, param := range fn.Params {
		if param.Name == "" {
			params = append(params, w.goType(param.Type))
			continue
		}
		params = append(params, jen.Id(param.Name).Add(w.goType(param.Type)))
	}
	sig := jen.Func().Params(params...)
	if fn.Ret != nil && fn.Ret.Kind != kindVoid {
		sig = sig.Add(w.goType(fn.Ret))
	}
	return sig
}

func (w *writer) emitSymbols(f *jen.File) {
	name := w.unit.BindName
	f.Type().Id(name).StructFunc(func(g *jen.Group) {
		for i := range w.unit.Funcs {
			fn := &w.unit.Funcs[i]
			tag := fn.Name
			if w.unit.BindSoft {
				tag += ",soft"
			}
			g.Id(fn.Go).Add(w.funcType(fn)).Tag(map[string]string{"sym": tag})
		}
	})
}

func (w *writer) emitOpen(f *jen.File) {
	args := []jen.Code{jen.Qual(nativePath, "Now").Op("|").Qual(nativePath, "Global")}
	for _, lib := range w.libs {
		args = append(args, jen.Lit(lib))
	}
	name := w.unit.BindName
	f.Func().Id("Open").Params().Params(jen.Id(name), jen.Error()).Block(
		jen.Return(jen.Qual(nativePath, "Bind").Types(jen.Id(name)).Call(args...)),
	)
}

func (w *writer) emitObject(f *jen.File, obj *Object) {
	ret := jen.Op("*").Add(w.goType(obj.Type))
	f.Func().Id(obj.Go).Params(jen.Id("lib").Uintptr()).Params(ret, jen.Error()).Block(
		jen.List(jen.Id("addr"), jen.Err()).Op(":=").Qual(nativePath, "Symbol").Call(jen.Id("lib"), jen.Lit(obj.Name)),
		jen.If(jen.Err().Op("!=").Nil()).Block(jen.Return(jen.Nil(), jen.Err())),
		jen.Return(jen.Parens(ret).Call(jen.Qual("unsafe", "Pointer").Call(jen.Id("addr"))), jen.Nil()),
	)
}

func (w *writer) emitMethods(f *jen.File, t *Type) {
	for _, field := range t.Fields {
		if field == nil || field.Go == "" {
			continue
		}
		if field.IsBit {
			w.emitBitMethod(f, t, field)
			continue
		}
		w.emitCopyMethod(f, t, field)
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

func (w *writer) sliceSrc(sizeExpr jen.Code) *jen.Statement {
	return jen.Id("src").Op(":=").Qual("unsafe", "Slice").Call(
		jen.Parens(jen.Op("*").Byte()).Call(jen.Qual("unsafe", "Pointer").Call(jen.Id("v"))),
		sizeExpr,
	)
}

func (w *writer) emitBitMethod(f *jen.File, t *Type, field *Field) {
	n := loadLen(t, field)
	off := field.Container
	width := field.BitWidth
	if width < 0 {
		width = 0
	}
	signed := field.Type != nil && field.Type.Kind == kindSInt
	mask := bitMask(width)
	f.Func().Params(jen.Id("v").Op("*").Id(t.Go)).Id(field.Go).Params().Add(w.bitGo(field)).BlockFunc(func(g *jen.Group) {
		g.Var().Id("out").Uint64()
		g.Id("dst").Op(":=").Qual("unsafe", "Slice").Call(
			jen.Parens(jen.Op("*").Byte()).Call(jen.Qual("unsafe", "Pointer").Call(jen.Op("&").Id("out"))),
			jen.Lit(n),
		)
		g.Add(w.sliceSrc(jen.Id("int").Call(jen.Qual("unsafe", "Sizeof").Call(jen.Op("*").Id("v")))))
		g.Id("copyAt").Call(jen.Id("dst"), jen.Id("src").Index(jen.Lit(off), jen.Lit(off+n)))
		if field.BitOff > 0 {
			g.Id("out").Op(">>=").Lit(field.BitOff)
		}
		if width > 0 && width < 64 {
			g.Id("out").Op("&=").Lit(mask)
		}
		g.Return(w.bitReturn(field, signed, width))
	})
	f.Func().Params(jen.Id("v").Op("*").Id(t.Go)).Id("Set" + field.Go).Params(jen.Id("x").Add(w.bitGo(field))).BlockFunc(func(g *jen.Group) {
		g.Var().Id("out").Uint64()
		g.Id("dst").Op(":=").Qual("unsafe", "Slice").Call(
			jen.Parens(jen.Op("*").Byte()).Call(jen.Qual("unsafe", "Pointer").Call(jen.Op("&").Id("out"))),
			jen.Lit(n),
		)
		g.Add(w.sliceSrc(jen.Id("int").Call(jen.Qual("unsafe", "Sizeof").Call(jen.Op("*").Id("v")))))
		g.Id("copyAt").Call(jen.Id("dst"), jen.Id("src").Index(jen.Lit(off), jen.Lit(off+n)))
		shift := field.BitOff
		cleared := jen.Id("out").Op("&^=").Lit(mask)
		stored := jen.Parens(jen.Uint64().Call(jen.Id("x")).Op("&").Lit(mask))
		if shift > 0 {
			cleared = jen.Id("out").Op("&^=").Parens(jen.Lit(mask).Op("<<").Lit(shift))
			stored = jen.Parens(stored).Op("<<").Lit(shift)
		}
		if width >= 64 {
			cleared = jen.Id("out").Op("=").Lit(0)
			stored = jen.Uint64().Call(jen.Id("x"))
		}
		g.Add(cleared)
		g.Id("out").Op("|=").Add(stored)
		g.Var().Id("buf").Index(jen.Lit(8)).Byte()
		g.Id("copyAt").Call(
			jen.Id("buf").Index(jen.Empty(), jen.Lit(n)),
			jen.Qual("unsafe", "Slice").Call(
				jen.Parens(jen.Op("*").Byte()).Call(jen.Qual("unsafe", "Pointer").Call(jen.Op("&").Id("out"))),
				jen.Lit(n),
			),
		)
		g.Id("copyAt").Call(jen.Id("src").Index(jen.Lit(off), jen.Lit(off+n)), jen.Id("buf").Index(jen.Empty(), jen.Lit(n)))
	})
}

func (w *writer) bitGo(field *Field) *jen.Statement {
	if field.Type == nil {
		return jen.Uint32()
	}
	if field.Type.Kind == kindSInt || field.Type.Kind == kindUInt || field.Type.Kind == kindBool {
		return w.goType(field.Type)
	}
	return jen.Uint32()
}

func (w *writer) bitReturn(field *Field, signed bool, width int) jen.Code {
	if !signed || width <= 0 || width >= 64 {
		return w.bitGo(field).Call(jen.Id("out"))
	}
	shift := 64 - width
	return w.bitGo(field).Call(jen.Int64().Call(jen.Id("out").Op("<<").Lit(shift)).Op(">>").Lit(shift))
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

func (w *writer) emitCopyMethod(f *jen.File, t *Type, field *Field) {
	off := field.Offset
	n := field.CSize
	if off+n > t.Size {
		n = t.Size - off
	}
	if n < 0 {
		n = 0
	}
	f.Func().Params(jen.Id("v").Op("*").Id(t.Go)).Id(field.Go).Params().Add(w.goType(field.Type)).Block(
		jen.Var().Id("out").Add(w.goType(field.Type)),
		jen.Id("dst").Op(":=").Qual("unsafe", "Slice").Call(
			jen.Parens(jen.Op("*").Byte()).Call(jen.Qual("unsafe", "Pointer").Call(jen.Op("&").Id("out"))),
			jen.Lit(n),
		),
		w.sliceSrc(jen.Id("int").Call(jen.Qual("unsafe", "Sizeof").Call(jen.Op("*").Id("v")))),
		jen.Id("copyAt").Call(jen.Id("dst"), jen.Id("src").Index(jen.Lit(off), jen.Lit(off+n))),
		jen.Return(jen.Id("out")),
	)
	f.Func().Params(jen.Id("v").Op("*").Id(t.Go)).Id("Set"+field.Go).Params(jen.Id("x").Add(w.goType(field.Type))).Block(
		w.sliceSrc(jen.Id("int").Call(jen.Qual("unsafe", "Sizeof").Call(jen.Op("*").Id("v")))),
		jen.Id("copyAt").Call(
			jen.Id("src").Index(jen.Lit(off), jen.Lit(off+n)),
			jen.Qual("unsafe", "Slice").Call(
				jen.Parens(jen.Op("*").Byte()).Call(jen.Qual("unsafe", "Pointer").Call(jen.Op("&").Id("x"))),
				jen.Lit(n),
			),
		),
	)
}
