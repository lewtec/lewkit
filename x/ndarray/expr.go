package ndarray

import (
	"fmt"
	"math"
	"slices"
)

type kind uint8

const (
	kindConst kind = iota
	kindIn
	kindOp
)

// Node is one vertex of a fused kernel: a const, an input view, or an ALU op.
type Node struct {
	kind kind
	op   Op
	dt   DType
	srcs []*Node
	st   Tracker
	slot int
	bits uint32
	err  error
}

func bad(err error) *Node { return &Node{err: err} }

// In is a float32 input buffer at slot, addressed by st.
func In(slot int, st Tracker) *Node {
	return InT(slot, st, F32)
}

// InT is an input buffer at slot with dtype dt.
func InT(slot int, st Tracker, dt DType) *Node {
	if slot < 0 || st.check() != nil || (dt != F32 && dt != I32) {
		return bad(ErrOp)
	}
	return &Node{kind: kindIn, dt: dt, st: st, slot: slot}
}

// Const is a float32 splat.
func Const(v float32) *Node {
	return &Node{kind: kindConst, dt: F32, bits: math.Float32bits(v)}
}

// ConstI is an int32 splat.
func ConstI(v int32) *Node {
	return &Node{kind: kindConst, dt: I32, bits: uint32(v)}
}

// Shape is the logical shape, or nil for a splat.
func (n *Node) Shape() []int {
	if n == nil || n.err != nil {
		return nil
	}
	if n.kind == kindIn {
		return n.st.Shape()
	}
	if n.kind == kindOp {
		for _, s := range n.srcs {
			if sh := s.Shape(); sh != nil {
				return sh
			}
		}
	}
	return nil
}

// DType is the result type.
func (n *Node) DType() DType {
	if n == nil {
		return 0
	}
	return n.dt
}

func (n *Node) Add(b *Node) *Node       { return Add(n, b) }
func (n *Node) Mul(b *Node) *Node       { return Mul(n, b) }
func (n *Node) IDiv(b *Node) *Node      { return IDiv(n, b) }
func (n *Node) Max(b *Node) *Node       { return Max(n, b) }
func (n *Node) Mod(b *Node) *Node       { return Mod(n, b) }
func (n *Node) CmpLt(b *Node) *Node     { return CmpLt(n, b) }
func (n *Node) CmpNe(b *Node) *Node     { return CmpNe(n, b) }
func (n *Node) Xor(b *Node) *Node       { return Xor(n, b) }
func (n *Node) Shl(b *Node) *Node       { return Shl(n, b) }
func (n *Node) Shr(b *Node) *Node       { return Shr(n, b) }
func (n *Node) Or(b *Node) *Node        { return Or(n, b) }
func (n *Node) And(b *Node) *Node       { return And(n, b) }
func (n *Node) Exp2() *Node             { return Exp2(n) }
func (n *Node) Log2() *Node             { return Log2(n) }
func (n *Node) Sin() *Node              { return Sin(n) }
func (n *Node) Sqrt() *Node             { return Sqrt(n) }
func (n *Node) Recip() *Node            { return Recip(n) }
func (n *Node) Neg() *Node              { return Neg(n) }
func (n *Node) Cast(dt DType) *Node     { return Cast(n, dt) }
func (n *Node) Where(a, b *Node) *Node  { return Where(n, a, b) }
func (n *Node) MulAcc(b, c *Node) *Node { return MulAcc(n, b, c) }

// Add is a + b.
func Add(a, b *Node) *Node { return bin(ADD, a, b) }

// Mul is a * b.
func Mul(a, b *Node) *Node { return bin(MUL, a, b) }

// IDiv is integer a / b.
func IDiv(a, b *Node) *Node { return bin(IDIV, a, b) }

// Max is max(a, b).
func Max(a, b *Node) *Node { return bin(MAX, a, b) }

// Mod is integer a % b.
func Mod(a, b *Node) *Node { return bin(MOD, a, b) }

// CmpLt is 1 if a < b, else 0.
func CmpLt(a, b *Node) *Node { return bin(CMPLT, a, b) }

// CmpNe is 1 if a != b, else 0.
func CmpNe(a, b *Node) *Node { return bin(CMPNE, a, b) }

// Xor is a ^ b.
func Xor(a, b *Node) *Node { return bin(XOR, a, b) }

// Shl is a << b.
func Shl(a, b *Node) *Node { return bin(SHL, a, b) }

// Shr is a >> b.
func Shr(a, b *Node) *Node { return bin(SHR, a, b) }

