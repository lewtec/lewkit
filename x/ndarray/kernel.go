package ndarray

import (
	"encoding/binary"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

const (
	localSize = 256
	PushBytes = 20 // n, d0, d1, d2, d3
)

// Kernel is the flattened graph: nodes, leaf buffers, compile-time shape.
// Backends lower their own code from this (CPU tape, GLSL). Resize updates
// the runtime shape only.
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
	expr = simplify(expr)
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

// GLSL lowers the compute shader. The kernel does not keep the source.
func (k *Kernel) GLSL() (string, error) {
	if k == nil {
		return "", ErrOp
	}
	w := &glslWriter{
		root:     k.root,
		order:    k.order,
		bufs:     k.bufs,
		outShape: k.built,
		bufBind:  make(map[*buffer]int, len(k.bufs)),
		names:    make(map[*node]string, len(k.order)),
	}
	return w.program()
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

func simplify(n *node) *node {
	return cseFold(n, map[*node]*node{}, map[string]*node{})
}

func cseFold(n *node, memo map[*node]*node, cse map[string]*node) *node {
	if n == nil {
		return n
	}
	if m := memo[n]; m != nil {
		return m
	}
	if n.kind != kindOp {
		memo[n] = n
		return n
	}
	srcs := make([]*node, len(n.sources))
	same := true
	for i, s := range n.sources {
		srcs[i] = cseFold(s, memo, cse)
		if srcs[i] != s {
			same = false
		}
	}
	out := n
	if !same {
		clone := *n
		clone.sources = srcs
		out = &clone
	}
	if folded := foldConstOp(out); folded != nil {
		memo[n] = folded
		return folded
	}
	out = identityOp(out)
	if out.kind != kindOp {
		memo[n] = out
		return out
	}
	key := cseKey(out)
	if hit := cse[key]; hit != nil {
		memo[n] = hit
		return hit
	}
	cse[key] = out
	memo[n] = out
	return out
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
	if n == nil || n.kind != kindOp {
		return n
	}
	switch n.op {
	case ADD:
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

type glslWriter struct {
	b        strings.Builder
	next     int
	coords   []string
	outShape Shape
	bufBind  map[*buffer]int
	names    map[*node]string
	root     *node
	order    []*node
	bufs     []*buffer
}

func (w *glslWriter) program() (string, error) {
	for i, b := range w.bufs {
		w.bufBind[b] = i + 1
	}
	w.b.WriteString("#version 450\n")
	w.b.WriteString("layout(local_size_x = ")
	w.b.WriteString(strconv.Itoa(localSize))
	w.b.WriteString(") in;\n")
	w.b.WriteString("layout(push_constant) uniform Push { uint n; uint d0; uint d1; uint d2; uint d3; };\n")
	w.b.WriteString("layout(set = 0, binding = 0) buffer Out { float o[]; };\n")
	for i, b := range w.bufs {
		dtype := F32
		if b != nil {
			dtype = b.dtype
		}
		fmt.Fprintf(&w.b, "layout(set = 0, binding = %d) buffer In%d { %s x%d[]; };\n", i+1, i+1, dtype.glsl(), i+1)
	}
	w.b.WriteString("void main() {\n")
	w.b.WriteString("    uint gi = gl_GlobalInvocationID.x;\n")
	w.b.WriteString("    if (gi >= n) return;\n")
	w.b.WriteString("    int i = int(gi);\n")
	w.coords = w.unravelPush(len(w.outShape))
	for _, n := range w.order {
		name, err := w.node(n)
		if err != nil {
			return "", err
		}
		w.names[n] = name
	}
	out := w.names[w.root]
	if w.root.dtype != F32 {
		out = "float(" + out + ")"
	}
	if w.root.viewed() {
		_, valid := w.indexAt(w.root.tracker, w.coords)
		fmt.Fprintf(&w.b, "    o[i] = (%s) ? %s : 0.0;\n}\n", valid, out)
	} else {
		fmt.Fprintf(&w.b, "    o[i] = %s;\n}\n", out)
	}
	return w.b.String(), nil
}

func (w *glslWriter) name(prefix string) string {
	w.next++
	return prefix + strconv.Itoa(w.next)
}

func (w *glslWriter) node(n *node) (string, error) {
	id := w.name("t")
	switch n.kind {
	case kindConst:
		valid := "true"
		if len(n.chain) > 0 {
			_, valid = w.indexChain(n.chain)
		} else if n.viewed() {
			_, valid = w.index(n.tracker)
		}
		if valid != "true" {
			zero := "0.0"
			switch n.dtype {
			case I32:
				zero = "0"
			case U8:
				zero = "0u"
			}
			fmt.Fprintf(&w.b, "    %s %s = %s;\n", n.dtype.glsl(), id, zero)
			fmt.Fprintf(&w.b, "    if (%s) %s = %s;\n", valid, id, glslConst(n))
			return id, nil
		}
		fmt.Fprintf(&w.b, "    %s %s = %s;\n", n.dtype.glsl(), id, glslConst(n))
		return id, nil
	case kindCoord:
		coords := w.coords
		if len(n.chain) > 0 {
			var err error
			coords, err = w.chainCoords(n.chain)
			if err != nil {
				return "", err
			}
		}
		if n.slot < 0 || n.slot >= len(coords) {
			return "", ErrAxis
		}
		fmt.Fprintf(&w.b, "    int %s = %s;\n", id, coords[n.slot])
		return id, nil
	case kindInput:
		off, valid := "0", "true"
		if len(n.chain) > 0 {
			off, valid = w.indexChain(n.chain)
		} else if len(n.tracker.Shape()) == 0 {
			at, ok, err := n.tracker.At(0)
			if err != nil || !ok {
				return "", ErrIndex
			}
			off = strconv.Itoa(at)
		} else {
			off, valid = w.index(n.tracker)
		}
		zero := "0.0"
		switch n.dtype {
		case I32:
			zero = "0"
		case U8:
			zero = "0u"
		}
		fmt.Fprintf(&w.b, "    %s %s = %s;\n", n.dtype.glsl(), id, zero)
		fmt.Fprintf(&w.b, "    if (%s) %s = x%d[%s];\n", valid, id, w.bufBind[n.buf], off)
		return id, nil
	case kindOp:
		args := make([]string, len(n.sources))
		for i, s := range n.sources {
			args[i] = w.names[s]
		}
		fmt.Fprintf(&w.b, "    %s %s = %s;\n", n.dtype.glsl(), id, n.glslALU(args))
		return id, nil
	default:
		return "", ErrOp
	}
}

func (w *glslWriter) index(tracker Tracker) (off, valid string) {
	return w.indexAt(tracker, w.coords)
}

func (w *glslWriter) indexAt(tracker Tracker, coords []string) (off, valid string) {
	if len(tracker.views) == 0 {
		return "0", "true"
	}
	if tracker.Contiguous() && tracker.Shape().Equal(w.outShape) && sameCoordVars(coords, w.coords) {
		return "i", "true"
	}
	off, valid = w.view(tracker.views[len(tracker.views)-1], coords)
	for i := len(tracker.views) - 2; i >= 0; i-- {
		v := tracker.views[i]
		mid := w.unravel(v.shape, off)
		off2, val2 := w.view(v, mid)
		both := w.name("ok")
		fmt.Fprintf(&w.b, "    bool %s = (%s) && (%s);\n", both, valid, val2)
		off, valid = off2, both
	}
	return off, valid
}

func (w *glslWriter) indexChain(chain []stStep) (off, valid string) {
	coords := w.coords
	valid = "true"
	for i, s := range chain {
		o, v := w.indexAt(s.tr, coords)
		both := w.name("ok")
		fmt.Fprintf(&w.b, "    bool %s = (%s) && (%s);\n", both, valid, v)
		valid = both
		if i == len(chain)-1 {
			return o, valid
		}
		if len(s.origin) == 0 {
			return o, valid
		}
		coords = w.unravel(s.origin, o)
	}
	return "0", valid
}

func (w *glslWriter) chainCoords(chain []stStep) ([]string, error) {
	coords := w.coords
	valid := "true"
	for _, s := range chain {
		o, v := w.indexAt(s.tr, coords)
		both := w.name("ok")
		fmt.Fprintf(&w.b, "    bool %s = (%s) && (%s);\n", both, valid, v)
		valid = both
		if len(s.origin) == 0 {
			return coords, nil
		}
		coords = w.unravel(s.origin, o)
	}
	return coords, nil
}

func sameCoordVars(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (w *glslWriter) view(v view, coords []string) (off, valid string) {
	off = w.name("p")
	fmt.Fprintf(&w.b, "    int %s = %d;\n", off, v.offset)
	for i, c := range coords {
		if v.strides[i] == 0 {
			continue
		}
		fmt.Fprintf(&w.b, "    %s += %s * (%d);\n", off, c, v.strides[i])
	}
	valid = "true"
	if v.mask != nil {
		valid = w.name("m")
		fmt.Fprintf(&w.b, "    bool %s = true;\n", valid)
		for i, c := range coords {
			fmt.Fprintf(&w.b, "    %s = %s && %s >= %d && %s < %d;\n", valid, valid, c, v.mask[i][0], c, v.mask[i][1])
		}
	}
	return off, valid
}

func (w *glslWriter) unravelPush(rank int) []string {
	if rank == 0 {
		return nil
	}
	coords := make([]string, rank)
	rem := "i"
	for d := rank - 1; d >= 0; d-- {
		dim := "d" + strconv.Itoa(d)
		c := w.name("c")
		fmt.Fprintf(&w.b, "    int %s = (%s <= 1u) ? 0 : (%s) %% int(%s);\n", c, dim, rem, dim)
		coords[d] = c
		if d > 0 {
			nxt := w.name("u")
			fmt.Fprintf(&w.b, "    int %s = (%s <= 1u) ? (%s) : (%s) / int(%s);\n", nxt, dim, rem, rem, dim)
			rem = nxt
		}
	}
	return coords
}

func (w *glslWriter) unravel(shape Shape, idx string) []string {
	if len(shape) == 0 {
		return nil
	}
	coords := make([]string, len(shape))
	rem := idx
	for d := len(shape) - 1; d >= 0; d-- {
		s := shape[d]
		if s <= 1 {
			coords[d] = "0"
			continue
		}
		c := w.name("c")
		fmt.Fprintf(&w.b, "    int %s = (%s) %% %d;\n", c, rem, s)
		coords[d] = c
		if d > 0 {
			nxt := w.name("u")
			fmt.Fprintf(&w.b, "    int %s = (%s) / %d;\n", nxt, rem, s)
			rem = nxt
		}
	}
	return coords
}

func glslConst(n *node) string {
	if n.dtype == I32 {
		return strconv.FormatInt(int64(int32(n.bits)), 10)
	}
	if n.dtype == U8 {
		return strconv.FormatUint(uint64(uint8(n.bits)), 10) + "u"
	}
	s := strconv.FormatFloat(float64(math.Float32frombits(n.bits)), 'g', -1, 32)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}

func (n *node) glslALU(args []string) string {
	switch n.op {
	case EXP2:
		return "exp2(" + args[0] + ")"
	case LOG2:
		return "log2(" + args[0] + ")"
	case SIN:
		return "sin(" + args[0] + ")"
	case SQRT:
		return "sqrt(" + args[0] + ")"
	case RECIP:
		return "(1.0/" + args[0] + ")"
	case NEG:
		return "(-" + args[0] + ")"
	case CAST:
		arg := args[0]
		from := F32
		if len(n.sources) > 0 {
			from = n.sources[0].dtype
		}
		switch n.dtype {
		case U8:
			if from == F32 {
				return "uint(clamp(" + arg + ", 0.0, 255.0))"
			}
			return "uint(clamp(" + arg + ", 0, 255))"
		default:
			return n.dtype.glsl() + "(" + arg + ")"
		}
	case ADD:
		return "(" + args[0] + "+" + args[1] + ")"
	case MUL:
		return "(" + args[0] + "*" + args[1] + ")"
	case IDIV:
		return "(" + args[0] + "/" + args[1] + ")"
	case MAX:
		return "max(" + args[0] + "," + args[1] + ")"
	case MOD:
		return "(" + args[0] + "%" + args[1] + ")"
	case CMPLT:
		return "int(" + args[0] + "<" + args[1] + ")"
	case CMPNE:
		return "int(" + args[0] + "!=" + args[1] + ")"
	case XOR:
		return "(" + args[0] + "^" + args[1] + ")"
	case SHL:
		return "(" + args[0] + "<<" + args[1] + ")"
	case SHR:
		return "(" + args[0] + ">>" + args[1] + ")"
	case OR:
		return "(" + args[0] + "|" + args[1] + ")"
	case AND:
		return "(" + args[0] + "&" + args[1] + ")"
	case WHERE:
		return "((" + args[0] + "!=0)?" + args[1] + ":" + args[2] + ")"
	case MultiplyAccumulate:
		return "(" + args[0] + "*" + args[1] + "+" + args[2] + ")"
	default:
		return "0"
	}
}
