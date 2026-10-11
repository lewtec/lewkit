package clib

import (
	"fmt"
	"go/format"
	"strings"
)

// callSymbol is x/ffi/clib.call in the assembler's package spelling.
const callSymbol = "github\u00b7com\u2215lewtec\u2215lewkit\u2215x\u2215ffi\u2215clib\u00b7call"

func emitGo(pkg, bind string, skipped []string, funcs []fn) ([]byte, error) {
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
			if usesMath(p.val) {
				needMath = true
			}
			if usesUnsafe(p.val, false) {
				needUnsafe = true
			}
		}
	}
	var b strings.Builder
	b.WriteString("//go:build linux && amd64\n\n")
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
	fmt.Fprintf(&b, "func %s(args *[6]uintptr, xmm *[8]uint64) (uintptr, uint64) {\n", fn.call)
	b.WriteString("\tif args == nil || xmm == nil || ")
	b.WriteString(bind)
	b.WriteString(".")
	b.WriteString(fn.goName)
	b.WriteString(" == nil {\n\t\treturn 0, 0\n\t}\n\t")
	gpr, xmm := 0, 0
	args := make([]string, len(fn.params))
	for i, p := range fn.params {
		args[i] = argExpr(p.val, gpr, xmm)
		if p.val.reg == regXMM {
			xmm++
		} else {
			gpr++
		}
	}
	call := bind + "." + fn.goName + "(" + strings.Join(args, ", ") + ")"
	switch fn.ret.kind {
	case vkVoid:
		b.WriteString(call)
		b.WriteString("\n\treturn 0, 0\n")
	case vkBool:
		b.WriteString("if ")
		b.WriteString(call)
		b.WriteString(" {\n\t\treturn 1, 0\n\t}\n\treturn 0, 0\n")
	case vkF32:
		b.WriteString("v := ")
		b.WriteString(call)
		b.WriteString("\n\treturn 0, uint64(math.Float32bits(v))\n")
	case vkF64:
		b.WriteString("v := ")
		b.WriteString(call)
		b.WriteString("\n\treturn 0, math.Float64bits(v)\n")
	default:
		b.WriteString("v := ")
		b.WriteString(call)
		if fn.ret.kind == vkPtr {
			b.WriteString("\n\treturn uintptr(unsafe.Pointer(v)), 0\n")
		} else {
			b.WriteString("\n\treturn uintptr(v), 0\n")
		}
	}
	b.WriteString("}\n")
	return b.String()
}

func argExpr(v val, gpr, xmm int) string {
	switch v.kind {
	case vkBool:
		return fmt.Sprintf("args[%d] != 0", gpr)
	case vkF32:
		return fmt.Sprintf("math.Float32frombits(uint32(xmm[%d]))", xmm)
	case vkF64:
		return fmt.Sprintf("math.Float64frombits(xmm[%d])", xmm)
	case vkPtr:
		if v.goType == "unsafe.Pointer" {
			return fmt.Sprintf("clib.Ptr(args[%d])", gpr)
		}
		return fmt.Sprintf("(%s)(clib.Ptr(args[%d]))", v.goType, gpr)
	default:
		return fmt.Sprintf("%s(args[%d])", v.goType, gpr)
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

func emitAsm(funcs []fn) []byte {
	var b strings.Builder
	b.WriteString("#include \"textflag.h\"\n\n")
	b.WriteString("// " + generatedBy + "\n")
	b.WriteString("// Each TEXT is the C symbol. AX carries the handler id.\n\n")
	for _, fn := range funcs {
		fmt.Fprintf(&b, "TEXT %s(SB),NOSPLIT|NOFRAME,$0\n", fn.cName)
		fmt.Fprintf(&b, "\tMOVL $%d, AX\n", fn.id)
		b.WriteString("\tJMP ")
		b.WriteString(callSymbol)
		b.WriteString("(SB)\n\n")
	}
	return []byte(b.String())
}

func emitExport(pkg string, funcs []fn) []byte {
	var b strings.Builder
	b.WriteString("//go:build linux && amd64\n\n")
	b.WriteString("// " + generatedBy + "\n")
	b.WriteString("// cmd/go skips this file. clibtoolexec passes it to the compiler\n")
	b.WriteString("// so the linker writes the C names into .dynsym.\n\n")
	b.WriteString("package " + pkg + "\n\n")
	for _, fn := range funcs {
		fmt.Fprintf(&b, "//go:cgo_export_dynamic %s %s\n", fn.cName, fn.cName)
	}
	return []byte(b.String())
}
