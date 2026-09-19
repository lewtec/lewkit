package ndarray

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"

	"github.com/lewtec/lewkit/x/ffi/vulkan"
)

// Tensor is the array brick: a lazy node tree plus, for leaves, a host buffer.
// View ops (Reshape, Permute, …) share the buffer. ALU ops build the tree.
// Slots are assigned at Eval/Exec. The graph stays after realize so a loop
// can Resize and run again.
type Tensor struct {
	node    *node
	out     []float32
	kernel  *Kernel
	session *Session
	device  *vulkan.Device
}

func wrap(n *node) *Tensor {
	if n == nil {
		return &Tensor{node: failed(ErrOp)}
	}
	return &Tensor{node: n}
}

func (t *Tensor) err() error {
	if t == nil || t.node == nil {
		return ErrOp
	}
	return t.node.err
}

// Const is a float32 splat. It broadcasts against a shaped tensor.
func Const(v float32) *Tensor { return wrap(splat(v)) }

// ConstInt is an int32 splat.
func ConstInt(v int32) *Tensor { return wrap(splatInt(v)) }

// Coord is the logical index on axis, as int32.
func Coord(axis int, shape Shape) *Tensor { return wrap(coord(axis, shape)) }

// New is a tensor that owns a copy of data. len(data) must equal the shape size.
func New(data []float32, shape Shape) (*Tensor, error) {
	tracker, err := Of(shape)
	if err != nil {
		return nil, err
	}
	if len(data) != tracker.Size() {
		return nil, fmt.Errorf("%w: got %d want %d", ErrSize, len(data), tracker.Size())
	}
	buf := &buffer{data: slices.Clone(data), dtype: F32}
	return wrap(input(buf, tracker, F32)), nil
}

// Zeros is a float32 tensor filled with 0.
func Zeros(shape Shape) (*Tensor, error) { return Full(0, shape) }

// Ones is a float32 tensor filled with 1.
func Ones(shape Shape) (*Tensor, error) { return Full(1, shape) }

// Full is a shaped float32 const (no buffer).
func Full(v float32, shape Shape) (*Tensor, error) {
	tracker, err := Of(shape)
	if err != nil {
		return nil, err
	}
	return wrap(filled(v, tracker)), nil
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
	buf := &buffer{data: data, dtype: F32}
	return wrap(input(buf, tracker, F32)), nil
}

// Shape is the logical shape, or nil for a splat.
func (t *Tensor) Shape() Shape {
	if t == nil || t.node == nil {
		return nil
	}
	return t.node.Shape()
}

// Size is the number of logical cells.
func (t *Tensor) Size() int {
	if t == nil {
		return 0
	}
	s := t.Shape()
	if s == nil {
		return 0
	}
	return s.Size()
}

// DType is the element type.
func (t *Tensor) DType() DType {
	if t == nil || t.node == nil {
		return 0
	}
	return t.node.dtype
}

// Tracker is the address map.
func (t *Tensor) Tracker() Tracker {
	if t == nil || t.node == nil {
		return Tracker{}
	}
	return t.node.tracker
}

// Buffer is the host storage for a leaf. Views of the same leaf share it.
func (t *Tensor) Buffer() []float32 {
	if t == nil || t.node == nil || t.node.buf == nil {
		return nil
	}
	return t.node.buf.data
}

// Data evaluates on CPU if needed and returns the dense host result.
func (t *Tensor) Data() ([]float32, error) {
	if t == nil {
		return nil, ErrOp
	}
	if t.out != nil {
		return t.out, nil
	}
	if err := t.Eval(); err != nil {
		return nil, err
	}
	if t.out != nil {
		return t.out, nil
	}
	return t.Buffer(), nil
}

// Eval realizes the tensor on the CPU tape. The graph is kept.
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

