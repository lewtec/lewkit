package ndarray

import (
	"math"
	"runtime"
	"slices"
	"sync"
)

const cpuMinPar = 1024

const (
	ckConst uint8 = iota
	ckCoord
	ckLoad
	ckALU
)

type inst struct {
	kind         uint8
	alu          Op
	dst, a, b, c int
	dt, inDT     DType
	bits         uint32
	buf          int
	axis         int
	dense        bool
	scalar       bool
	i32          bool
	views        []view
}

type cpuProg struct {
	code []inst
	nreg int
	root int
}

func lowerCPU(order []*Node, slots, shape []int) (cpuProg, error) {
	slotIdx := make(map[int]int, len(slots))
	for i, s := range slots {
		slotIdx[s] = i
	}
	reg := make(map[*Node]int, len(order))
	code := make([]inst, 0, len(order))
	for _, n := range order {
		dst := len(code)
		reg[n] = dst
		in := inst{dst: dst, dt: n.dt}
		switch n.kind {
		case kindConst:
			in.kind = ckConst
			in.bits = n.bits
		case kindCoord:
			if n.slot < 0 || n.slot >= len(shape) {
				return cpuProg{}, ErrAxis
			}
			in.kind = ckCoord
			in.axis = n.slot
		case kindIn:
			in.kind = ckLoad
			in.buf = slotIdx[n.slot]
			in.i32 = n.dt == I32
			in.scalar = len(n.st.Shape()) == 0
			in.dense = !in.scalar && n.st.Contiguous() && slices.Equal(n.st.Shape(), shape)
			if !in.dense && !in.scalar {
				in.views = n.st.views
			}
		case kindOp:
			in.kind = ckALU
			in.alu = n.op
			in.inDT = n.srcs[0].dt
			in.a = reg[n.srcs[0]]
			if len(n.srcs) > 1 {
				in.b = reg[n.srcs[1]]
			}
			if len(n.srcs) > 2 {
				in.c = reg[n.srcs[2]]
			}
		default:
			return cpuProg{}, ErrOp
		}
		code = append(code, in)
	}
	return cpuProg{code: code, nreg: len(code), root: reg[order[len(order)-1]]}, nil
}

type cpuJob struct {
	p      cpuProg
	shape  []int
	outDT  DType
	srcs   [][]float32
	out    []float32
	lo, hi int
	done   func()
}

type cpuScratch struct {
	regs    []uint32
	coords  []int
	scratch []int
}

var scratchPool = sync.Pool{New: func() any { return &cpuScratch{scratch: make([]int, 8)} }}

func takeScratch(nreg, rank int) *cpuScratch {
	s := scratchPool.Get().(*cpuScratch)
	if cap(s.regs) < nreg {
		s.regs = make([]uint32, nreg)
	} else {
		s.regs = s.regs[:nreg]
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
	if workers < 2 || k.n < cpuMinPar {
		k.evalSerial(dst, srcs)
		return
	}
	k.evalParallel(dst, srcs, workers)
}

func (k *Kernel) evalSerial(dst []float32, srcs [][]float32) {
	cpuJob{p: k.cpu, shape: k.shape, outDT: k.outDT, srcs: srcs, out: dst, hi: k.n}.run()
}

var (
	cpuOnce sync.Once
	cpuCh   chan cpuJob
)

func startCPUWorkers() {
	cpuOnce.Do(func() {
		n := runtime.GOMAXPROCS(0)
		if n < 1 {
			n = 1
		}
		cpuCh = make(chan cpuJob, n)
		for range n {
			go cpuWorker()
		}
	})
}

func cpuWorker() {
	var s cpuScratch
	for j := range cpuCh {
		if cap(s.regs) < j.p.nreg {
			s.regs = make([]uint32, j.p.nreg)
		} else {
			s.regs = s.regs[:j.p.nreg]
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
		j.done()
	}
}

func (k *Kernel) evalParallel(dst []float32, srcs [][]float32, workers int) {
	startCPUWorkers()
	var wg sync.WaitGroup
	chunk := (k.n + workers - 1) / workers
	base := cpuJob{p: k.cpu, shape: k.shape, outDT: k.outDT, srcs: srcs, out: dst}
	for w := range workers {
		lo := w * chunk
		hi := min(lo+chunk, k.n)
		if lo >= hi {
			break
		}
		wg.Add(1)
		j := base
		j.lo, j.hi, j.done = lo, hi, wg.Done
		cpuCh <- j
	}
	wg.Wait()
}

func (j cpuJob) run() {
	s := takeScratch(j.p.nreg, len(j.shape))
	j.loop(s)
	scratchPool.Put(s)
}

func (j cpuJob) loop(s *cpuScratch) {
	for i := j.lo; i < j.hi; i++ {
		unravelInto(j.shape, i, s.coords)
		for _, in := range j.p.code {
			switch in.kind {
			case ckConst:
				s.regs[in.dst] = in.bits
			case ckCoord:
				s.regs[in.dst] = uint32(int32(s.coords[in.axis]))
			case ckLoad:
				s.regs[in.dst] = in.load(i, s.coords, j.srcs, &s.scratch)
			case ckALU:
				s.regs[in.dst] = in.evalALU(s.regs)
			}
		}
		if j.outDT == I32 {
			j.out[i] = float32(int32(s.regs[j.p.root]))
		} else {
			j.out[i] = math.Float32frombits(s.regs[j.p.root])
		}
	}
}

func (in inst) load(i int, coords []int, srcs [][]float32, scratch *[]int) uint32 {
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
	if !ok || in.buf < 0 || in.buf >= len(srcs) || off < 0 || off >= len(srcs[in.buf]) {
		return 0
	}
	x := srcs[in.buf][off]
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

func (in inst) evalALU(regs []uint32) uint32 {
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
	asF := in.inDT == F32
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
		if in.dt == I32 {
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
