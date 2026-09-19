package ndarray

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/ffi/vulkan"
)

const (
	localSize = 256
	pushBytes = 20 // n, d0, d1, d2, d3
)

// Kernel is one fused compute shader for a whole expression.
type Kernel struct {
	root           *Node
	glsl           string
	shape          []int
	slots          []int
	outType        DType
	n              int
	spirv          []byte
	cpu            cpuProgram
	pipeline       *vulkan.Shader
	pipelineDevice *vulkan.Device
	runBuffers     []*vulkan.Buffer
}

// Compile lowers expr to one GLSL compute kernel. One dispatch covers every cell.
func Compile(expr *Node) (*Kernel, error) {
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
	n := prod(shape)
	if n < 0 {
		return nil, ErrShape
	}
	order, slots, err := flatten(expr)
	if err != nil {
		return nil, err
	}
	w := &glslWriter{
		root:     expr,
		order:    order,
		slots:    slots,
		outShape: shape,
		slotBind: make(map[int]int, len(slots)),
		ssa:      make(map[*Node]string, len(order)),
	}
	src, err := w.program()
	if err != nil {
		return nil, err
	}
	cpu, err := lowerCPU(order, slots, shape)
	if err != nil {
		return nil, err
	}
	return &Kernel{root: expr, glsl: src, shape: slices.Clone(shape), slots: slots, outType: expr.dtype, n: n, cpu: cpu}, nil
}

// Resize sets the runtime output shape. Rank must match Compile.
func (k *Kernel) Resize(shape []int) error {
	if k == nil || len(shape) != len(k.shape) {
		return ErrShape
	}
	n := 1
	for _, s := range shape {
		if s < 0 {
			return ErrShape
		}
		n *= s
	}
	k.shape = slices.Clone(shape)
	k.n = n
	return nil
}

// GLSL is the compute shader source.
func (k *Kernel) GLSL() string {
	if k == nil {
		return ""
	}
	return k.glsl
}

// Shape is the logical output shape.
func (k *Kernel) Shape() []int {
	if k == nil {
		return nil
	}
	return slices.Clone(k.shape)
}

// Bindings is 1 (output) plus the input buffer count.
func (k *Kernel) Bindings() int {
	if k == nil {
		return 0
	}
	return 1 + len(k.slots)
}

// Slots is the input slot for each source buffer after the output binding.
func (k *Kernel) Slots() []int {
	if k == nil {
		return nil
	}
	return slices.Clone(k.slots)
}

func flatten(root *Node) ([]*Node, []int, error) {
	seen := map[*Node]bool{}
	var order []*Node
	slotSet := map[int]bool{}
	var walk func(*Node) error
	walk = func(n *Node) error {
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
			if n.slot < 0 {
				return ErrOp
			}
			slotSet[n.slot] = true
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
	slots := make([]int, 0, len(slotSet))
	for s := range slotSet {
		slots = append(slots, s)
	}
	slices.Sort(slots)
	return order, slots, nil
}

type glslWriter struct {
	b        strings.Builder
	next     int
	coords   []string
	outShape []int
	slotBind map[int]int
	ssa      map[*Node]string
	root     *Node
	order    []*Node
	slots    []int
}

func (w *glslWriter) program() (string, error) {
	for i, s := range w.slots {
		w.slotBind[s] = i + 1
	}
	w.b.WriteString("#version 450\n")
	w.b.WriteString("layout(local_size_x = ")
	w.b.WriteString(strconv.Itoa(localSize))
	w.b.WriteString(") in;\n")
	w.b.WriteString("layout(push_constant) uniform Push { uint n; uint d0; uint d1; uint d2; uint d3; };\n")
	fmt.Fprintf(&w.b, "layout(set = 0, binding = 0) buffer Out { %s o[]; };\n", w.root.dtype.glsl())
	for i, s := range w.slots {
		dtype := F32
		for _, n := range w.order {
			if n.kind == kindInput && n.slot == s {
				dtype = n.dtype
				break
			}
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
		w.ssa[n] = name
	}
	fmt.Fprintf(&w.b, "    o[i] = %s;\n}\n", w.ssa[w.root])
	return w.b.String(), nil
}

func (w *glslWriter) name(prefix string) string {
	w.next++
	return prefix + strconv.Itoa(w.next)
}

func (w *glslWriter) node(n *Node) (string, error) {
	id := w.name("t")
	switch n.kind {
	case kindConst:
		fmt.Fprintf(&w.b, "    %s %s = %s;\n", n.dtype.glsl(), id, glslConst(n))
		return id, nil
	case kindCoord:
		if n.slot < 0 || n.slot >= len(w.coords) || !slices.Equal(n.tracker.Shape(), w.outShape) {
			return "", ErrAxis
		}
		fmt.Fprintf(&w.b, "    int %s = %s;\n", id, w.coords[n.slot])
		return id, nil
	case kindInput:
		off, valid := "0", "true"
		if len(n.tracker.Shape()) != 0 {
			off, valid = w.index(n.tracker)
		}
		zero := "0.0"
		if n.dtype == I32 {
			zero = "0"
		}
		fmt.Fprintf(&w.b, "    %s %s = %s;\n", n.dtype.glsl(), id, zero)
		fmt.Fprintf(&w.b, "    if (%s) %s = x%d[%s];\n", valid, id, w.slotBind[n.slot], off)
		return id, nil
	case kindOp:
		args := make([]string, len(n.sources))
		for i, s := range n.sources {
			args[i] = w.ssa[s]
		}
		fmt.Fprintf(&w.b, "    %s %s = %s;\n", n.dtype.glsl(), id, n.glslALU(args))
		return id, nil
	default:
		return "", ErrOp
	}
}

func (w *glslWriter) index(tracker Tracker) (off, valid string) {
	if tracker.Contiguous() && slices.Equal(tracker.Shape(), w.outShape) {
		return "i", "true"
	}
	off, valid = w.view(tracker.views[len(tracker.views)-1], w.coords)
	for i := len(tracker.views) - 2; i >= 0; i-- {
		v := tracker.views[i]
		coords := w.unravel(v.shape, off)
		off2, val2 := w.view(v, coords)
		both := w.name("ok")
		fmt.Fprintf(&w.b, "    bool %s = (%s) && (%s);\n", both, valid, val2)
		off, valid = off2, both
	}
	return off, valid
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

func (w *glslWriter) unravel(shape []int, idx string) []string {
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

func glslConst(n *Node) string {
	if n.dtype == I32 {
		return strconv.FormatInt(int64(int32(n.bits)), 10)
	}
	s := strconv.FormatFloat(float64(math.Float32frombits(n.bits)), 'g', -1, 32)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}

func (n *Node) glslALU(args []string) string {
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
		return n.dtype.glsl() + "(" + args[0] + ")"
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
	case MULACC:
		return "(" + args[0] + "*" + args[1] + "+" + args[2] + ")"
	default:
		return "0"
	}
}
