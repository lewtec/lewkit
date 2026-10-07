package ndarray

import (
	"encoding/binary"
	"fmt"
	"math"
	"slices"
)

const (
	// localSize is the compute workgroup. Vulkan 1.1 guarantees 128 invocations.
	localSize = 128
	// PushBytes is n plus d0..d3. Rank above 4 cannot be lowered.
	PushBytes   = 20
	maxPushRank = 4
)

// Kernel is the flattened graph: nodes, leaf buffers, compile-time shape.
// [Kernel.Code] is the schedule a backend renders. Resize updates the runtime
// shape only.
type Kernel struct {
	root    *node
	order   []*node
	bufs    []*buffer
	built   Shape
	shape   Shape
	outType DType
	size    int
}

func compile(expr *node) (*Kernel, error) {
	if expr == nil {
		return nil, ErrOp
	}
	if expr.err != nil {
		return nil, expr.err
	}
	shape := expr.Shape()
	if shape == nil {
		return nil, fmt.Errorf("%w: constant kernel", ErrShape)
	}
	size := shape.Size()
	if size < 0 {
		return nil, ErrShape
	}
	if needsRewrite(expr, map[*node]bool{}) {
		expr = rewrite(expr, nil, map[rewriteMemo]*node{})
	}
	expr = optimize(expr)
	order, bufs, err := flatten(expr)
	if err != nil {
		return nil, err
	}
	built := shape.Clone()
	return &Kernel{root: expr, order: order, bufs: bufs, built: built, shape: built.Clone(), outType: expr.dtype, size: size}, nil
}

// Resize sets the runtime output shape. Rank must match Compile.
func (k *Kernel) Resize(shape Shape) error {
	if k == nil || shape.Rank() != k.shape.Rank() {
		return ErrShape
	}
	if k.shape.Equal(shape) {
		return nil
	}
	if err := shape.check(); err != nil {
		return err
	}
	copy(k.shape, shape)
	k.size = shape.Size()
	return nil
}

// Shape is the logical output shape.
func (k *Kernel) Shape() Shape {
	if k == nil {
		return nil
	}
	return k.shape.Clone()
}

// Bindings is 1 (output) plus the input buffer count.
func (k *Kernel) Bindings() int {
	if k == nil {
		return 0
	}
	return 1 + len(k.bufs)
}

// Size is the dense output length.
func (k *Kernel) Size() int {
	if k == nil {
		return 0
	}
	return k.size
}

// InputCount is the number of leaf buffers.
func (k *Kernel) InputCount() int {
	if k == nil {
		return 0
	}
	return len(k.bufs)
}

// Input is leaf i's host storage, or nil.
func (k *Kernel) Input(i int) []byte {
	if k == nil || i < 0 || i >= len(k.bufs) || k.bufs[i] == nil {
		return nil
	}
	return k.bufs[i].raw
}

// InputDType is leaf i's element type, or 0.
func (k *Kernel) InputDType(i int) DType {
	if k == nil || i < 0 || i >= len(k.bufs) || k.bufs[i] == nil {
		return 0
	}
	return k.bufs[i].dtype
}

// DType is the output element type.
func (k *Kernel) DType() DType {
	if k == nil {
		return 0
	}
	return k.outType
}

// OutputBytes is Size times the output element width.
func (k *Kernel) OutputBytes() int {
	if k == nil {
		return 0
	}
	return k.size * k.outType.size()
}

// Groups is the compute workgroup count for Size.
func (k *Kernel) Groups() uint32 {
	if k == nil || k.size == 0 {
		return 0
	}
	return uint32((k.size + localSize - 1) / localSize)
}

// Push is n and the first four shape dims, little-endian.
func (k *Kernel) Push() []byte {
	push := make([]byte, PushBytes)
	k.FillPush(push)
	return push
}

func (k *Kernel) FillPush(push []byte) {
	if len(push) < PushBytes {
		return
	}
	clear(push[:PushBytes])
	if k == nil {
		return
	}
	binary.LittleEndian.PutUint32(push[0:], uint32(k.size))
	for i, s := range k.shape {
		if i >= 4 {
			break
		}
		binary.LittleEndian.PutUint32(push[4+4*i:], uint32(s))
	}
}

func needsRewrite(n *node, seen map[*node]bool) bool {
	if n == nil || seen[n] {
		return false
	}
	seen[n] = true
	if n.viewed() {
		return true
	}
	for _, s := range n.sources {
		if needsRewrite(s, seen) {
			return true
		}
	}
	return false
}

type rewriteMemo struct {
	n *node
	h uint64
}

func rewrite(n *node, steps []stStep, memo map[rewriteMemo]*node) *node {
	if n == nil || n.err != nil {
		return n
	}
	key := rewriteMemo{n: n, h: stepsHash(steps)}
	if m := memo[key]; m != nil {
		return m
	}
	switch n.kind {
	case kindInput:
		if len(steps) == 0 {
			memo[key] = n
			return n
		}
		out := *n
		out.chain = append(slices.Clone(steps), stStep{tr: n.tracker})
		memo[key] = &out
		return &out
	case kindConst:
		if len(n.tracker.views) == 0 || len(steps) == 0 {
			memo[key] = n
			return n
		}
		out := *n
		out.chain = append(slices.Clone(steps), stStep{tr: n.tracker})
		memo[key] = &out
		return &out
	case kindCoord:
		if len(steps) == 0 && !n.viewed() {
			memo[key] = n
			return n
		}
		out := *n
		out.chain = append(slices.Clone(steps), stStep{tr: n.tracker, origin: n.origin})
		memo[key] = &out
		return &out
	case kindOp:
		next := steps
		if n.viewed() {
			next = append(slices.Clone(steps), stStep{tr: n.tracker, origin: n.origin})
		}
		srcs := make([]*node, len(n.sources))
		same := !n.viewed() && len(steps) == 0
		for i, s := range n.sources {
			srcs[i] = rewrite(s, next, memo)
			if srcs[i] != s {
				same = false
			}
		}
		if same {
			memo[key] = n
			return n
		}
		out := *n
		out.sources = srcs
		out.chain = nil
		memo[key] = &out
		return &out
	default:
		return n
	}
}

