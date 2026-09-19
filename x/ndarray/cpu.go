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
	slotIdx := make(map[int]int, len(slots))
	for i, s := range slots {
		slotIdx[s] = i
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
		case kindIn:
			in.kind = cpuLoad
			in.source = slotIdx[n.slot]
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
	p       cpuProgram
	shape   []int
	outType DType
	srcs    [][]float32
	out     []float32
	lo, hi  int
}

type cpuScratch struct {
	regs    []uint32
	coords  []int
	scratch []int
}

var scratchPool = sync.Pool{New: func() any { return &cpuScratch{scratch: make([]int, 8)} }}

func takeScratch(registers, rank int) *cpuScratch {
	s := scratchPool.Get().(*cpuScratch)
	if cap(s.regs) < registers {
		s.regs = make([]uint32, registers)
	} else {
		s.regs = s.regs[:registers]
	}
	if cap(s.coords) < rank {
		s.coords = make([]int, rank)
	} else {
		s.coords = s.coords[:rank]
	}
	return s
}

func (k *Kernel) evalCPU(dst []float32, srcs [][]float32) {
	workers := min(runtime.GOMAXPROCS(0), k.n)
	if workers < 2 || k.n < cpuMinParallel {
		k.evalSerial(dst, srcs)
		return
	}
	k.evalParallel(dst, srcs, workers)
}

func (k *Kernel) evalSerial(dst []float32, srcs [][]float32) {
	cpuJob{p: k.cpu, shape: k.shape, outType: k.outType, srcs: srcs, out: dst, hi: k.n}.run()
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
		if cap(s.regs) < j.p.registers {
			s.regs = make([]uint32, j.p.registers)
		} else {
			s.regs = s.regs[:j.p.registers]
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

func (k *Kernel) evalParallel(dst []float32, srcs [][]float32, workers int) {
	startCPUWorkers()
	if workers > len(cpuReady) {
		workers = len(cpuReady)
	}
	chunk := (k.n + workers - 1) / workers
	base := cpuJob{p: k.cpu, shape: k.shape, outType: k.outType, srcs: srcs, out: dst}
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
	s := takeScratch(j.p.registers, len(j.shape))
	j.loop(s)
	scratchPool.Put(s)
}

func (j cpuJob) loop(s *cpuScratch) {
	for i := j.lo; i < j.hi; i++ {
		unravelInto(j.shape, i, s.coords)
		for _, in := range j.p.code {
			switch in.kind {
			case cpuConst:
				s.regs[in.dst] = in.bits
			case cpuCoord:
				s.regs[in.dst] = uint32(int32(s.coords[in.axis]))
			case cpuLoad:
				s.regs[in.dst] = in.load(i, s.coords, j.srcs, &s.scratch)
			case cpuALU:
				s.regs[in.dst] = in.evalALU(s.regs)
			}
		}
		if j.outType == I32 {
			j.out[i] = float32(int32(s.regs[j.p.root]))
		} else {
			j.out[i] = math.Float32frombits(s.regs[j.p.root])
		}
	}
}

func (in instruction) load(i int, coords []int, srcs [][]float32, scratch *[]int) uint32 {
	var off int
	ok := true
	switch {
	case in.scalar:
		off = 0
	case in.dense:
		off = i
	default:
		off, ok = indexViews(in.views, coords, scratch)
	}
	if !ok || in.source < 0 || in.source >= len(srcs) || off < 0 || off >= len(srcs[in.source]) {
		return 0
	}
	x := srcs[in.source][off]
	if in.i32 {
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
		var vok bool
		off, vok = v.index(mid)
		ok = ok && vok
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

func (in instruction) evalALU(regs []uint32) uint32 {
	a, b, c := regs[in.a], uint32(0), uint32(0)
	if in.alu.arity() > 1 {
		b = regs[in.b]
	}
	if in.alu.arity() > 2 {
		c = regs[in.c]
	}
	fa := math.Float32frombits(a)
	fb := math.Float32frombits(b)
	fc := math.Float32frombits(c)
	ia, ib, ic := int32(a), int32(b), int32(c)
	asF := in.inType == F32
	packF := math.Float32bits
	packI := func(v int32) uint32 { return uint32(v) }
	switch in.alu {
	case EXP2:
		return packF(float32(math.Exp2(float64(fa))))
	case LOG2:
		return packF(float32(math.Log2(float64(fa))))
	case SIN:
		return packF(float32(math.Sin(float64(fa))))
	case SQRT:
		return packF(float32(math.Sqrt(float64(fa))))
	case RECIP:
		return packF(1 / fa)
	case NEG:
		if asF {
			return packF(-fa)
		}
		return packI(-ia)
	case CAST:
		if in.dtype == I32 {
			if asF {
				return packI(int32(fa))
			}
			return a
		}
		if asF {
			return a
		}
		return packF(float32(ia))
	case ADD:
		if asF {
			return packF(fa + fb)
		}
		return packI(ia + ib)
	case MUL:
		if asF {
			return packF(fa * fb)
		}
		return packI(ia * ib)
	case IDIV:
		if ib == 0 {
			return 0
		}
		return packI(ia / ib)
	case MAX:
		if asF {
			return packF(max(fa, fb))
		}
		return packI(max(ia, ib))
	case MOD:
		if ib == 0 {
			return 0
		}
		return packI(ia % ib)
	case CMPLT:
		var t int32
		if asF {
			if fa < fb {
				t = 1
			}
		} else if ia < ib {
			t = 1
		}
		return packI(t)
	case CMPNE:
		var t int32
		if asF {
			if fa != fb {
				t = 1
			}
		} else if ia != ib {
			t = 1
		}
		return packI(t)
	case XOR:
		return packI(ia ^ ib)
	case SHL:
		return packI(ia << uint32(ib))
	case SHR:
		return packI(ia >> uint32(ib))
	case OR:
		return packI(ia | ib)
	case AND:
		return packI(ia & ib)
	case WHERE:
		if ia != 0 {
			return b
		}
		return c
	case MULACC:
		if asF {
			return packF(fa*fb + fc)
		}
		return packI(ia*ib + ic)
	default:
		return 0
	}
}
