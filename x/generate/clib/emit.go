package clib

import (
	"fmt"
	"go/format"
	"strings"
)

// callSymbol is x/ffi/clib.call in the assembler's package spelling.
const callSymbol = "github\u00b7com\u2215lewtec\u2215lewkit\u2215x\u2215ffi\u2215clib\u00b7call"

func emitGo(constraint, pkg, bind string, skipped []string, funcs []fn) ([]byte, error) {
	needMath := false
	needUnsafe := false
	for _, fn := range funcs {
		if usesMath(fn.ret) {
			needMath = true
		}
		if usesUnsafe(fn.ret, true) {
			needUnsafe = true
		}
		for _, p := range fn.params {
			if usesMath(p.val) || p.at.kind == locArmF32 || p.at.kind == locArmF64 {
				needMath = true
			}
			if usesUnsafe(p.val, false) {
				needUnsafe = true
			}
		}
	}
	var b strings.Builder
	b.WriteString("//go:build " + constraint + "\n\n")
	b.WriteString("// " + generatedBy + "\n\n")
	b.WriteString("package " + pkg + "\n\n")
	b.WriteString("import (\n")
	if needMath {
		b.WriteString("\t\"math\"\n")
	}
	if needUnsafe {
		b.WriteString("\t\"unsafe\"\n")
	}
	if needMath || needUnsafe {
		b.WriteString("\n")
	}
	b.WriteString("\t\"github.com/lewtec/lewkit/x/ffi/clib\"\n")
	b.WriteString(")\n\n")
	if len(skipped) > 0 {
		b.WriteString("// Skipped variadic: ")
		b.WriteString(strings.Join(skipped, ", "))
		b.WriteString("\n\n")
	}
	b.WriteString("// ")
	b.WriteString(bind)
	b.WriteString(" is the library. Assign the fields from init.\n")
	b.WriteString("// The handlers read them when a C caller enters.\n")
	b.WriteString("// Do not call the library from init.\n")
	b.WriteString("var ")
	b.WriteString(bind)
	b.WriteString(" struct {\n")
	for _, fn := range funcs {
		b.WriteString("\t")
		b.WriteString(fn.goName)
		b.WriteString(" func(")
		for i, p := range fn.params {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(p.name)
			b.WriteString(" ")
			b.WriteString(p.val.goType)
		}
		b.WriteString(")")
		if fn.ret.goType != "" {
			b.WriteString(" ")
			b.WriteString(fn.ret.goType)
		}
		b.WriteString("\n")
	}
	b.WriteString("}\n\n")
	b.WriteString("func init() {\n")
	for _, fn := range funcs {
		fmt.Fprintf(&b, "\tclib.Handle(%d, %s)\n", fn.id, fn.call)
	}
	b.WriteString("}\n\n")
	for _, fn := range funcs {
		b.WriteString(emitHandler(bind, fn))
		b.WriteString("\n")
	}
	return format.Source([]byte(b.String()))
}