func stepsHash(steps []stStep) uint64 {
	h := uint64(14695981039346656037)
	for _, s := range steps {
		h = hashTracker(h, s.tr)
		h = hashInts(h, s.origin)
	}
	return h
}

func hashTracker(h uint64, tracker Tracker) uint64 {
	for _, v := range tracker.views {
		h = hashInts(h, v.shape)
		h = hashInts(h, v.strides)
		h ^= uint64(v.offset) + 0x9e3779b97f4a7c15
		h *= 1099511628211
		for _, m := range v.mask {
			h = hashInts(h, m[:])
		}
	}
	return h
}

func hashInts(h uint64, xs []int) uint64 {
	for _, x := range xs {
		h ^= uint64(x) + 0x9e3779b97f4a7c15
		h *= 1099511628211
	}
	h ^= uint64(len(xs))
	h *= 1099511628211
	return h
}

func flatten(root *node) ([]*node, []*buffer, error) {
	seen := map[*node]bool{}
	var order []*node
	var bufs []*buffer
	idx := map[*buffer]int{}
	var walk func(*node) error
	walk = func(n *node) error {
		if n == nil {
			return ErrOp
		}
		if n.err != nil {
			return n.err
		}
		if seen[n] {
			return nil
		}
		seen[n] = true
		switch n.kind {
		case kindConst:
		case kindCoord:
			if n.slot < 0 || n.tracker.check() != nil {
				return ErrOp
			}
		case kindInput:
			if n.buf == nil {
				return ErrOp
			}
			if _, ok := idx[n.buf]; !ok {
				idx[n.buf] = len(bufs)
				bufs = append(bufs, n.buf)
			}
		case kindOp:
			if n.op.arity() != len(n.sources) {
				return fmt.Errorf("%w: %s", ErrOp, n.op)
			}
			for _, s := range n.sources {
				if err := walk(s); err != nil {
					return err
				}
			}
		default:
			return ErrOp
		}
		order = append(order, n)
		return nil
	}
	if err := walk(root); err != nil {
		return nil, nil, err
	}
	return order, bufs, nil
}

func cseKey(n *node) string {
	switch len(n.sources) {
	case 1:
		return fmt.Sprintf("%d/%d/%p", n.op, n.dtype, n.sources[0])
	case 2:
		return fmt.Sprintf("%d/%d/%p/%p", n.op, n.dtype, n.sources[0], n.sources[1])
	default:
		return fmt.Sprintf("%d/%d/%p/%p/%p", n.op, n.dtype, n.sources[0], n.sources[1], n.sources[2])
	}
}

func foldConstOp(n *node) *node {
	if n == nil || n.kind != kindOp || n.viewed() || len(n.chain) > 0 {
		return nil
	}
	for _, s := range n.sources {
		if s == nil || s.kind != kindConst || len(s.tracker.views) != 0 || len(s.chain) > 0 {
			return nil
		}
	}
	var a, b, c uint32
	inType := n.dtype
	if len(n.sources) > 0 {
		a = n.sources[0].bits
		inType = n.sources[0].dtype
	}
	if len(n.sources) > 1 {
		b = n.sources[1].bits
	}
	if len(n.sources) > 2 {
		c = n.sources[2].bits
	}
	bits := instruction{alu: n.op, dtype: n.dtype, inType: inType, a: 0, b: 1, c: 2}.evalALU([]uint32{a, b, c})
	return internConst(n.dtype, bits)
}

func identityOp(n *node) *node {
	if n == nil || n.kind != kindOp || n.viewed() {
		return n
	}
	switch n.op {
	case ADD:
		if n.dtype == F32 {
			return n
		}
		if isZeroConst(n.sources[1]) {
			return n.sources[0]
		}
		if isZeroConst(n.sources[0]) {
			return n.sources[1]
		}
	case MUL:
		if isOneConst(n.sources[1]) {
			return n.sources[0]
		}
		if isOneConst(n.sources[0]) {
			return n.sources[1]
		}
	case MAX:
		if n.sources[0] == n.sources[1] {
			return n.sources[0]
		}
	case WHERE:
		if isZeroConst(n.sources[0]) {
			return n.sources[2]
		}
		if isOneConst(n.sources[0]) {
			return n.sources[1]
		}
	}
	return n
}

func isZeroConst(n *node) bool {
	return n != nil && n.kind == kindConst && len(n.tracker.views) == 0 && len(n.chain) == 0 && n.bits == 0
}

func isOneConst(n *node) bool {
	if n == nil || n.kind != kindConst || len(n.tracker.views) != 0 || len(n.chain) > 0 {
		return false
	}
	switch n.dtype {
	case F32:
		return n.bits == math.Float32bits(1)
	default:
		return n.bits == 1
	}
}