// Or is a | b.
func Or(a, b *Node) *Node { return bin(OR, a, b) }

// And is a & b.
func And(a, b *Node) *Node { return bin(AND, a, b) }

// Exp2 is 2^a.
func Exp2(a *Node) *Node { return unary(EXP2, a) }

// Log2 is log2(a).
func Log2(a *Node) *Node { return unary(LOG2, a) }

// Sin is sin(a).
func Sin(a *Node) *Node { return unary(SIN, a) }

// Sqrt is sqrt(a).
func Sqrt(a *Node) *Node { return unary(SQRT, a) }

// Recip is 1/a.
func Recip(a *Node) *Node { return unary(RECIP, a) }

// Neg is -a.
func Neg(a *Node) *Node { return unary(NEG, a) }

// Cast converts a to dt.
func Cast(a *Node, dt DType) *Node {
	n := unary(CAST, a)
	if n.err != nil {
		return n
	}
	if dt != F32 && dt != I32 {
		return bad(ErrType)
	}
	n.dt = dt
	return n
}

// Where is a if p != 0 else b. Call as Where(p, a, b) or p.Where(a, b).
func Where(p, a, b *Node) *Node {
	if p == nil || a == nil || b == nil {
		return bad(ErrOp)
	}
	if p.err != nil {
		return p
	}
	if a.err != nil {
		return a
	}
	if b.err != nil {
		return b
	}
	if p.dt != I32 || a.dt != b.dt {
		return bad(ErrType)
	}
	if err := sameShape(p, a, b); err != nil {
		return bad(err)
	}
	return &Node{kind: kindOp, op: WHERE, dt: a.dt, srcs: []*Node{p, a, b}}
}

// MulAcc is a*b + c.
func MulAcc(a, b, c *Node) *Node {
	if a == nil || b == nil || c == nil {
		return bad(ErrOp)
	}
	if a.err != nil {
		return a
	}
	if b.err != nil {
		return b
	}
	if c.err != nil {
		return c
	}
	if a.dt != b.dt || a.dt != c.dt {
		return bad(ErrType)
	}
	if err := sameShape(a, b, c); err != nil {
		return bad(err)
	}
	return &Node{kind: kindOp, op: MULACC, dt: a.dt, srcs: []*Node{a, b, c}}
}

func unary(op Op, a *Node) *Node {
	if a == nil {
		return bad(ErrOp)
	}
	if a.err != nil {
		return a
	}
	dt, err := unaryDT(op, a.dt)
	if err != nil {
		return bad(err)
	}
	return &Node{kind: kindOp, op: op, dt: dt, srcs: []*Node{a}}
}

func bin(op Op, a, b *Node) *Node {
	if a == nil || b == nil {
		return bad(ErrOp)
	}
	if a.err != nil {
		return a
	}
	if b.err != nil {
		return b
	}
	dt, err := binaryDT(op, a.dt, b.dt)
	if err != nil {
		return bad(err)
	}
	if err := sameShape(a, b); err != nil {
		return bad(err)
	}
	return &Node{kind: kindOp, op: op, dt: dt, srcs: []*Node{a, b}}
}

func unaryDT(op Op, a DType) (DType, error) {
	switch op {
	case EXP2, LOG2, SIN, SQRT, RECIP:
		if a != F32 {
			return 0, ErrType
		}
		return F32, nil
	case NEG, CAST:
		if a != F32 && a != I32 {
			return 0, ErrType
		}
		return a, nil
	default:
		return 0, ErrOp
	}
}

func binaryDT(op Op, a, b DType) (DType, error) {
	if a != b {
		return 0, ErrType
	}
	switch op {
	case ADD, MUL, MAX:
		if a != F32 && a != I32 {
			return 0, ErrType
		}
		return a, nil
	case IDIV, MOD, XOR, SHL, SHR, OR, AND:
		if a != I32 {
			return 0, ErrType
		}
		return I32, nil
	case CMPLT, CMPNE:
		if a != F32 && a != I32 {
			return 0, ErrType
		}
		return I32, nil
	default:
		return 0, ErrOp
	}
}

func sameShape(ns ...*Node) error {
	var sh []int
	for _, n := range ns {
		s := n.Shape()
		if s == nil {
			continue
		}
		if sh == nil {
			sh = s
			continue
		}
		if !slices.Equal(sh, s) {
			return fmt.Errorf("%w: %v vs %v", ErrShape, sh, s)
		}
	}
	return nil
}
