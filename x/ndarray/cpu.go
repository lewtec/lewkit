package ndarray

import (
	"math"
	"runtime"
	"slices"
	"sync"
)

const cpuMinParallel = 1024

const (
	cpuConst uint8 = iota
	cpuCoord
	cpuLoad
	cpuALU
)

type instruction struct {
	kind          uint8
	alu           Op
	dst, a, b, c  int
	dtype, inType DType
	bits          uint32
	source        int
	axis          int
	dense         bool
	scalar        bool
	i32           bool
	views         []view
}

type cpuProgram struct {
	code      []instruction
	registers int
	root      int
}

func lowerCPU(order []*Node, slots, shape []int) (cpuProgram, error) {
	slotIndex := make(map[int]int, len(slots))
	for i, s := range slots {
		slotIndex[s] = i
	}
	reg := make(map[*Node]int, len(order))
	code := make([]instruction, 0, len(order))
	for _, n := range order {
		dst := len(code)
		reg[n] = dst
		in := instruction{dst: dst, dtype: n.dtype}
		switch n.kind {
		case kindConst:
			in.kind = cpuConst
			in.bits = n.bits
		case kindCoord:
			if n.slot < 0 || n.slot >= len(shape) {
				return cpuProgram{}, ErrAxis
			}
			in.kind = cpuCoord
			in.axis = n.slot
		case kindInput:
			in.kind = cpuLoad
			in.source = slotIndex[n.slot]
			in.i32 = n.dtype == I32
			in.scalar = len(n.tracker.Shape()) == 0
			in.dense = !in.scalar && n.tracker.Contiguous() && slices.Equal(n.tracker.Shape(), shape)
			if !in.dense && !in.scalar {
				in.views = n.tracker.views
			}
		case kindOp:
			in.kind = cpuALU
			in.alu = n.op
			in.inType = n.sources[0].dtype
			in.a = reg[n.sources[0]]
			if len(n.sources) > 1 {
				in.b = reg[n.sources[1]]
			}
			if len(n.sources) > 2 {
				in.c = reg[n.sources[2]]
			}
		default:
			return cpuProgram{}, ErrOp
		}
		code = append(code, in)
	}
	return cpuProgram{code: code, registers: len(code), root: reg[order[len(order)-1]]}, nil
}

type cpuJob struct {
	program cpuProgram
	shape   []int
	outType DType
	inputs  [][]float32
	output  []float32
	lo, hi  int
}

type cpuScratch struct {
	registers []uint32
	coords    []int
	scratch   []int
}

var scratchPool = sync.Pool{New: func() any { return &cpuScratch{scratch: make([]int, 8)} }}

func takeScratch(registers, rank int) *cpuScratch {
	s := scratchPool.Get().(*cpuScratch)
	if cap(s.registers) < registers {
		s.registers = make([]uint32, registers)
	} else {
		s.registers = s.registers[:registers]
	}
	if cap(s.coords) < rank {
		s.coords = make([]int, rank)
	} else {
		s.coords = s.coords[:rank]
	}
	return s
}

func (k *Kernel) evalCPU(output []float32, inputs [][]float32) {
	workers := min(runtime.GOMAXPROCS(0), k.n)
	if workers < 2 || k.n < cpuMinParallel {
		k.evalSerial(output, inputs)
		return
	}
	k.evalParallel(output, inputs, workers)
}

func (k *Kernel) evalSerial(output []float32, inputs [][]float32) {
	cpuJob{program: k.cpu, shape: k.shape, outType: k.outType, inputs: inputs, output: output, hi: k.n}.run()
}

var (
	cpuOnce  sync.Once
	cpuReady []chan cpuJob
	cpuDone  []chan struct{}
)

func startCPUWorkers() {
	cpuOnce.Do(func() {
		n := runtime.GOMAXPROCS(0)
		if n < 1 {
			n = 1
		}
		cpuReady = make([]chan cpuJob, n)
		cpuDone = make([]chan struct{}, n)
		for i := range n {
			cpuReady[i] = make(chan cpuJob)
			cpuDone[i] = make(chan struct{})
			go cpuWorker(i)
		}
	})
}

func cpuWorker(id int) {
	var s cpuScratch
	for j := range cpuReady[id] {
		if cap(s.registers) < j.program.registers {
			s.registers = make([]uint32, j.program.registers)
		} else {
			s.registers = s.registers[:j.program.registers]
		}
		rank := len(j.shape)
		if cap(s.coords) < rank {
			s.coords = make([]int, rank)
		} else {
			s.coords = s.coords[:rank]
		}
		if cap(s.scratch) < 8 {
			s.scratch = make([]int, 8)
		}
		j.loop(&s)
		cpuDone[id] <- struct{}{}
	}
}

func (k *Kernel) evalParallel(output []float32, inputs [][]float32, workers int) {
	startCPUWorkers()
	if workers > len(cpuReady) {
		workers = len(cpuReady)
	}
	chunk := (k.n + workers - 1) / workers
	base := cpuJob{program: k.cpu, shape: k.shape, outType: k.outType, inputs: inputs, output: output}
	n := 0
	for w := range workers {
		lo := w * chunk
		hi := min(lo+chunk, k.n)
		if lo >= hi {
			break
		}
		j := base
		j.lo, j.hi = lo, hi
		cpuReady[w] <- j
		n++
	}
	for w := range n {
		<-cpuDone[w]
	}
}

