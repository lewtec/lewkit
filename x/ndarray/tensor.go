package ndarray

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"sync"
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

// Const is a float32 splat. It broadcasts against a shaped tensor.
func Const(v float32) *Tensor[float32] { return wrap[float32](splat(v)) }

// ConstInt is an int32 splat.
func ConstInt(v int32) *Tensor[int32] { return wrap[int32](splatInt(v)) }

// Coord is the logical index on axis, as int32.
func Coord(axis int, shape Shape) *Tensor[int32] { return wrap[int32](coord(axis, shape)) }

// New is a tensor that owns a copy of data. len(data) must equal the shape size.
func New(data []float32, shape Shape) (*Tensor[float32], error) {
	tracker, err := Of(shape)
	if err != nil {
		return nil, err
	}
	if len(data) != tracker.Size() {
		return nil, fmt.Errorf("%w: got %d want %d", ErrSize, len(data), tracker.Size())
	}
	buf := &buffer{data: slices.Clone(data), dtype: F32}
	return wrap[float32](input(buf, tracker, F32)), nil
}

// Zeros is a float32 tensor filled with 0.
func Zeros(shape Shape) (*Tensor[float32], error) { return Full(0, shape) }

// Ones is a float32 tensor filled with 1.
func Ones(shape Shape) (*Tensor[float32], error) { return Full(1, shape) }

// Full is a shaped float32 const (no buffer).
func Full(v float32, shape Shape) (*Tensor[float32], error) {
	tracker, err := Of(shape)
	if err != nil {
		return nil, err
	}
	return wrap[float32](filled(v, tracker)), nil
}

// Rand is a float32 tensor of uniform values in [0, 1) from r
// (4 bytes per cell as uint32 / 2^32).
func Rand(r io.Reader, shape Shape) (*Tensor[float32], error) {
	tracker, err := Of(shape)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrOp
	}
	data := make([]float32, tracker.Size())
	var raw [4]byte
	for i := range data {
		if _, err := io.ReadFull(r, raw[:]); err != nil {
			return nil, err
		}
		data[i] = float32(float64(binary.LittleEndian.Uint32(raw[:])) / (1 << 32))
	}
	buf := &buffer{data: data, dtype: F32}
	return wrap[float32](input(buf, tracker, F32)), nil
}

// RandInt is an int32 tensor of uniform values in [0, 2^31) from r
// (4 bytes per cell).
func RandInt(r io.Reader, shape Shape) (*Tensor[int32], error) {
	tracker, err := Of(shape)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrOp
	}
	data := make([]float32, tracker.Size())
	var raw [4]byte
	for i := range data {
		if _, err := io.ReadFull(r, raw[:]); err != nil {
			return nil, err
		}
		v := int32(binary.LittleEndian.Uint32(raw[:]) >> 1)
		data[i] = float32(v)
	}
	buf := &buffer{data: data, dtype: I32}
	return wrap[int32](input(buf, tracker, I32)), nil
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

// Buffer is the host storage for a float32 leaf. Views of the same leaf share it.
func (t *Tensor[T]) Buffer() []T {
	if t == nil || t.node == nil || t.node.buf == nil {
		return nil
	}
	if t.node.dtype != F32 || dtypeOf[T]() != F32 {
		return nil
	}
	return bitsAs[T](t.node.buf.data)
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
	data := t.node.buf.data
	if dtypeOf[T]() == F32 {
		return bitsAs[T](data), nil
	}
	out := make([]T, len(data))
	writeHost(out, data)
	return out, nil
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
		data := t.node.buf.data
		if len(destination) < len(data) {
			return fmt.Errorf("%w: destination %d < %d", ErrSize, len(destination), len(data))
		}
		writeHost(destination, data)
		return nil
	}
	if err := t.ensure(); err != nil {
		return err
	}
	if len(destination) < t.kernel.size {
		return fmt.Errorf("%w: destination %d < %d", ErrSize, len(destination), t.kernel.size)
	}
	program, err := evaluator.Program(ctx, t.kernel)
	if err != nil {
		return err
	}
	if host, ok := floatHost(destination[:t.kernel.size]); ok {
		return program.Eval(ctx, host)
	}
	scratch := borrowHostFloat(t.kernel.size)
	err = program.Eval(ctx, scratch.data)
	if err == nil {
		writeHost(destination, scratch.data)
	}
	returnHostFloat(scratch)
	return err
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

