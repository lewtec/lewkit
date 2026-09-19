package ndarray

import (
	"fmt"
	"math"
	"sync"
)

type kind uint8

const (
	kindConst kind = iota
	kindInput
	kindCoord
	kindOp
)

// node is one vertex of a fused kernel: a const, a buffer view, or an ALU op.
// kindInput nodes share a *buffer; slots are assigned at compile.
type node struct {
	kind    kind
	op      Op
	dtype   DType
	sources []*node
	tracker Tracker
	buf     *buffer
	slot    int // Coord axis
	bits    uint32
	err     error
}

type buffer struct {
	raw   []byte
	dtype DType
}

func (b *buffer) cells() int {
	if b == nil {
		return 0
	}
	n := b.dtype.size()
	if n == 0 {
		return 0
	}
	return len(b.raw) / n
}

func (b *buffer) word(i int) uint32 {
	if b == nil || i < 0 {
		return 0
	}
	switch b.dtype {
	case U8:
		if i >= len(b.raw) {
			return 0
		}
		return uint32(b.raw[i])
	default:
		off := i * 4
		if off+4 > len(b.raw) {
			return 0
		}
		return uint32(b.raw[off]) | uint32(b.raw[off+1])<<8 | uint32(b.raw[off+2])<<16 | uint32(b.raw[off+3])<<24
	}
}

func failed(err error) *node { return &node{err: err} }

func input(buf *buffer, tracker Tracker, dtype DType) *node {
	if buf == nil || tracker.check() != nil || (dtype != F32 && dtype != I32 && dtype != U8) {
		return failed(ErrOp)
	}
	return &node{kind: kindInput, dtype: dtype, tracker: tracker, buf: buf}
}

type splatKey struct {
	bits  uint32
	dtype DType
}

var splats = struct {
	mu   sync.Mutex
	node map[splatKey]*node
}{node: map[splatKey]*node{}}

func splat[T Number](v T) *node {
	key := splatKey{bits: bitsOf(v), dtype: dtypeOf[T]()}
	splats.mu.Lock()
	defer splats.mu.Unlock()
	if n := splats.node[key]; n != nil {
		return n
	}
	n := &node{kind: kindConst, dtype: key.dtype, bits: key.bits}
	splats.node[key] = n
	return n
}

func filled[T Number](v T, tracker Tracker) *node {
	if err := tracker.check(); err != nil {
		return failed(err)
	}
	return &node{kind: kindConst, dtype: dtypeOf[T](), bits: bitsOf(v), tracker: tracker}
}

func bitsOf[T Number](v T) uint32 {
	switch x := any(v).(type) {
	case float32:
		return math.Float32bits(x)
	case int32:
		return uint32(x)
	case uint8:
		return uint32(x)
	default:
		return 0
	}
}

func coord(axis int, shape Shape) *node {
	tracker, err := Of(shape)
	if err != nil {
		return failed(err)
	}
	if axis < 0 || axis >= shape.Rank() {
		return failed(ErrAxis)
	}
	return &node{kind: kindCoord, dtype: I32, tracker: tracker, slot: axis}
}

// Shape is the logical shape, or nil for a splat.
func (n *node) Shape() Shape {
	if n == nil || n.err != nil {
		return nil
	}
	if n.kind == kindConst {
		if len(n.tracker.views) == 0 {
			return nil
		}
		return n.tracker.Shape()
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
func (n *node) DType() DType {
	if n == nil {
		return 0
	}
	return n.dtype
}

func (n *node) Add(b *node) *node   { return binaryOp(ADD, n, b) }
func (n *node) Mul(b *node) *node   { return binaryOp(MUL, n, b) }
func (n *node) IDiv(b *node) *node  { return binaryOp(IDIV, n, b) }
func (n *node) Max(b *node) *node   { return binaryOp(MAX, n, b) }
func (n *node) Mod(b *node) *node   { return binaryOp(MOD, n, b) }
func (n *node) CmpLt(b *node) *node { return binaryOp(CMPLT, n, b) }
func (n *node) CmpNe(b *node) *node { return binaryOp(CMPNE, n, b) }
func (n *node) Xor(b *node) *node   { return binaryOp(XOR, n, b) }
func (n *node) Shl(b *node) *node   { return binaryOp(SHL, n, b) }
func (n *node) Shr(b *node) *node   { return binaryOp(SHR, n, b) }
func (n *node) Or(b *node) *node    { return binaryOp(OR, n, b) }
func (n *node) And(b *node) *node   { return binaryOp(AND, n, b) }
func (n *node) Exp2() *node         { return unary(EXP2, n) }
func (n *node) Log2() *node         { return unary(LOG2, n) }
func (n *node) Sin() *node          { return unary(SIN, n) }
func (n *node) Sqrt() *node         { return unary(SQRT, n) }
func (n *node) Recip() *node        { return unary(RECIP, n) }
func (n *node) Neg() *node          { return unary(NEG, n) }
func (n *node) Div(b *node) *node   { return n.Mul(b.Recip()) }
func (n *node) Equal(b *node) *node {
	return n.CmpNe(b).CmpNe(splat(int32(1)))
}
func (n *node) GreaterEqual(b *node) *node {
	return n.CmpLt(b).CmpNe(splat(int32(1)))
}
func (n *node) Cast(dtype DType) *node              { return castNode(n, dtype) }
func (n *node) Where(a, b *node) *node              { return whereNode(n, a, b) }
func (n *node) MultiplyAccumulate(b, c *node) *node { return multiplyAccumulateNode(n, b, c) }

func castNode(a *node, dtype DType) *node {
	n := unary(CAST, a)
	if n.err != nil {
		return n
	}
	if dtype != F32 && dtype != I32 && dtype != U8 {
		return failed(ErrType)
	}
	n.dtype = dtype
	return n
}

// Where is a if p != 0 else b. Call as p.Where(a, b).
func whereNode(p, a, b *node) *node {
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
	return &node{kind: kindOp, op: WHERE, dtype: a.dtype, sources: []*node{p, a, b}}
}

// MultiplyAccumulate is a*b + c.
func multiplyAccumulateNode(a, b, c *node) *node {
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
	return &node{kind: kindOp, op: MultiplyAccumulate, dtype: a.dtype, sources: []*node{a, b, c}}
}

func unary(op Op, a *node) *node {
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
	return &node{kind: kindOp, op: op, dtype: dtype, sources: []*node{a}}
}

func binaryOp(op Op, a, b *node) *node {
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
	return &node{kind: kindOp, op: op, dtype: dtype, sources: []*node{a, b}}
}

func unaryType(op Op, a DType) (DType, error) {
	switch op {
	case EXP2, LOG2, SIN, SQRT, RECIP:
		if a != F32 {
			return 0, ErrType
		}
		return F32, nil
	case NEG, CAST:
		if a != F32 && a != I32 && a != U8 {
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

func sameShape(ns ...*node) error {
	var shape Shape
	for _, n := range ns {
		s := n.Shape()
		if s == nil {
			continue
		}
		if shape == nil {
			shape = s
			continue
		}
		if !shape.Equal(s) {
			return fmt.Errorf("%w: %v vs %v", ErrShape, shape, s)
		}
	}
	return nil
}