// Resize sets the runtime output shape. Rank must match the compiled graph.
func (t *Tensor) Resize(shape Shape) error {
	if err := t.err(); err != nil {
		return err
	}
	if t.kernel == nil {
		k, err := compile(t.node)
		if err != nil {
			return err
		}
		t.kernel = k
	}
	return t.kernel.Resize(shape)
}

// Close drops a cached kernel and session. Leaf buffers stay.
func (t *Tensor) Close() error {
	if t == nil {
		return nil
	}
	var err error
	if t.session != nil {
		err = t.session.Close()
		t.session = nil
	}
	if t.kernel != nil {
		if e := t.kernel.Close(); err == nil {
			err = e
		}
		t.kernel = nil
	}
	t.device = nil
	return err
}

func (t *Tensor) realize(ctx context.Context, device *vulkan.Device) error {
	if err := t.err(); err != nil {
		return err
	}
	if t.node.kind == kindInput && t.node.buf != nil && t.node.tracker.Contiguous() {
		t.out = t.node.buf.data
		return nil
	}
	if err := t.ensure(ctx, device); err != nil {
		return err
	}
	inputs := make([][]float32, len(t.kernel.bufs))
	for i, b := range t.kernel.bufs {
		if b == nil {
			return ErrOp
		}
		inputs[i] = b.data
	}
	if device == nil {
		out, err := t.kernel.Eval(inputs...)
		if err != nil {
			return err
		}
		t.out = out
		return nil
	}
	if cap(t.out) < t.kernel.size {
		t.out = make([]float32, t.kernel.size)
	} else {
		t.out = t.out[:t.kernel.size]
	}
	return t.session.Run(ctx, t.out, inputs)
}

func (t *Tensor) ensure(ctx context.Context, device *vulkan.Device) error {
	if t.kernel != nil && t.device == device {
		if device != nil && t.session != nil {
			return nil
		}
		if device == nil {
			return nil
		}
	}
	if t.kernel != nil && t.device != device {
		if err := t.Close(); err != nil {
			return err
		}
	}
	if t.kernel == nil {
		k, err := compile(t.node)
		if err != nil {
			return err
		}
		t.kernel = k
	}
	t.device = device
	if device == nil {
		return nil
	}
	if t.session != nil {
		return nil
	}
	sess, err := t.kernel.Attach(ctx, device)
	if err != nil {
		return err
	}
	t.session = sess
	return nil
}

func (t *Tensor) withTracker(tr Tracker) *Tensor {
	n := *t.node
	n.tracker = tr
	return &Tensor{node: &n}
}

func (t *Tensor) view(tr Tracker, err error) (*Tensor, error) {
	if e := t.err(); e != nil {
		return nil, e
	}
	if err != nil {
		return nil, err
	}
	if t.node.kind != kindInput && t.node.kind != kindConst {
		return nil, ErrOp
	}
	return t.withTracker(tr), nil
}

// Reshape changes the logical shape. Product must match. One -1 is inferred.
func (t *Tensor) Reshape(shape Shape) (*Tensor, error) {
	if t == nil {
		return nil, ErrOp
	}
	return t.view(t.node.tracker.Reshape(shape))
}

// Permute reorders axes.
func (t *Tensor) Permute(axes ...int) (*Tensor, error) {
	if t == nil {
		return nil, ErrOp
	}
	return t.view(t.node.tracker.Permute(axes...))
}

// Expand broadcasts size-1 axes.
func (t *Tensor) Expand(shape Shape) (*Tensor, error) {
	if t == nil {
		return nil, ErrOp
	}
	return t.view(t.node.tracker.Expand(shape))
}

// Pad adds zeros around the logical tensor.
func (t *Tensor) Pad(arg [][2]int) (*Tensor, error) {
	if t == nil {
		return nil, ErrOp
	}
	return t.view(t.node.tracker.Pad(arg))
}

// Shrink crops to the given half-open ranges.
func (t *Tensor) Shrink(arg [][2]int) (*Tensor, error) {
	if t == nil {
		return nil, ErrOp
	}
	return t.view(t.node.tracker.Shrink(arg))
}

