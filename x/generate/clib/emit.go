package clib

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/lewtec/lewkit/x/takemake/golang"
)

// callSymbol is x/ffi/clib.call in the assembler's package spelling.
const callSymbol = "github\u00b7com\u2215lewtec\u2215lewkit\u2215x\u2215ffi\u2215clib\u00b7call"

const clibPath = "github.com/lewtec/lewkit/x/ffi/clib"

func emitGo(constraint, pkg, bind string, skipped []string, funcs []fn, defs []*sdef) ([]byte, error) {
	file := golang.NewFile(pkg)
	header := "//go:build " + constraint + "\n\n// " + generatedBy
	if len(skipped) > 0 {
		header += "\n// Skipped variadic: " + strings.Join(skipped, ", ")
	}
	file.Header = header
	for _, d := range defs {
		emitStruct(file, d)
	}
	fields := make([]*golang.Field, 0, len(funcs))
	for _, fn := range funcs {
		fields = append(fields, golang.NewField(funcTypeOf(fn), fn.goName))
	}
	file.Var().Comment(bind + " is the library. Assign the fields from init.\nThe handlers read them when a C caller enters.\nDo not call the library from init.").Id(bind).Type(golang.Struct(fields...))
	lines := make([]golang.Code, 0, len(funcs))
	for _, fn := range funcs {
		lines = append(lines, golang.Qual(clibPath, "Handle").Call(golang.Lit(fn.id), golang.Id(fn.call)))
	}
	file.Func().Id("init").Block(lines...)
	for _, fn := range funcs {
		emitHandler(file, bind, fn)
	}
	var buf bytes.Buffer
	if err := file.Make(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func emitHandler(file *golang.File, bind string, fn fn) {
	word := golang.Pointer(golang.Array(golang.Lit(8), golang.Named("uint64")))
	body := []golang.Code{
		golang.If(golang.Binary(golang.OpLor,
			golang.Binary(golang.OpLor,
				golang.Binary(golang.OpEql, golang.Id("args"), golang.Lit(nil)),
				golang.Binary(golang.OpEql, golang.Id("xmm"), golang.Lit(nil)),
			),
			golang.Binary(golang.OpEql, golang.Id(bind).Sel(fn.goName), golang.Lit(nil)),
		), golang.Return(golang.Lit(0), golang.Lit(0), golang.Lit(0))),
	}
	args := make([]*golang.Expr, len(fn.params))
	for i, p := range fn.params {
		if p.val.kind == vkStruct {
			name := fmt.Sprintf("a%d", i)
			body = append(body, localVar(name, typeOf(p.val.goType)))
			for w, at := range p.ats {
				slot := golang.Id("args").Index(golang.Lit(at.a))
				if at.kind == locXMM {
					slot = golang.Id("xmm").Index(golang.Lit(at.a))
				}
				body = append(body, golang.Qual(clibPath, "Store").Call(
					golang.Qual("unsafe", "Pointer").Call(golang.Unary(golang.OpAnd, golang.Id(name))),
					conv("int", golang.Qual("unsafe", "Sizeof").Call(golang.Id(name))),
					golang.Lit(w*8),
					slot,
				))
			}
			args[i] = golang.Id(name)
			continue
		}
		args[i] = argExpr(p)
	}
	call := golang.Id(bind).Sel(fn.goName).Call(args...)
	switch fn.ret.kind {
	case vkStruct:
		body = append(body,
			golang.Define([]*golang.Expr{golang.Id("v")}, call),
			golang.Return(
				loadWord(0),
				loadWord(8),
				golang.Lit(0),
			),
		)
	case vkVoid:
		body = append(body, call, golang.Return(golang.Lit(0), golang.Lit(0), golang.Lit(0)))
	case vkBool:
		body = append(body,
			golang.If(call, golang.Return(golang.Lit(1), golang.Lit(0), golang.Lit(0))),
			golang.Return(golang.Lit(0), golang.Lit(0), golang.Lit(0)),
		)
	case vkF32:
		body = append(body,
			golang.Define([]*golang.Expr{golang.Id("v")}, call),
			golang.Return(golang.Lit(0), conv("uint64", golang.Qual("math", "Float32bits").Call(golang.Id("v"))), golang.Qual(clibPath, "KindF32")),
		)
	case vkF64:
		body = append(body,
			golang.Define([]*golang.Expr{golang.Id("v")}, call),
			golang.Return(golang.Lit(0), golang.Qual("math", "Float64bits").Call(golang.Id("v")), golang.Qual(clibPath, "KindF64")),
		)
	default:
		body = append(body,
			golang.Define([]*golang.Expr{golang.Id("v")}, call),
			golang.Return(retExpr(fn.ret), golang.Lit(0), golang.Lit(0)),
		)
	}
	file.Func().Id(fn.call).Params(
		golang.NewField(word, "args"),
		golang.NewField(word, "xmm"),
	).Results(
		golang.NewField(golang.Named("uint64")),
		golang.NewField(golang.Named("uint64")),
		golang.NewField(golang.Named("uint32")),
	).Block(body...)
}

func loadWord(off int) *golang.Expr {
	return golang.Qual(clibPath, "Load").Call(
		golang.Qual("unsafe", "Pointer").Call(golang.Unary(golang.OpAnd, golang.Id("v"))),
		conv("int", golang.Qual("unsafe", "Sizeof").Call(golang.Id("v"))),
		golang.Lit(off),
	)
}

func emitStruct(file *golang.File, d *sdef) {
	fields := []*golang.Field{golang.NewField(golang.QualType("structs", "HostLayout"), "_")}
	off := 0
	for _, f := range d.fields {
		if f.offset > off {
			fields = append(fields, golang.NewField(golang.Array(golang.Lit(f.offset-off), golang.Named("byte")), "_"))
			off = f.offset
		}
		fields = append(fields, golang.NewField(typeOf(f.goType), f.name))
		off += f.size
	}
	if d.size > off {
		fields = append(fields, golang.NewField(golang.Array(golang.Lit(d.size-off), golang.Named("byte")), "_"))
	}
	file.Type().Id(d.name).Struct(fields...)
}

func funcTypeOf(fn fn) *golang.Type {
	params := make([]*golang.Field, 0, len(fn.params))
	for _, p := range fn.params {
		params = append(params, golang.NewField(typeOf(p.val.goType), p.name))
	}
	var results []*golang.Field
	if fn.ret.goType != "" {
		results = []*golang.Field{golang.NewField(typeOf(fn.ret.goType))}
	}
	return golang.FuncType(params, results)
}

func retExpr(v val) *golang.Expr {
	switch v.kind {
	case vkPtr:
		return conv("uint64", conv("uintptr", golang.Qual("unsafe", "Pointer").Call(golang.Id("v"))))
	case vkSInt:
		return conv("uint64", conv("int64", golang.Id("v")))
	default:
		return conv("uint64", golang.Id("v"))
	}
}

func argExpr(p param) *golang.Expr {
	v := p.val
	switch p.at.kind {
	case locStack:
		return stackExpr(v, p.at.a)
	case locArmF32:
		bits := golang.Id("xmm").Index(golang.Lit(p.at.a))
		if p.at.s%2 != 0 {
			bits = golang.Binary(golang.OpShr, bits, golang.Lit(32))
		}
		return golang.Qual("math", "Float32frombits").Call(conv("uint32", bits))
	case locArmF64:
		return golang.Qual("math", "Float64frombits").Call(golang.Id("xmm").Index(golang.Lit(p.at.a)))
	case locXMM:
		if v.kind == vkF32 {
			return golang.Qual("math", "Float32frombits").Call(conv("uint32", golang.Id("xmm").Index(golang.Lit(p.at.a))))
		}
		return golang.Qual("math", "Float64frombits").Call(golang.Id("xmm").Index(golang.Lit(p.at.a)))
	default:
		return gprExpr(v, p.at)
	}
}

func gprExpr(v val, at loc) *golang.Expr {
	slot := golang.Id("args").Index(golang.Lit(at.a))
	switch v.kind {
	case vkBool:
		return golang.Binary(golang.OpNeq, slot, golang.Lit(0))
	case vkPtr:
		ptr := golang.Qual(clibPath, "Ptr").Call(conv("uintptr", slot))
		if v.goType == "unsafe.Pointer" {
			return ptr
		}
		return conv(v.goType, ptr)
	case vkSInt, vkUInt:
		if at.b >= 0 {
			pair := golang.Binary(golang.OpOr,
				conv("uint64", conv("uint32", golang.Id("args").Index(golang.Lit(at.a)))),
				golang.Binary(golang.OpShl,
					conv("uint64", conv("uint32", golang.Id("args").Index(golang.Lit(at.b)))),
					golang.Lit(32),
				),
			)
			if v.kind == vkSInt {
				return conv("int64", pair)
			}
			return pair
		}
		return conv(v.goType, slot)
	default:
		return conv(v.goType, slot)
	}
}

func stackExpr(v val, off int) *golang.Expr {
	switch v.kind {
	case vkBool:
		return golang.Binary(golang.OpNeq, golang.Qual(clibPath, "U32").Call(golang.Id("args"), golang.Lit(off)), golang.Lit(0))
	case vkF32:
		return golang.Qual("math", "Float32frombits").Call(golang.Qual(clibPath, "U32").Call(golang.Id("args"), golang.Lit(off)))
	case vkF64:
		return golang.Qual("math", "Float64frombits").Call(golang.Qual(clibPath, "U64").Call(golang.Id("args"), golang.Lit(off)))
	case vkPtr:
		ptr := golang.Qual(clibPath, "Ptr").Call(conv("uintptr", golang.Qual(clibPath, "U32").Call(golang.Id("args"), golang.Lit(off))))
		if v.goType == "unsafe.Pointer" {
			return ptr
		}
		return conv(v.goType, ptr)
	case vkSInt, vkUInt:
		if v.size > 4 {
			got := golang.Qual(clibPath, "U64").Call(golang.Id("args"), golang.Lit(off))
			if v.kind == vkSInt {
				return conv("int64", got)
			}
			return got
		}
		return conv(v.goType, golang.Qual(clibPath, "U32").Call(golang.Id("args"), golang.Lit(off)))
	default:
		return conv(v.goType, golang.Qual(clibPath, "U32").Call(golang.Id("args"), golang.Lit(off)))
	}
}

func conv(spell string, x *golang.Expr) *golang.Expr {
	return typeOf(spell).AsExpr().Call(x)
}

func typeOf(spell string) *golang.Type {
	if strings.HasPrefix(spell, "*") {
		return golang.Pointer(typeOf(spell[1:]))
	}
	pkg, name, ok := strings.Cut(spell, ".")
	if ok {
		return golang.QualType(pkg, name)
	}
	return golang.Named(spell)
}

func localVar(name string, typ *golang.Type) *golang.Stmt {
	group := &golang.Var{}
	group.Id(name).Type(typ)
	return golang.Declare(group)
}

func emitAsm(t target, funcs []fn) []byte {
	var b strings.Builder
	b.WriteString("//go:build " + asmTag(t) + "\n\n")
	b.WriteString("#include \"textflag.h\"\n\n")
	b.WriteString("// " + generatedBy + "\n")
	switch t.mach {
	case machArm64:
		b.WriteString("// R9 carries the handler id into x/ffi/clib.call.\n\n")
	case machARM:
		b.WriteString("// R4 carries the handler id. The saved R4 is at 0(SP).\n\n")
	default:
		b.WriteString("// AX carries the handler id into x/ffi/clib.call.\n\n")
	}
	for _, fn := range funcs {
		fmt.Fprintf(&b, "TEXT %s(SB),NOSPLIT|NOFRAME,$0\n", fn.cName)
		switch t.mach {
		case machArm64:
			fmt.Fprintf(&b, "\tMOVD $%d, R9\n", fn.id)
		case machARM:
			b.WriteString("\tMOVW.W R4, -8(R13)\n")
			fmt.Fprintf(&b, "\tMOVW $%d, R4\n", fn.id)
		default:
			fmt.Fprintf(&b, "\tMOVL $%d, AX\n", fn.id)
		}
		b.WriteString("\tJMP ")
		b.WriteString(callSymbol)
		b.WriteString("(SB)\n\n")
	}
	return []byte(b.String())
}

func asmTag(t target) string {
	switch t.fileOS {
	case "windows":
		return "windows && clibexport"
	case "darwin":
		return "darwin && clibexport"
	default:
		return "(linux || android) && clibexport"
	}
}

func emitExport(constraint, pkg string, funcs []fn) []byte {
	var b strings.Builder
	b.WriteString("//go:build " + constraint + "\n\n")
	b.WriteString("// " + generatedBy + "\n")
	b.WriteString("// cmd/go skips this file. clibtoolexec passes it to the compiler\n")
	b.WriteString("// so the linker records the C names for the dynamic symbol table.\n\n")
	b.WriteString("package " + pkg + "\n\n")
	for _, fn := range funcs {
		fmt.Fprintf(&b, "//go:cgo_export_dynamic %s %s\n", fn.cName, fn.cName)
	}
	return []byte(b.String())
}
