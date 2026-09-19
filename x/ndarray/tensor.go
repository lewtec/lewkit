package ndarray

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"

	"github.com/lewtec/lewkit/x/ffi/vulkan"
)

// Tensor is a dense row-major array. Zeros and Ones stay lazy until Eval or
// Exec; Rand and New already hold host data. Exec uses a Vulkan Session.
type Tensor struct {
	tracker Tracker
	dtype   DType
	data    []float32
	expr    *Node
	leaves  []*Tensor
	err     error
}

// New is a tensor that owns a copy of data. len(data) must equal the shape size.
func New(data []float32, shape Shape) (*Tensor, error) {
	tracker, err := Of(shape)
	if err != nil {
		return nil, err
	}
	n := tracker.Size()
	if len(data) != n {
		return nil, fmt.Errorf("%w: got %d want %d", ErrSize, len(data), n)
	}
	return &Tensor{tracker: tracker, dtype: F32, data: slices.Clone(data)}, nil
}

// Zeros is a float32 tensor filled with 0.
func Zeros(shape Shape) (*Tensor, error) {
	return Full(0, shape)
}

// Ones is a float32 tensor filled with 1.
func Ones(shape Shape) (*Tensor, error) {
	return Full(1, shape)
}

// Full is a float32 tensor filled with v.
func Full(v float32, shape Shape) (*Tensor, error) {
	tracker, err := Of(shape)
	if err != nil {
		return nil, err
	}
	return &Tensor{tracker: tracker, dtype: F32, expr: filled(v, tracker)}, nil
}

// Rand is a float32 tensor of uniform values in [0, 1).
func Rand(shape Shape) (*Tensor, error) {
	tracker, err := Of(shape)
	if err != nil {
		return nil, err
	}
	data := make([]float32, tracker.Size())
	for i := range data {
		data[i] = rand.Float32()
	}
	return &Tensor{tracker: tracker, dtype: F32, data: data}, nil
}

// Shape is the logical shape.
func (t *Tensor) Shape() Shape {
	if t == nil {
		return nil
	}
	return t.tracker.Shape()
}

// Size is the number of cells.
func (t *Tensor) Size() int {
	if t == nil {
		return 0
	}
	return t.tracker.Size()
}

// DType is the element type.
func (t *Tensor) DType() DType {
	if t == nil {
		return 0
	}
	return t.dtype
}

// Tracker is the address map for this tensor's buffer.
func (t *Tensor) Tracker() Tracker {
	if t == nil {
		return Tracker{}
	}
	return t.tracker
}

// Data evaluates on CPU if needed and returns the host buffer.
func (t *Tensor) Data() ([]float32, error) {
	if err := t.Eval(); err != nil {
		return nil, err
	}
	return t.data, nil
}

// Eval realizes the tensor on the CPU tape.
func (t *Tensor) Eval() error {
	return t.realize(nil, nil)
}

// Exec realizes the tensor on device. device must be non-nil.
func (t *Tensor) Exec(ctx context.Context, device *vulkan.Device) error {
	if device == nil {
		return ErrOp
	}
	return t.realize(ctx, device)
}

func (t *Tensor) realize(ctx context.Context, device *vulkan.Device) error {
	if t == nil {
		return ErrOp
	}
	if t.err != nil {
		return t.err
	}
	if t.expr == nil {
		if t.data == nil {
			return ErrOp
		}
		return nil
	}
	inputs := make([][]float32, len(t.leaves))
	for i, leaf := range t.leaves {
		if err := leaf.realize(ctx, device); err != nil {
			return err
		}
		inputs[i] = leaf.data
	}
	kernel, err := Compile(t.expr)
	if err != nil {
		return err
	}
	defer kernel.Close()
	if device == nil {
		t.data, err = kernel.Eval(inputs...)
	} else {
		var session *Session
		session, err = kernel.Attach(ctx, device)
		if err != nil {
			return err
		}
		defer session.Close()
		out := make([]float32, kernel.size)
		err = session.Run(ctx, out, inputs)
		if err == nil {
			t.data = out
		}
	}
	if err != nil {
		return err
	}
	t.expr = nil
	t.leaves = nil
	return nil
}