// Flip reverses the given axes.
func (t *Tensor) Flip(axes ...int) (*Tensor, error) {
	if t == nil {
		return nil, ErrOp
	}
	return t.view(t.node.tracker.Flip(axes...))
}

func (t *Tensor) Add(o *Tensor) *Tensor { return t.bin(func(a, b *node) *node { return a.Add(b) }, o) }
func (t *Tensor) Mul(o *Tensor) *Tensor { return t.bin(func(a, b *node) *node { return a.Mul(b) }, o) }
func (t *Tensor) IDiv(o *Tensor) *Tensor {
	return t.bin(func(a, b *node) *node { return a.IDiv(b) }, o)
}
func (t *Tensor) Max(o *Tensor) *Tensor { return t.bin(func(a, b *node) *node { return a.Max(b) }, o) }
func (t *Tensor) Mod(o *Tensor) *Tensor { return t.bin(func(a, b *node) *node { return a.Mod(b) }, o) }
func (t *Tensor) CmpLt(o *Tensor) *Tensor {
	return t.bin(func(a, b *node) *node { return a.CmpLt(b) }, o)
}
func (t *Tensor) CmpNe(o *Tensor) *Tensor {
	return t.bin(func(a, b *node) *node { return a.CmpNe(b) }, o)
}
func (t *Tensor) Xor(o *Tensor) *Tensor { return t.bin(func(a, b *node) *node { return a.Xor(b) }, o) }
func (t *Tensor) Shl(o *Tensor) *Tensor { return t.bin(func(a, b *node) *node { return a.Shl(b) }, o) }
func (t *Tensor) Shr(o *Tensor) *Tensor { return t.bin(func(a, b *node) *node { return a.Shr(b) }, o) }
func (t *Tensor) Or(o *Tensor) *Tensor  { return t.bin(func(a, b *node) *node { return a.Or(b) }, o) }
func (t *Tensor) And(o *Tensor) *Tensor { return t.bin(func(a, b *node) *node { return a.And(b) }, o) }
func (t *Tensor) Div(o *Tensor) *Tensor { return t.bin(func(a, b *node) *node { return a.Div(b) }, o) }
func (t *Tensor) Equal(o *Tensor) *Tensor {
	return t.bin(func(a, b *node) *node { return a.Equal(b) }, o)
}
func (t *Tensor) GreaterEqual(o *Tensor) *Tensor {
	return t.bin(func(a, b *node) *node { return a.GreaterEqual(b) }, o)
}
func (t *Tensor) Exp2() *Tensor  { return wrap(t.node.Exp2()) }
func (t *Tensor) Log2() *Tensor  { return wrap(t.node.Log2()) }
func (t *Tensor) Sin() *Tensor   { return wrap(t.node.Sin()) }
func (t *Tensor) Sqrt() *Tensor  { return wrap(t.node.Sqrt()) }
func (t *Tensor) Recip() *Tensor { return wrap(t.node.Recip()) }
func (t *Tensor) Neg() *Tensor   { return wrap(t.node.Neg()) }
func (t *Tensor) Cast(dtype DType) *Tensor {
	if t == nil {
		return wrap(failed(ErrOp))
	}
	return wrap(t.node.Cast(dtype))
}
func (t *Tensor) Where(a, b *Tensor) *Tensor {
	if t == nil || a == nil || b == nil {
		return wrap(failed(ErrOp))
	}
	return wrap(t.node.Where(a.node, b.node))
}
func (t *Tensor) MulAcc(b, c *Tensor) *Tensor {
	if t == nil || b == nil || c == nil {
		return wrap(failed(ErrOp))
	}
	return wrap(t.node.MulAcc(b.node, c.node))
}

func (t *Tensor) bin(op func(a, b *node) *node, o *Tensor) *Tensor {
	if t == nil || o == nil {
		return wrap(failed(ErrOp))
	}
	return wrap(op(t.node, o.node))
}
