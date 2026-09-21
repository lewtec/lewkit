package ndarray

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"slices"
	"unsafe"
)

// Number is a tensor element type.
type Number interface {
	~float32 | ~int32 | ~uint8
}

// Tensor is the array brick: a lazy node tree plus, for leaves, a host buffer.
// T is the element type. View ops (Reshape, Permute, …) share the buffer.
// ALU ops build the tree. Slots are assigned at Eval. The graph stays after
// realize so a loop can Resize and run again.
type Tensor[T Number] struct {
	node   *node
	kernel *Kernel
}

func wrap[T Number](n *node) *Tensor[T] {
	if n == nil {
		return &Tensor[T]{node: failed(ErrOp)}
	}
	return &Tensor[T]{node: n}
}

func dtypeOf[T Number]() DType {
	var z T
	switch any(z).(type) {
	case float32:
		return F32
	case int32:
		return I32
	case uint8:
		return U8
	default:
		return 0
	}
}

func (t *Tensor[T]) err() error {
	if t == nil || t.node == nil {
		return ErrOp
	}
	return t.node.err
}

// Const is a splat. It broadcasts against a shaped tensor.
func Const[T Number](v T) *Tensor[T] { return wrap[T](splat(v)) }

// Coord is the logical index on axis, as int32.
func Coord(axis int, shape Shape) *Tensor[int32] { return wrap[int32](coord(axis, shape)) }

// New is a tensor that owns a copy of data. len(data) must equal the shape size.
func New[T Number](data []T, shape Shape) (*Tensor[T], error) {
	tracker, err := Of(shape)
	if err != nil {
		return nil, err
	}
	if len(data) != tracker.Size() {
		return nil, fmt.Errorf("%w: got %d want %d", ErrSize, len(data), tracker.Size())
	}
	buf := &buffer{raw: slices.Clone(asBytes(data)), dtype: dtypeOf[T]()}
	return wrap[T](input(buf, tracker, dtypeOf[T]())), nil
}

// Zeros is a tensor filled with 0.
func Zeros[T Number](shape Shape) (*Tensor[T], error) {
	var z T
	return Full(z, shape)
}

// Ones is a tensor filled with 1.
func Ones[T Number](shape Shape) (*Tensor[T], error) {
	return Full(T(1), shape)
}

// Lowest is the smallest value of T: -inf, min int32, or 0.
func Lowest[T Number]() T {
	var z T
	switch any(z).(type) {
	case float32:
		return any(float32(math.Inf(-1))).(T)
	case int32:
		return any(int32(-1 << 31)).(T)
	default:
		return z
	}
}

// Full is a shaped const (no buffer).
func Full[T Number](v T, shape Shape) (*Tensor[T], error) {
	tracker, err := Of(shape)
	if err != nil {
		return nil, err
	}
	return wrap[T](filled(v, tracker)), nil
}

// Rand fills shape from r. uint8 is raw bytes. int32 and float32 read
// 4-byte cells; a fused Shr maps ints to [0, 2^31) and floats to [0, 1).
func Rand[T Number](r io.Reader, shape Shape) (*Tensor[T], error) {
	tracker, err := Of(shape)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrOp
	}
	n := tracker.Size()
	switch any(*new(T)).(type) {
	case uint8:
		data := make([]uint8, n)
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, err
		}
		t, err := New(data, shape)
		if err != nil {
			return nil, err
		}
		return any(t).(*Tensor[T]), nil
	default:
		raw := make([]byte, n*4)
		if _, err := io.ReadFull(r, raw); err != nil {
			return nil, err
		}
		bits := unsafe.Slice((*int32)(unsafe.Pointer(unsafe.SliceData(raw))), n)
		src, err := New(bits, shape)
		if err != nil {
			return nil, err
		}
		shifted := src.Shr(Const(int32(1)))
		switch any(*new(T)).(type) {
		case int32:
			return any(shifted).(*Tensor[T]), nil
		case float32:
			out := shifted.Cast[float32]().Mul(Const(float32(1.0 / (1 << 31))))
			return any(out).(*Tensor[T]), nil
		default:
			return nil, ErrType
		}
	}
}

// Shape is the logical shape, or nil for a splat. After compile, this is
// the kernel's runtime shape (Resize).
func (t *Tensor[T]) Shape() Shape {
	if t == nil {
		return nil
	}
	if t.kernel != nil {
		return t.kernel.Shape()
	}
	if t.node == nil {
		return nil
	}
	return t.node.Shape()
}