func (t *Tensor) Add(o *Tensor) *Tensor   { return t.binaryOp(Add, o) }
func (t *Tensor) Mul(o *Tensor) *Tensor   { return t.binaryOp(Mul, o) }
func (t *Tensor) IDiv(o *Tensor) *Tensor  { return t.binaryOp(IDiv, o) }
func (t *Tensor) Max(o *Tensor) *Tensor   { return t.binaryOp(Max, o) }
func (t *Tensor) Mod(o *Tensor) *Tensor   { return t.binaryOp(Mod, o) }
func (t *Tensor) CmpLt(o *Tensor) *Tensor { return t.binaryOp(CmpLt, o) }
func (t *Tensor) CmpNe(o *Tensor) *Tensor { return t.binaryOp(CmpNe, o) }
func (t *Tensor) Xor(o *Tensor) *Tensor   { return t.binaryOp(Xor, o) }
func (t *Tensor) Shl(o *Tensor) *Tensor   { return t.binaryOp(Shl, o) }
func (t *Tensor) Shr(o *Tensor) *Tensor   { return t.binaryOp(Shr, o) }
func (t *Tensor) Or(o *Tensor) *Tensor    { return t.binaryOp(Or, o) }
func (t *Tensor) And(o *Tensor) *Tensor   { return t.binaryOp(And, o) }
func (t *Tensor) Exp2() *Tensor           { return t.unary(Exp2) }
func (t *Tensor) Log2() *Tensor           { return t.unary(Log2) }
func (t *Tensor) Sin() *Tensor            { return t.unary(Sin) }
func (t *Tensor) Sqrt() *Tensor           { return t.unary(Sqrt) }
func (t *Tensor) Recip() *Tensor          { return t.unary(Recip) }
func (t *Tensor) Neg() *Tensor            { return t.unary(Neg) }
func (t *Tensor) Cast(dtype DType) *Tensor {
	return t.unary(func(n *Node) *Node { return Cast(n, dtype) })
}
func (t *Tensor) Where(a, b *Tensor) *Tensor  { return t.ternaryOp(Where, a, b) }
func (t *Tensor) MulAcc(b, c *Tensor) *Tensor { return t.ternaryOp(MulAcc, b, c) }

func (t *Tensor) binaryOp(op func(a, b *Node) *Node, o *Tensor) *Tensor {
	if t == nil || o == nil {
		return &Tensor{err: ErrOp}
	}
	if t.err != nil {
		return t
	}
	if o.err != nil {
		return o
	}
	leaves := mergeInputs(t.inputs(), o.inputs())
	slots := slotMap(leaves)
	return fromNode(op(t.asNode(slots), o.asNode(slots)), leaves)
}

func (t *Tensor) unary(op func(*Node) *Node) *Tensor {
	if t == nil {
		return &Tensor{err: ErrOp}
	}
	if t.err != nil {
		return t
	}
	leaves := t.inputs()
	return fromNode(op(t.asNode(slotMap(leaves))), leaves)
}

func (t *Tensor) ternaryOp(op func(a, b, c *Node) *Node, b, c *Tensor) *Tensor {
	if t == nil || b == nil || c == nil {
		return &Tensor{err: ErrOp}
	}
	if t.err != nil {
		return t
	}
	if b.err != nil {
		return b
	}
	if c.err != nil {
		return c
	}
	leaves := mergeInputs(t.inputs(), mergeInputs(b.inputs(), c.inputs()))
	slots := slotMap(leaves)
	return fromNode(op(t.asNode(slots), b.asNode(slots), c.asNode(slots)), leaves)
}

func (t *Tensor) inputs() []*Tensor {
	if t == nil {
		return nil
	}
	if t.expr != nil {
		return t.leaves
	}
	return []*Tensor{t}
}

func (t *Tensor) asNode(slots map[*Tensor]int) *Node {
	if t.expr == nil {
		return InTyped(slots[t], t.tracker, t.dtype)
	}
	if len(t.leaves) == 0 {
		return t.expr
	}
	return rebind(t.expr, t.leaves, slots)
}

func fromNode(n *Node, leaves []*Tensor) *Tensor {
	if n == nil {
		return &Tensor{err: ErrOp}
	}
	if n.err != nil {
		return &Tensor{err: n.err}
	}
	tracker, err := Of(n.Shape())
	if err != nil {
		return &Tensor{err: err}
	}
	return &Tensor{tracker: tracker, dtype: n.dtype, expr: n, leaves: leaves}
}

func mergeInputs(a, b []*Tensor) []*Tensor {
	if len(b) == 0 {
		return a
	}
	if len(a) == 0 {
		return b
	}
	out := slices.Clone(a)
	seen := make(map[*Tensor]bool, len(a)+len(b))
	for _, t := range a {
		seen[t] = true
	}
	for _, t := range b {
		if seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

func slotMap(leaves []*Tensor) map[*Tensor]int {
	m := make(map[*Tensor]int, len(leaves))
	for i, t := range leaves {
		m[t] = i
	}
	return m
}

func rebind(n *Node, leaves []*Tensor, slots map[*Tensor]int) *Node {
	memo := make(map[*Node]*Node)
	var walk func(*Node) *Node
	walk = func(n *Node) *Node {
		if n == nil {
			return nil
		}
		if c, ok := memo[n]; ok {
			return c
		}
		cp := *n
		memo[n] = &cp
		if n.kind == kindInput {
			if n.slot < 0 || n.slot >= len(leaves) {
				fail := failed(ErrOp)
				memo[n] = fail
				return fail
			}
			cp.slot = slots[leaves[n.slot]]
		}
		if len(n.sources) > 0 {
			cp.sources = make([]*Node, len(n.sources))
			for i, s := range n.sources {
				cp.sources[i] = walk(s)
			}
		}
		return &cp
	}
	return walk(n)
}
