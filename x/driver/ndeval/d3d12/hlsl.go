package d3d12

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/ndarray"
)

// hlslSource renders the kernel schedule as one HLSL compute kernel.
// n and d0..d3 are the cbuffer members, so the body does not redeclare them.
func hlslSource(k *ndarray.Kernel) (string, int, error) {
	if k == nil {
		return "", 0, ndarray.ErrOp
	}
	code, err := k.Code()
	if err != nil {
		return "", 0, err
	}
	return hlslOf(code), code.Threads, nil
}

func hlslOf(c ndarray.Code) string {
	var b strings.Builder
	fmt.Fprintf(&b, "RWStructuredBuffer<%s> o : register(u0);\n", scalar(c.Out))
	for i, dt := range c.In {
		fmt.Fprintf(&b, "StructuredBuffer<%s> x%d : register(t%d);\n", scalar(dt), i+1, i)
	}
	b.WriteString("cbuffer Push : register(b0) {\n")
	b.WriteString("    uint n;\n    uint d0;\n    uint d1;\n    uint d2;\n    uint d3;\n};\n")
	fmt.Fprintf(&b, "[numthreads(%d, 1, 1)]\n", c.Threads)
	b.WriteString("void ndeval(uint3 tid : SV_DispatchThreadID) {\n")
	b.WriteString("    uint gid = tid.x;\n")
	b.WriteString("    uint gi = gid;\n")
	writeBody(&b, c, func(e ndarray.Expr) string {
		var s strings.Builder
		(&hlsl{b: &s}).expr(e, 0)
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

type hlsl struct{ b *strings.Builder }

func (p *hlsl) expr(e ndarray.Expr, limit int) {
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
		// cs_5_0 rejects int(bool). A predicate cast is a 0/1 select.
		if e.DType == ndarray.I32 && len(e.Args) == 1 && e.Args[0].Op == ndarray.ExprRel {
			a := e.Args[0]
			p.b.WriteString("((")
			p.expr(a.Args[0], 0)
			p.b.WriteString(relTok(a.Rel))
			p.expr(a.Args[1], 0)
			p.b.WriteString(")?1:0)")
			return
		}
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

func (p *hlsl) infix(e ndarray.Expr, op string) {
	p.expr(e.Args[0], prec(e.Op))
	p.b.WriteString(op)
	p.expr(e.Args[1], prec(e.Op)+1)
}

func (p *hlsl) constant(e ndarray.Expr) {
	switch e.DType {
	case ndarray.I32:
		if int32(e.Bits) == math.MinInt32 {
			p.b.WriteString("asint(0x80000000u)")
			return
		}
		p.b.WriteString(strconv.FormatInt(int64(int32(e.Bits)), 10))
	case ndarray.U8:
		p.b.WriteString(strconv.FormatUint(uint64(uint8(e.Bits)), 10))
		p.b.WriteByte('u')
	default:
		f := math.Float32frombits(e.Bits)
		if math.IsNaN(float64(f)) || math.IsInf(float64(f), 0) {
			fmt.Fprintf(p.b, "asfloat(%du)", e.Bits)
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