func (t *Tensor[T]) view(tr Tracker, err error) (*Tensor[T], error) {
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
func (t *Tensor[T]) Reshape(shape Shape) (*Tensor[T], error) {
	if t == nil {
		return nil, ErrOp
	}
	return t.view(t.node.tracker.Reshape(shape))
}

// Permute reorders axes.
func (t *Tensor[T]) Permute(axes ...int) (*Tensor[T], error) {
	if t == nil {
		return nil, ErrOp
	}
	return t.view(t.node.tracker.Permute(axes...))
}

// Expand broadcasts size-1 axes.
func (t *Tensor[T]) Expand(shape Shape) (*Tensor[T], error) {
	if t == nil {
		return nil, ErrOp
	}
	return t.view(t.node.tracker.Expand(shape))
}

// Pad adds zeros around the logical tensor.
func (t *Tensor[T]) Pad(arg [][2]int) (*Tensor[T], error) {
	if t == nil {
		return nil, ErrOp
	}
	return t.view(t.node.tracker.Pad(arg))
}

// Shrink crops to the given half-open ranges.
func (t *Tensor[T]) Shrink(arg [][2]int) (*Tensor[T], error) {
	if t == nil {
		return nil, ErrOp
	}
	return t.view(t.node.tracker.Shrink(arg))
}

// Flip reverses the given axes.
func (t *Tensor[T]) Flip(axes ...int) (*Tensor[T], error) {
	if t == nil {
		return nil, ErrOp
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
func Cast[To Number, From Number](t *Tensor[From]) *Tensor[To] {
	if t == nil {
		return wrap[To](failed(ErrOp))
	}
	return wrap[To](t.node.Cast(dtypeOf[To]()))
}

// Where is a if cond != 0 else b.
func Where[T Number](cond *Tensor[int32], a, b *Tensor[T]) *Tensor[T] {
	if cond == nil || a == nil || b == nil {
		return wrap[T](failed(ErrOp))
	}
	return wrap[T](cond.node.Where(a.node, b.node))
}

func (t *Tensor[T]) MulAcc(b, c *Tensor[T]) *Tensor[T] {
	if t == nil || b == nil || c == nil {
		return wrap[T](failed(ErrOp))
	}
	return wrap[T](t.node.MulAcc(b.node, c.node))
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

func bitsAs[T Number](data []float32) []T {
	if len(data) == 0 {
		return nil
	}
	return unsafe.Slice((*T)(unsafe.Pointer(unsafe.SliceData(data))), len(data))
}

func floatHost[T Number](destination []T) ([]float32, bool) {
	if dtypeOf[T]() != F32 {
		return nil, false
	}
	if len(destination) == 0 {
		return nil, true
	}
	return unsafe.Slice((*float32)(unsafe.Pointer(unsafe.SliceData(destination))), len(destination)), true
}

func writeHost[T Number](destination []T, source []float32) {
	if len(destination) == 0 || len(source) == 0 {
		return
	}
	n := min(len(destination), len(source))
	switch dtypeOf[T]() {
	case F32:
		copy(unsafe.Slice((*float32)(unsafe.Pointer(unsafe.SliceData(destination))), n), source[:n])
	case I32:
		out := unsafe.Slice((*int32)(unsafe.Pointer(unsafe.SliceData(destination))), n)
		for i := 0; i < n; i++ {
			out[i] = int32(source[i])
		}
	case U8:
		out := unsafe.Slice((*uint8)(unsafe.Pointer(unsafe.SliceData(destination))), n)
		for i := 0; i < n; i++ {
			out[i] = uint8(source[i])
		}
	}
}

type hostFloatBuffer struct {
	data []float32
}

var hostFloatPool = sync.Pool{New: func() any { return &hostFloatBuffer{} }}

func borrowHostFloat(n int) *hostFloatBuffer {
	b := hostFloatPool.Get().(*hostFloatBuffer)
	if cap(b.data) < n {
		b.data = make([]float32, n)
	} else {
		b.data = b.data[:n]
	}
	return b
}

func returnHostFloat(b *hostFloatBuffer) {
	if b == nil {
		return
	}
	b.data = b.data[:0]
	hostFloatPool.Put(b)
}
