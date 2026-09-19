package ndarray

import (
	"fmt"
	"math"
	"slices"
)

type kind uint8

const (
	kindConst kind = iota
	kindInput
	kindCoord
	kindOp
)

// Node is one vertex of a fused kernel: a const, an input view, or an ALU op.
type Node struct {
	kind    kind
	op      Op
	dtype   DType
	sources []*Node
	tracker Tracker
	slot    int
	bits    uint32
	err     error
}

func failed(err error) *Node { return &Node{err: err} }

// In is a float32 input buffer at slot, addressed by tracker.
func In(slot int, tracker Tracker) *Node {
	return InTyped(slot, tracker, F32)
}

// InTyped is an input buffer at slot with the given element type.
func InTyped(slot int, tracker Tracker, dtype DType) *Node {
	if slot < 0 || tracker.check() != nil || (dtype != F32 && dtype != I32) {
		return failed(ErrOp)
	}
	return &Node{kind: kindInput, dtype: dtype, tracker: tracker, slot: slot}
}

// Const is a float32 splat.
func Const(v float32) *Node {
	return &Node{kind: kindConst, dtype: F32, bits: math.Float32bits(v)}
}

// ConstInt is an int32 splat.
func ConstInt(v int32) *Node {
	return &Node{kind: kindConst, dtype: I32, bits: uint32(v)}
}

// Coord is the logical index on axis, as int32. shape is the tensor it indexes.
func Coord(axis int, shape ...int) *Node {
	tracker, err := Of(shape...)
	if err != nil {
		return failed(err)
	}
	if axis < 0 || axis >= len(shape) {
		return failed(ErrAxis)
	}
	return &Node{kind: kindCoord, dtype: I32, tracker: tracker, slot: axis}
}

// Div is a * (1/b).
func Div(a, b *Node) *Node { return Mul(a, Recip(b)) }

// Equal is 1 if a == b, else 0.
func Equal(a, b *Node) *Node { return CmpNe(CmpNe(a, b), ConstInt(1)) }

// GreaterEqual is 1 if a >= b, else 0.
func GreaterEqual(a, b *Node) *Node { return CmpNe(CmpLt(a, b), ConstInt(1)) }

// Shape is the logical shape, or nil for a splat.
func (n *Node) Shape() []int {
	if n == nil || n.err != nil {
		return nil
	}
	if n.kind == kindInput {
		shape := n.tracker.Shape()
		if len(shape) == 0 {
			return nil
		}
		return shape
	}
	if n.kind == kindCoord {
		return n.tracker.Shape()
	}
	if n.kind == kindOp {
		for _, s := range n.sources {
			if shape := s.Shape(); shape != nil {
				return shape
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
	return n.dtype
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
func (n *Node) Cast(dtype DType) *Node  { return Cast(n, dtype) }
func (n *Node) Where(a, b *Node) *Node  { return Where(n, a, b) }
func (n *Node) MulAcc(b, c *Node) *Node { return MulAcc(n, b, c) }

// Add is a + b.
func Add(a, b *Node) *Node { return binaryOp(ADD, a, b) }

// Mul is a * b.
func Mul(a, b *Node) *Node { return binaryOp(MUL, a, b) }

// IDiv is integer a / b.
func IDiv(a, b *Node) *Node { return binaryOp(IDIV, a, b) }

// Max is max(a, b).
func Max(a, b *Node) *Node { return binaryOp(MAX, a, b) }

// Mod is integer a % b.
func Mod(a, b *Node) *Node { return binaryOp(MOD, a, b) }

// CmpLt is 1 if a < b, else 0.
func CmpLt(a, b *Node) *Node { return binaryOp(CMPLT, a, b) }

// CmpNe is 1 if a != b, else 0.
func CmpNe(a, b *Node) *Node { return binaryOp(CMPNE, a, b) }

// Xor is a ^ b.
func Xor(a, b *Node) *Node { return binaryOp(XOR, a, b) }

// Shl is a << b.
func Shl(a, b *Node) *Node { return binaryOp(SHL, a, b) }

// Shr is a >> b.
func Shr(a, b *Node) *Node { return binaryOp(SHR, a, b) }

// Or is a | b.
func Or(a, b *Node) *Node { return binaryOp(OR, a, b) }

// And is a & b.
func And(a, b *Node) *Node { return binaryOp(AND, a, b) }

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

// Cast converts a to dtype.
func Cast(a *Node, dtype DType) *Node {
	n := unary(CAST, a)
	if n.err != nil {
		return n
	}
	if dtype != F32 && dtype != I32 {
		return failed(ErrType)
	}
	n.dtype = dtype
	return n
}

// Where is a if p != 0 else b. Call as Where(p, a, b) or p.Where(a, b).
func Where(p, a, b *Node) *Node {
	if p == nil || a == nil || b == nil {
		return failed(ErrOp)
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
	if p.dtype != I32 || a.dtype != b.dtype {
		return failed(ErrType)
	}
	if err := sameShape(p, a, b); err != nil {
		return failed(err)
	}
	return &Node{kind: kindOp, op: WHERE, dtype: a.dtype, sources: []*Node{p, a, b}}
}

// MulAcc is a*b + c.
func MulAcc(a, b, c *Node) *Node {
	if a == nil || b == nil || c == nil {
		return failed(ErrOp)
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
	if a.dtype != b.dtype || a.dtype != c.dtype {
		return failed(ErrType)
	}
	if err := sameShape(a, b, c); err != nil {
		return failed(err)
	}
	return &Node{kind: kindOp, op: MULACC, dtype: a.dtype, sources: []*Node{a, b, c}}
}

func unary(op Op, a *Node) *Node {
	if a == nil {
		return failed(ErrOp)
	}
	if a.err != nil {
		return a
	}
	dtype, err := unaryType(op, a.dtype)
	if err != nil {
		return failed(err)
	}
	return &Node{kind: kindOp, op: op, dtype: dtype, sources: []*Node{a}}
}

func binaryOp(op Op, a, b *Node) *Node {
	if a == nil || b == nil {
		return failed(ErrOp)
	}
	if a.err != nil {
		return a
	}
	if b.err != nil {
		return b
	}
	dtype, err := binaryType(op, a.dtype, b.dtype)
	if err != nil {
		return failed(err)
	}
	if err := sameShape(a, b); err != nil {
		return failed(err)
	}
	return &Node{kind: kindOp, op: op, dtype: dtype, sources: []*Node{a, b}}
}

func unaryType(op Op, a DType) (DType, error) {
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

func binaryType(op Op, a, b DType) (DType, error) {
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
	var shape []int
	for _, n := range ns {
		s := n.Shape()
		if s == nil {
			continue
		}
		if shape == nil {
			shape = s
			continue
		}
		if !slices.Equal(shape, s) {
			return fmt.Errorf("%w: %v vs %v", ErrShape, shape, s)
		}
	}
	return nil
}