func (j cpuJob) run() {
	s := takeScratch(j.program.registers, len(j.shape))
	j.loop(s)
	scratchPool.Put(s)
}

func (j cpuJob) loop(s *cpuScratch) {
	for i := j.lo; i < j.hi; i++ {
		unravelInto(j.shape, i, s.coords)
		for _, instr := range j.program.code {
			switch instr.kind {
			case cpuConst:
				s.registers[instr.dst] = instr.bits
			case cpuCoord:
				s.registers[instr.dst] = uint32(int32(s.coords[instr.axis]))
			case cpuLoad:
				s.registers[instr.dst] = instr.load(i, s.coords, j.inputs, &s.scratch)
			case cpuALU:
				s.registers[instr.dst] = instr.evalALU(s.registers)
			}
		}
		if j.outType == I32 {
			j.output[i] = float32(int32(s.registers[j.program.root]))
		} else {
			j.output[i] = math.Float32frombits(s.registers[j.program.root])
		}
	}
}

func (instr instruction) load(i int, coords []int, inputs [][]float32, scratch *[]int) uint32 {
	var off int
	ok := true
	switch {
	case instr.scalar:
		off = 0
	case instr.dense:
		off = i
	default:
		off, ok = indexViews(instr.views, coords, scratch)
	}
	if !ok || instr.source < 0 || instr.source >= len(inputs) || off < 0 || off >= len(inputs[instr.source]) {
		return 0
	}
	x := inputs[instr.source][off]
	if instr.i32 {
		return uint32(int32(x))
	}
	return math.Float32bits(x)
}

func indexViews(views []view, coords []int, scratch *[]int) (int, bool) {
	if len(views) == 0 {
		return 0, false
	}
	off, ok := views[len(views)-1].index(coords)
	for i := len(views) - 2; i >= 0; i-- {
		v := views[i]
		need := len(v.shape)
		if cap(*scratch) < need {
			*scratch = make([]int, need)
		}
		mid := (*scratch)[:need]
		unravelInto(v.shape, off, mid)
		var viewOK bool
		off, viewOK = v.index(mid)
		ok = ok && viewOK
	}
	return off, ok
}

func unravelInto(shape []int, i int, coords []int) {
	acc := 1
	for d := len(shape) - 1; d >= 0; d-- {
		s := shape[d]
		if s == 0 {
			coords[d] = 0
			continue
		}
		coords[d] = (i / acc) % s
		acc *= s
	}
}

func (instr instruction) evalALU(regs []uint32) uint32 {
	a, b, c := regs[instr.a], uint32(0), uint32(0)
	if instr.alu.arity() > 1 {
		b = regs[instr.b]
	}
	if instr.alu.arity() > 2 {
		c = regs[instr.c]
	}
	fa := math.Float32frombits(a)
	fb := math.Float32frombits(b)
	fc := math.Float32frombits(c)
	ia, ib, ic := int32(a), int32(b), int32(c)
	asFloat := instr.inType == F32
	packFloat := math.Float32bits
	packInt := func(v int32) uint32 { return uint32(v) }
	switch instr.alu {
	case EXP2:
		return packFloat(float32(math.Exp2(float64(fa))))
	case LOG2:
		return packFloat(float32(math.Log2(float64(fa))))
	case SIN:
		return packFloat(float32(math.Sin(float64(fa))))
	case SQRT:
		return packFloat(float32(math.Sqrt(float64(fa))))
	case RECIP:
		return packFloat(1 / fa)
	case NEG:
		if asFloat {
			return packFloat(-fa)
		}
		return packInt(-ia)
	case CAST:
		if instr.dtype == I32 {
			if asFloat {
				return packInt(int32(fa))
			}
			return a
		}
		if asFloat {
			return a
		}
		return packFloat(float32(ia))
	case ADD:
		if asFloat {
			return packFloat(fa + fb)
		}
		return packInt(ia + ib)
	case MUL:
		if asFloat {
			return packFloat(fa * fb)
		}
		return packInt(ia * ib)
	case IDIV:
		if ib == 0 {
			return 0
		}
		return packInt(ia / ib)
	case MAX:
		if asFloat {
			return packFloat(max(fa, fb))
		}
		return packInt(max(ia, ib))
	case MOD:
		if ib == 0 {
			return 0
		}
		return packInt(ia % ib)
	case CMPLT:
		var t int32
		if asFloat {
			if fa < fb {
				t = 1
			}
		} else if ia < ib {
			t = 1
		}
		return packInt(t)
	case CMPNE:
		var t int32
		if asFloat {
			if fa != fb {
				t = 1
			}
		} else if ia != ib {
			t = 1
		}
		return packInt(t)
	case XOR:
		return packInt(ia ^ ib)
	case SHL:
		return packInt(ia << uint32(ib))
	case SHR:
		return packInt(ia >> uint32(ib))
	case OR:
		return packInt(ia | ib)
	case AND:
		return packInt(ia & ib)
	case WHERE:
		if ia != 0 {
			return b
		}
		return c
	case MULACC:
		if asFloat {
			return packFloat(fa*fb + fc)
		}
		return packInt(ia*ib + ic)
	default:
		return 0
	}
}
