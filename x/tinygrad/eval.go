package tinygrad

import (
	"fmt"
	"math"
)

// Eval runs the kernel on the CPU. srcs[i] is the buffer for Slots()[i].
func (k *Kernel) Eval(srcs ...[]float32) ([]float32, error) {
	if k == nil || k.root == nil {
		return nil, ErrOp
	}
	if len(srcs) != len(k.slots) {
		return nil, fmt.Errorf("%w: want %d inputs, got %d", ErrOp, len(k.slots), len(srcs))
	}
	if k.n == 0 {
		return nil, nil
	}
	c := cpu{srcs: srcs, slotIdx: make(map[int]int, len(k.slots))}
	for i, s := range k.slots {
		c.slotIdx[s] = i
	}
	out := make([]float32, k.n)
	for i := range k.n {
		c.coords = unravel(k.shape, i)
		bits, err := c.node(k.root)
		if err != nil {
			return nil, err
		}
		if k.outDT == I32 {
			out[i] = float32(int32(bits))
		} else {
			out[i] = math.Float32frombits(bits)
		}
	}
	return out, nil
}

type cpu struct {
	srcs    [][]float32
	slotIdx map[int]int
	coords  []int
}

func (c cpu) node(n *Node) (uint32, error) {
	if n == nil {
		return 0, ErrOp
	}
	if n.err != nil {
		return 0, n.err
	}
	switch n.kind {
	case kindConst:
		return n.bits, nil
	case kindIn:
		off, ok, err := n.st.Index(c.coords...)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, nil
		}
		si := c.slotIdx[n.slot]
		if si < 0 || si >= len(c.srcs) || off < 0 || off >= len(c.srcs[si]) {
			return 0, fmt.Errorf("%w: load slot %d off %d", ErrIndex, n.slot, off)
		}
		x := c.srcs[si][off]
		if n.dt == I32 {
			return uint32(int32(x)), nil
		}
		return math.Float32bits(x), nil
	case kindOp:
		in := make([]uint32, len(n.srcs))
		for i, s := range n.srcs {
			v, err := c.node(s)
			if err != nil {
				return 0, err
			}
			in[i] = v
		}
		return n.apply(in)
	default:
		return 0, ErrOp
	}
}

func (n *Node) apply(args []uint32) (uint32, error) {
	f := func(i int) float32 { return math.Float32frombits(args[i]) }
	iv := func(i int) int32 { return int32(args[i]) }
	packF := func(v float32) uint32 { return math.Float32bits(v) }
	packI := func(v int32) uint32 { return uint32(v) }
	asF := n.srcs[0].dt == F32
	switch n.op {
	case EXP2:
		return packF(float32(math.Exp2(float64(f(0))))), nil
	case LOG2:
		return packF(float32(math.Log2(float64(f(0))))), nil
	case SIN:
		return packF(float32(math.Sin(float64(f(0))))), nil
	case SQRT:
		return packF(float32(math.Sqrt(float64(f(0))))), nil
	case RECIP:
		return packF(1 / f(0)), nil
	case NEG:
		if asF {
			return packF(-f(0)), nil
		}
		return packI(-iv(0)), nil
	case CAST:
		if n.dt == I32 {
			if asF {
				return packI(int32(f(0))), nil
			}
			return args[0], nil
		}
		if asF {
			return args[0], nil
		}
		return packF(float32(iv(0))), nil
	case ADD:
		if asF {
			return packF(f(0) + f(1)), nil
		}
		return packI(iv(0) + iv(1)), nil
	case MUL:
		if asF {
			return packF(f(0) * f(1)), nil
		}
		return packI(iv(0) * iv(1)), nil
	case IDIV:
		if iv(1) == 0 {
			return 0, nil
		}
		return packI(iv(0) / iv(1)), nil
	case MAX:
		if asF {
			return packF(max(f(0), f(1))), nil
		}
		return packI(max(iv(0), iv(1))), nil
	case MOD:
		if iv(1) == 0 {
			return 0, nil
		}
		return packI(iv(0) % iv(1)), nil
	case CMPLT:
		var c int32
		if asF {
			if f(0) < f(1) {
				c = 1
			}
		} else if iv(0) < iv(1) {
			c = 1
		}
		return packI(c), nil
	case CMPNE:
		var c int32
		if asF {
			if f(0) != f(1) {
				c = 1
			}
		} else if iv(0) != iv(1) {
			c = 1
		}
		return packI(c), nil
	case XOR:
		return packI(iv(0) ^ iv(1)), nil
	case SHL:
		return packI(iv(0) << uint32(iv(1))), nil
	case SHR:
		return packI(iv(0) >> uint32(iv(1))), nil
	case OR:
		return packI(iv(0) | iv(1)), nil
	case AND:
		return packI(iv(0) & iv(1)), nil
	case WHERE:
		if iv(0) != 0 {
			return args[1], nil
		}
		return args[2], nil
	case MULACC:
		if asF {
			return packF(f(0)*f(1) + f(2)), nil
		}
		return packI(iv(0)*iv(1) + iv(2)), nil
	default:
		return 0, ErrOp
	}
}