// Size is the number of logical cells.
func (t *Tensor[T]) Size() int {
	if t == nil {
		return 0
	}
	if t.kernel != nil {
		return t.kernel.size
	}
	s := t.Shape()
	if s == nil {
		return 0
	}
	return s.Size()
}

// DType is the element type.
func (t *Tensor[T]) DType() DType {
	if t == nil || t.node == nil {
		return 0
	}
	return t.node.dtype
}

// Tracker is the address map.
func (t *Tensor[T]) Tracker() Tracker {
	if t == nil || t.node == nil {
		return Tracker{}
	}
	return t.node.tracker
}

// EnsureCells grows a leaf host buffer to at least n elements.
func (t *Tensor[T]) EnsureCells(n int) error {
	if t == nil || t.node == nil || t.node.kind != kindInput || t.node.buf == nil {
		return ErrOp
	}
	if t.node.dtype != dtypeOf[T]() {
		return ErrType
	}
	if n < 0 {
		return ErrSize
	}
	need := n * t.node.dtype.size()
	if len(t.node.buf.raw) >= need {
		return nil
	}
	raw := make([]byte, need)
	copy(raw, t.node.buf.raw)
	t.node.buf.raw = raw
	return nil
}

// Buffer is the host storage for a leaf. Views of the same leaf share it.
func (t *Tensor[T]) Buffer() []T {
	if t == nil || t.node == nil || t.node.buf == nil {
		return nil
	}
	if t.node.dtype != dtypeOf[T]() {
		return nil
	}
	return fromBytes[T](t.node.buf.raw)
}

// Data is the contiguous leaf buffer.
func (t *Tensor[T]) Data() ([]T, error) {
	if t == nil {
		return nil, ErrOp
	}
	if t.node == nil || t.node.kind != kindInput || t.node.buf == nil || !t.node.tracker.Contiguous() {
		return nil, ErrOp
	}
	if t.node.dtype != dtypeOf[T]() {
		return nil, ErrType
	}
	return fromBytes[T](t.node.buf.raw), nil
}

// Eval writes into destination. len(destination) must be at least Size.
func (t *Tensor[T]) Eval(ctx context.Context, evaluator Evaluator, destination []T) error {
	if evaluator == nil {
		evaluator = CPU
	}
	return t.realize(ctx, evaluator, destination)
}

