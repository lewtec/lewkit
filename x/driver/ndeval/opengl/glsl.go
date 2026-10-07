package opengl

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/ndarray"
)

// glslSource renders the kernel schedule as a desktop GL 4.3 or ES 3.1 compute shader.
// gles selects the ES header. The body is the same schedule Vulkan spells.
func glslSource(k *ndarray.Kernel, gles bool) (string, error) {
	if k == nil {
		return "", ndarray.ErrOp
	}
	code, err := k.Code()
	if err != nil {
		return "", err
	}
	return glslOf(code, gles), nil
}

func glslOf(c ndarray.Code, gles bool) string {
	var b strings.Builder
	if gles {
		b.WriteString("#version 310 es\nprecision highp float;\nprecision highp int;\n")
	} else {
		b.WriteString("#version 430\n")
	}
	b.WriteString("layout(local_size_x = ")
	b.WriteString(strconv.Itoa(c.Threads))
	b.WriteString(") in;\n")
	b.WriteString("uniform uint n;\nuniform uint d0;\nuniform uint d1;\nuniform uint d2;\nuniform uint d3;\n")
	fmt.Fprintf(&b, "layout(std430, binding = 0) buffer Out { %s o[]; };\n", scalar(c.Out))
	for i, dt := range c.In {
		fmt.Fprintf(&b, "layout(std430, binding = %d) buffer In%d { %s x%d[]; };\n", i+1, i+1, scalar(dt), i+1)
	}
	b.WriteString("void main() {\n")
	// X alone is at most GL_MAX_COMPUTE_WORK_GROUP_COUNT, which is 65535
	// on Adreno. A phone frame needs more groups, so Run may use Y and Z.
	// gi stays the linear element index either way.
	b.WriteString("    uint span = gl_NumWorkGroups.x * gl_WorkGroupSize.x;\n")
	b.WriteString("    uint gi = gl_GlobalInvocationID.x + gl_GlobalInvocationID.y * span + gl_GlobalInvocationID.z * span * gl_NumWorkGroups.y;\n")
	writeBody(&b, c, func(e ndarray.Expr) string {
		var s strings.Builder
		(&spell{b: &s}).expr(e, 0)
		return s.String()
	})
	b.WriteString("}\n")
	return b.String()
}

func writeBody(b *strings.Builder, c ndarray.Code, spell func(ndarray.Expr) string) {
	for _, st := range c.Stmts {
		b.WriteString("    ")
		switch st.Op {
		case ndarray.StmtReturn:
			fmt.Fprintf(b, "if (%s) return;\n", spell(st.Cond))
		case ndarray.StmtDecl:
			typ := "bool"
			if !st.Bool {
				typ = scalar(st.DType)
			}
			fmt.Fprintf(b, "%s %s = %s;\n", typ, st.Name, spell(st.RHS))
		case ndarray.StmtSet:
			fmt.Fprintf(b, "%s = %s;\n", st.Name, spell(st.RHS))
		case ndarray.StmtIfSet:
			fmt.Fprintf(b, "if (%s) %s = %s;\n", spell(st.Cond), st.Name, spell(st.RHS))
		case ndarray.StmtAdd:
			fmt.Fprintf(b, "%s += %s;\n", st.Name, spell(st.RHS))
		case ndarray.StmtStore:
			fmt.Fprintf(b, "o[i] = %s;\n", spell(st.RHS))
		default:
			b.WriteString("UNSUPPORTED_STMT;\n")
		}
	}
}

func scalar(d ndarray.DType) string {
	switch d {
	case ndarray.I32:
		return "int"
	case ndarray.U8:
		return "uint"
	default:
		return "float"
	}
}

type spell struct{ b *strings.Builder }

