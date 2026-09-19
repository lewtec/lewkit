package ndarray

import (
	"context"
	"fmt"
	"image"
	"log/slog"
	"math/rand/v2"
	"slices"
)

// Tensor is the array brick: a lazy node tree plus, for leaves, a host buffer.
// View ops (Reshape, Permute, …) share the buffer. ALU ops build the tree.
// Slots are assigned at Eval. The graph stays after realize so a loop
// can Resize and run again.
type Tensor struct {
	node        *node
	kernel      *Kernel
	rgbaScratch []float32
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

// Data is the leaf buffer for a contiguous input.
func (t *Tensor) Data() ([]float32, error) {
	if t == nil {
		return nil, ErrOp
	}
	if t.node != nil && t.node.kind == kindInput && t.node.buf != nil && t.node.tracker.Contiguous() {
		return t.node.buf.data, nil
	}
	return nil, ErrOp
}

// Eval writes the tensor into destination. len(destination) must be at least Size.
func (t *Tensor) Eval(ctx context.Context, evaluator Evaluator, destination []float32) error {
	if evaluator == nil {
		evaluator = CPU
	}
	return t.realize(ctx, evaluator, destination)
}

// EvalRGBA writes packed 0..255 RGBA into destination. Bounds must match Size.
func (t *Tensor) EvalRGBA(ctx context.Context, evaluator Evaluator, destination *image.RGBA) error {
	if destination == nil {
		return ErrOp
	}
	if err := t.err(); err != nil {
		return err
	}
	if err := t.ensure(); err != nil {
		return err
	}
	size := t.kernel.size
	if destination.Rect.Dx()*destination.Rect.Dy()*4 != size {
		return fmt.Errorf("%w: image %d×%d×4 != %d", ErrSize, destination.Rect.Dx(), destination.Rect.Dy(), size)
	}
	if cap(t.rgbaScratch) < size {
		t.rgbaScratch = make([]float32, size)
	} else {
		t.rgbaScratch = t.rgbaScratch[:size]
	}
	if err := t.Eval(ctx, evaluator, t.rgbaScratch); err != nil {
		return err
	}
	packRGBA(destination, t.rgbaScratch)
	return nil
}

// Resize sets the runtime output shape. Rank must match the compiled graph.
func (t *Tensor) Resize(shape Shape) error {
	if err := t.err(); err != nil {
		return err
	}
	if t.kernel == nil {
		kernel, err := compile(t.node)
		if err != nil {
			return err
		}
		t.kernel = kernel
	}
	return t.kernel.Resize(shape)
}

// Close drops a cached kernel. Leaf buffers stay.
func (t *Tensor) Close() error {
	if t == nil {
		return nil
	}
	if t.kernel == nil {
		return nil
	}
	err := t.kernel.Close()
	t.kernel = nil
	return err
}

func (t *Tensor) realize(ctx context.Context, evaluator Evaluator, destination []float32) error {
	if err := t.err(); err != nil {
		return err
	}
	if t.node.kind == kindInput && t.node.buf != nil && t.node.tracker.Contiguous() {
		data := t.node.buf.data
		if len(destination) < len(data) {
			return fmt.Errorf("%w: destination %d < %d", ErrSize, len(destination), len(data))
		}
		copy(destination, data)
		return nil
	}
	if err := t.ensure(); err != nil {
		return err
	}
	if len(destination) < t.kernel.size {
		return fmt.Errorf("%w: destination %d < %d", ErrSize, len(destination), t.kernel.size)
	}
	return evaluator.Run(ctx, t.kernel, destination[:t.kernel.size])
}

func (t *Tensor) ensure() error {
	if t.kernel != nil {
		return nil
	}
	kernel, err := compile(t.node)
	if err != nil {
		return err
	}
	t.kernel = kernel
	slog.Debug("ndarray compile", "shape", kernel.shape, "size", kernel.size)
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