func emitHandler(bind string, fn fn) string {
	var b strings.Builder
	fmt.Fprintf(&b, "func %s(args *[8]uint64, xmm *[8]uint64) (uint64, uint64, uint32) {\n", fn.call)
	b.WriteString("\tif args == nil || xmm == nil || ")
	b.WriteString(bind)
	b.WriteString(".")
	b.WriteString(fn.goName)
	b.WriteString(" == nil {\n\t\treturn 0, 0, 0\n\t}\n\t")
	args := make([]string, len(fn.params))
	for i, p := range fn.params {
		args[i] = argExpr(p)
	}
	call := bind + "." + fn.goName + "(" + strings.Join(args, ", ") + ")"
	switch fn.ret.kind {
	case vkVoid:
		b.WriteString(call)
		b.WriteString("\n\treturn 0, 0, 0\n")
	case vkBool:
		b.WriteString("if ")
		b.WriteString(call)
		b.WriteString(" {\n\t\treturn 1, 0, 0\n\t}\n\treturn 0, 0, 0\n")
	case vkF32:
		b.WriteString("v := ")
		b.WriteString(call)
		b.WriteString("\n\treturn 0, uint64(math.Float32bits(v)), clib.KindF32\n")
	case vkF64:
		b.WriteString("v := ")
		b.WriteString(call)
		b.WriteString("\n\treturn 0, math.Float64bits(v), clib.KindF64\n")
	default:
		b.WriteString("v := ")
		b.WriteString(call)
		b.WriteString("\n\treturn ")
		b.WriteString(retExpr(fn.ret))
		b.WriteString(", 0, 0\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func retExpr(v val) string {
	switch v.kind {
	case vkPtr:
		return "uint64(uintptr(unsafe.Pointer(v)))"
	case vkSInt:
		return "uint64(int64(v))"
	default:
		return "uint64(v)"
	}
}

func argExpr(p param) string {
	v := p.val
	switch p.at.kind {
	case locStack:
		return stackExpr(v, p.at.a)
	case locArmF32:
		if p.at.s%2 == 0 {
			return fmt.Sprintf("math.Float32frombits(uint32(xmm[%d]))", p.at.a)
		}
		return fmt.Sprintf("math.Float32frombits(uint32(xmm[%d]>>32))", p.at.a)
	case locArmF64:
		return fmt.Sprintf("math.Float64frombits(xmm[%d])", p.at.a)
	case locXMM:
		if v.kind == vkF32 {
			return fmt.Sprintf("math.Float32frombits(uint32(xmm[%d]))", p.at.a)
		}
		return fmt.Sprintf("math.Float64frombits(xmm[%d])", p.at.a)
	default:
		return gprExpr(v, p.at)
	}
}

func gprExpr(v val, at loc) string {
	switch v.kind {
	case vkBool:
		return fmt.Sprintf("args[%d] != 0", at.a)
	case vkPtr:
		if v.goType == "unsafe.Pointer" {
			return fmt.Sprintf("clib.Ptr(uintptr(args[%d]))", at.a)
		}
		return fmt.Sprintf("(%s)(clib.Ptr(uintptr(args[%d])))", v.goType, at.a)
	case vkSInt, vkUInt:
		if at.b >= 0 {
			pair := fmt.Sprintf("uint64(uint32(args[%d]))|uint64(uint32(args[%d]))<<32", at.a, at.b)
			if v.kind == vkSInt {
				return "int64(" + pair + ")"
			}
			return pair
		}
		return fmt.Sprintf("%s(args[%d])", v.goType, at.a)
	default:
		return fmt.Sprintf("%s(args[%d])", v.goType, at.a)
	}
}

func stackExpr(v val, off int) string {
	switch v.kind {
	case vkBool:
		return fmt.Sprintf("clib.U32(args, %d) != 0", off)
	case vkF32:
		return fmt.Sprintf("math.Float32frombits(clib.U32(args, %d))", off)
	case vkF64:
		return fmt.Sprintf("math.Float64frombits(clib.U64(args, %d))", off)
	case vkPtr:
		if v.goType == "unsafe.Pointer" {
			return fmt.Sprintf("clib.Ptr(uintptr(clib.U32(args, %d)))", off)
		}
		return fmt.Sprintf("(%s)(clib.Ptr(uintptr(clib.U32(args, %d))))", v.goType, off)
	case vkSInt, vkUInt:
		if v.size > 4 {
			if v.kind == vkSInt {
				return fmt.Sprintf("int64(clib.U64(args, %d))", off)
			}
			return fmt.Sprintf("clib.U64(args, %d)", off)
		}
		return fmt.Sprintf("%s(clib.U32(args, %d))", v.goType, off)
	default:
		return fmt.Sprintf("%s(clib.U32(args, %d))", v.goType, off)
	}
}

func usesMath(v val) bool {
	return v.kind == vkF32 || v.kind == vkF64
}

func usesUnsafe(v val, result bool) bool {
	if result && v.kind == vkPtr {
		return true
	}
	return strings.Contains(v.goType, "unsafe.")
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