// Resize sets the runtime output shape. Rank must match the compiled graph.
func (t *Tensor[T]) Resize(shape Shape) error {
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
func (t *Tensor[T]) Close() error {
	if t == nil {
		return nil
	}
	t.kernel = nil
	return nil
}

func (t *Tensor[T]) realize(ctx context.Context, evaluator Evaluator, destination []T) error {
	if err := t.err(); err != nil {
		return err
	}
	if t.node.kind == kindInput && t.node.buf != nil && t.node.tracker.Contiguous() && t.node.dtype == dtypeOf[T]() {
		n := t.Size()
		elem := t.node.dtype.size()
		need := n * elem
		raw := t.node.buf.raw
		dest := asBytes(destination)
		if len(dest) < need {
			return fmt.Errorf("%w: destination %d < %d", ErrSize, len(destination), n)
		}
		if t.node.tracker.last().offset != 0 || need > len(raw) {
			// fall through to compile
		} else {
			copy(dest[:need], raw[:need])
			return nil
		}
	}
	if err := t.ensure(); err != nil {
		return err
	}
	need := t.kernel.size * t.kernel.outType.size()
	dest := asBytes(destination)
	if len(dest) < need {
		return fmt.Errorf("%w: destination %d < %d", ErrSize, len(destination), t.kernel.size)
	}
	program, err := evaluator.Program(ctx, t.kernel)
	if err != nil {
		return err
	}
	return program.Eval(ctx, dest[:need])
}

func (t *Tensor[T]) ensure() error {
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

// Kernel is the flattened graph for this tensor, or nil before Eval/Resize.
func (t *Tensor[T]) Kernel() *Kernel {
	if t == nil {
		return nil
	}
	return t.kernel
}

func (t *Tensor[T]) withTracker(tr Tracker) *Tensor[T] {
	n := *t.node
	n.tracker = tr
	return &Tensor[T]{node: &n}
}

func (t *Tensor[T]) ensureTracker() (*Tensor[T], error) {
	if t == nil {
		return nil, ErrOp
	}
	if e := t.err(); e != nil {
		return nil, e
	}
	if t.node.kind != kindOp || t.node.tracker.check() == nil {
		return t, nil
	}
	shape := t.Shape()
	base, err := Of(shape)
	if err != nil {
		return nil, err
	}
	n := *t.node
	n.tracker = base
	n.origin = shape.Clone()
	return &Tensor[T]{node: &n}, nil
}

func (t *Tensor[T]) view(tr Tracker, err error) (*Tensor[T], error) {
	if e := t.err(); e != nil {
		return nil, e
	}
	if t.node.kind != kindInput && t.node.kind != kindConst && t.node.kind != kindOp && t.node.kind != kindCoord {
		return nil, ErrOp
	}
	if err != nil {
		return nil, err
	}
	return t.withTracker(tr), nil
}

// Reshape changes the logical shape. Product must match. One -1 is inferred.
func (t *Tensor[T]) Reshape(shape Shape) (*Tensor[T], error) {
	if t == nil {
		return nil, ErrOp
	}
	if t.Shape().Equal(shape) {
		return t, nil
	}
	t, err := t.ensureTracker()
	if err != nil {
		return nil, err
	}
	return t.view(t.node.tracker.Reshape(shape))
}

// Permute reorders axes.
func (t *Tensor[T]) Permute(axes ...int) (*Tensor[T], error) {
	if t == nil {
		return nil, ErrOp
	}
	t, err := t.ensureTracker()
	if err != nil {
		return nil, err
	}
	return t.view(t.node.tracker.Permute(axes...))
}

// Expand broadcasts size-1 axes.
func (t *Tensor[T]) Expand(shape Shape) (*Tensor[T], error) {
	if t == nil {
		return nil, ErrOp
	}
	t, err := t.ensureTracker()
	if err != nil {
		return nil, err
	}
	return t.view(t.node.tracker.Expand(shape))
}

// Pad adds zeros around the logical tensor.
func (t *Tensor[T]) Pad(arg [][2]int) (*Tensor[T], error) {
	if t == nil {
		return nil, ErrOp
	}
	t, err := t.ensureTracker()
	if err != nil {
		return nil, err
	}
	return t.view(t.node.tracker.Pad(arg))
}

// Splat broadcasts a 1-cell tensor.
func (t *Tensor[T]) Splat() (*Tensor[T], error) {
	if t == nil {
		return nil, ErrOp
	}
	return t.view(t.node.tracker.Splat())
}

// Shrink crops to the given half-open ranges.
func (t *Tensor[T]) Shrink(arg [][2]int) (*Tensor[T], error) {
	if t == nil {
		return nil, ErrOp
	}
	t, err := t.ensureTracker()
	if err != nil {
		return nil, err
	}
	return t.view(t.node.tracker.Shrink(arg))
}

// Flip reverses the given axes.
func (t *Tensor[T]) Flip(axes ...int) (*Tensor[T], error) {
	if t == nil {
		return nil, ErrOp
	}
	t, err := t.ensureTracker()
	if err != nil {
		return nil, err
	}
	return t.view(t.node.tracker.Flip(axes...))
}

func (t *Tensor[T]) Add(o *Tensor[T]) *Tensor[T] {
	return t.bin(func(a, b *node) *node { return a.Add(b) }, o)
}
func (t *Tensor[T]) Mul(o *Tensor[T]) *Tensor[T] {
	return t.bin(func(a, b *node) *node { return a.Mul(b) }, o)
}
func (t *Tensor[T]) IDiv(o *Tensor[T]) *Tensor[T] {
	return t.bin(func(a, b *node) *node { return a.IDiv(b) }, o)
}
func (t *Tensor[T]) Max(o *Tensor[T]) *Tensor[T] {
	return t.bin(func(a, b *node) *node { return a.Max(b) }, o)
}
func (t *Tensor[T]) Mod(o *Tensor[T]) *Tensor[T] {
	return t.bin(func(a, b *node) *node { return a.Mod(b) }, o)
}
func (t *Tensor[T]) CmpLt(o *Tensor[T]) *Tensor[int32] {
	return t.cmp(func(a, b *node) *node { return a.CmpLt(b) }, o)
}
func (t *Tensor[T]) CmpNe(o *Tensor[T]) *Tensor[int32] {
	return t.cmp(func(a, b *node) *node { return a.CmpNe(b) }, o)
}
func (t *Tensor[T]) Xor(o *Tensor[T]) *Tensor[T] {
	return t.bin(func(a, b *node) *node { return a.Xor(b) }, o)
}
func (t *Tensor[T]) Shl(o *Tensor[T]) *Tensor[T] {
	return t.bin(func(a, b *node) *node { return a.Shl(b) }, o)
}
func (t *Tensor[T]) Shr(o *Tensor[T]) *Tensor[T] {
	return t.bin(func(a, b *node) *node { return a.Shr(b) }, o)
}
func (t *Tensor[T]) Or(o *Tensor[T]) *Tensor[T] {
	return t.bin(func(a, b *node) *node { return a.Or(b) }, o)
}
func (t *Tensor[T]) And(o *Tensor[T]) *Tensor[T] {
	return t.bin(func(a, b *node) *node { return a.And(b) }, o)
}
func (t *Tensor[T]) Div(o *Tensor[T]) *Tensor[T] {
	return t.bin(func(a, b *node) *node { return a.Div(b) }, o)
}
func (t *Tensor[T]) Equal(o *Tensor[T]) *Tensor[int32] {
	return t.cmp(func(a, b *node) *node { return a.Equal(b) }, o)
}
func (t *Tensor[T]) GreaterEqual(o *Tensor[T]) *Tensor[int32] {
	return t.cmp(func(a, b *node) *node { return a.GreaterEqual(b) }, o)
}

// Exp2 is 2^x.
func (t *Tensor[T]) Exp2() *Tensor[T] { return t.unary((*node).Exp2) }

// Log2 is log2(x).
func (t *Tensor[T]) Log2() *Tensor[T] { return t.unary((*node).Log2) }

// Sin is sin(x) in radians.
func (t *Tensor[T]) Sin() *Tensor[T] { return t.unary((*node).Sin) }

// Sqrt is √x.
func (t *Tensor[T]) Sqrt() *Tensor[T] { return t.unary((*node).Sqrt) }

// Reciprocal is 1/x.
func (t *Tensor[T]) Reciprocal() *Tensor[T] { return t.unary((*node).Recip) }

// Neg is -x.
func (t *Tensor[T]) Neg() *Tensor[T] { return t.unary((*node).Neg) }

// Cast converts elements to To. uint8 saturates 0..255.
func (t *Tensor[T]) Cast[To Number]() *Tensor[To] {
	if t == nil {
		return wrap[To](failed(ErrOp))
	}
	return wrap[To](t.node.Cast(dtypeOf[To]()))
}

// Where is a if t != 0 else b.
func (t *Tensor[T]) Where[U Number](a, b *Tensor[U]) *Tensor[U] {
	if t == nil || a == nil || b == nil {
		return wrap[U](failed(ErrOp))
	}
	return wrap[U](t.node.Where(a.node, b.node))
}

func (t *Tensor[T]) MultiplyAccumulate(b, c *Tensor[T]) *Tensor[T] {
	if t == nil || b == nil || c == nil {
		return wrap[T](failed(ErrOp))
	}
	return wrap[T](t.node.MultiplyAccumulate(b.node, c.node))
}

func (t *Tensor[T]) unary(op func(*node) *node) *Tensor[T] {
	if t == nil {
		return wrap[T](failed(ErrOp))
	}
	return wrap[T](op(t.node))
}

func (t *Tensor[T]) bin(op func(a, b *node) *node, o *Tensor[T]) *Tensor[T] {
	if t == nil || o == nil {
		return wrap[T](failed(ErrOp))
	}
	return wrap[T](op(t.node, o.node))
}

func (t *Tensor[T]) cmp(op func(a, b *node) *node, o *Tensor[T]) *Tensor[int32] {
	if t == nil || o == nil {
		return wrap[int32](failed(ErrOp))
	}
	return wrap[int32](op(t.node, o.node))
}

func asBytes[T Number](data []T) []byte {
	if len(data) == 0 {
		return nil
	}
	var z T
	return unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(data))), len(data)*int(unsafe.Sizeof(z)))
}

func fromBytes[T Number](raw []byte) []T {
	if len(raw) == 0 {
		return nil
	}
	var z T
	n := int(unsafe.Sizeof(z))
	if n == 0 {
		return nil
	}
	return unsafe.Slice((*T)(unsafe.Pointer(unsafe.SliceData(raw))), len(raw)/n)
}