func (p *spell) expr(e ndarray.Expr, limit int) {
	if prec(e.Op) < limit {
		p.b.WriteByte('(')
		p.expr(e, 0)
		p.b.WriteByte(')')
		return
	}
	switch e.Op {
	case ndarray.ExprRef:
		p.b.WriteString(e.Name)
	case ndarray.ExprInt:
		p.b.WriteString(strconv.FormatInt(e.Int, 10))
	case ndarray.ExprUint:
		p.b.WriteString(strconv.FormatUint(uint64(e.Int), 10))
		p.b.WriteByte('u')
	case ndarray.ExprTrue:
		p.b.WriteString("true")
	case ndarray.ExprConst:
		p.constant(e)
	case ndarray.ExprLoad:
		p.b.WriteString(e.Name)
		p.b.WriteByte('[')
		p.expr(e.Args[0], 0)
		p.b.WriteByte(']')
	case ndarray.ExprNeg:
		p.b.WriteString("(-")
		p.expr(e.Args[0], 0)
		p.b.WriteByte(')')
	case ndarray.ExprAdd:
		p.infix(e, "+")
	case ndarray.ExprMul:
		p.infix(e, "*")
	case ndarray.ExprDiv:
		p.infix(e, "/")
	case ndarray.ExprMod:
		p.infix(e, "%")
	case ndarray.ExprBitAnd:
		p.infix(e, "&")
	case ndarray.ExprBitOr:
		p.infix(e, "|")
	case ndarray.ExprBitXor:
		p.infix(e, "^")
	case ndarray.ExprShl:
		p.infix(e, "<<")
	case ndarray.ExprShr:
		p.infix(e, ">>")
	case ndarray.ExprRel:
		p.expr(e.Args[0], prec(e.Op)+1)
		p.b.WriteString(relTok(e.Rel))
		p.expr(e.Args[1], prec(e.Op)+1)
	case ndarray.ExprAnd:
		p.expr(e.Args[0], prec(e.Op))
		p.b.WriteString(" && ")
		p.expr(e.Args[1], prec(e.Op)+1)
	case ndarray.ExprSel:
		p.expr(e.Args[0], prec(e.Op)+1)
		p.b.WriteByte('?')
		p.expr(e.Args[1], prec(e.Op)+1)
		p.b.WriteByte(':')
		p.expr(e.Args[2], prec(e.Op))
	case ndarray.ExprCast:
		p.b.WriteString(scalar(e.DType))
		p.b.WriteByte('(')
		p.expr(e.Args[0], 0)
		p.b.WriteByte(')')
	case ndarray.ExprFn:
		p.b.WriteString(fnName(e.Fn))
		p.b.WriteByte('(')
		for i, a := range e.Args {
			if i > 0 {
				p.b.WriteString(", ")
			}
			p.expr(a, 0)
		}
		p.b.WriteByte(')')
	default:
		p.b.WriteString("UNSUPPORTED_EXPR")
	}
}

func (p *spell) infix(e ndarray.Expr, op string) {
	p.expr(e.Args[0], prec(e.Op))
	p.b.WriteString(op)
	p.expr(e.Args[1], prec(e.Op)+1)
}

func (p *spell) constant(e ndarray.Expr) {
	switch e.DType {
	case ndarray.I32:
		if int32(e.Bits) == math.MinInt32 {
			p.b.WriteString("int(0x80000000u)")
			return
		}
		p.b.WriteString(strconv.FormatInt(int64(int32(e.Bits)), 10))
	case ndarray.U8:
		p.b.WriteString(strconv.FormatUint(uint64(uint8(e.Bits)), 10))
		p.b.WriteByte('u')
	default:
		f := math.Float32frombits(e.Bits)
		if math.IsNaN(float64(f)) || math.IsInf(float64(f), 0) {
			fmt.Fprintf(p.b, "uintBitsToFloat(%du)", e.Bits)
			return
		}
		p.b.WriteString(formatFloat(e.Bits))
	}
}

func prec(op ndarray.ExprOp) int {
	switch op {
	case ndarray.ExprSel:
		return 1
	case ndarray.ExprAnd:
		return 2
	case ndarray.ExprBitOr:
		return 3
	case ndarray.ExprBitXor:
		return 4
	case ndarray.ExprBitAnd:
		return 5
	case ndarray.ExprRel:
		return 6
	case ndarray.ExprShl, ndarray.ExprShr:
		return 7
	case ndarray.ExprAdd:
		return 8
	case ndarray.ExprMul, ndarray.ExprDiv, ndarray.ExprMod:
		return 9
	default:
		return 11
	}
}

func relTok(r ndarray.Rel) string {
	switch r {
	case ndarray.RelLT:
		return "<"
	case ndarray.RelLE:
		return "<="
	case ndarray.RelGT:
		return ">"
	case ndarray.RelGE:
		return ">="
	case ndarray.RelEQ:
		return "=="
	case ndarray.RelNE:
		return "!="
	default:
		return "?"
	}
}

func fnName(f ndarray.Fn) string {
	switch f {
	case ndarray.FnExp2:
		return "exp2"
	case ndarray.FnLog2:
		return "log2"
	case ndarray.FnSin:
		return "sin"
	case ndarray.FnSqrt:
		return "sqrt"
	case ndarray.FnMax:
		return "max"
	case ndarray.FnClamp:
		return "clamp"
	default:
		return "UNSUPPORTED_FN"
	}
}

func formatFloat(bits uint32) string {
	s := strconv.FormatFloat(float64(math.Float32frombits(bits)), 'g', -1, 32)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}
