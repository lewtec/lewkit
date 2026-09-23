package ndarray

import (
	"encoding/binary"
	"math"
	"runtime"
	"sync"
)

const cpuMinParallel = 1024

const (
	cpuConst uint8 = iota
	cpuCoord
	cpuLoad
	cpuALU
	cpuGather
	cpuIndex
	cpuCarry
)

type instruction struct {
	kind          uint8
	alu           Op
	dest, a, b, c int
	dtype, inType DType
	bits          uint32
	source        int
	axis          int
	dense         bool
	scalar        bool
	splatOff      int
	views         []view
	chain         []stStep
}

type cpuProgram struct {
	code      []instruction
	registers int
	root      int
	loopCount int
	loopBegin int
	loopEnd   int
	loopInit  int
	loopIndex int
	loopCarry int
	loopBody  int
	loopOut   int
	loopChain []stStep
}

func lowerCPU(order []*node, bufs []*buffer, shape Shape) (cpuProgram, error) {
	for _, n := range order {
		if n.kind == kindLoop {
			return lowerLoop(order, bufs, shape)
		}
	}
	bufIndex := make(map[*buffer]int, len(bufs))
	for i, b := range bufs {
		bufIndex[b] = i
	}
	reg := make(map[*node]int, len(order))
	code := make([]instruction, 0, len(order))
	for _, n := range order {
		dest := len(code)
		reg[n] = dest
		instr := instruction{dest: dest, dtype: n.dtype}
		switch n.kind {
		case kindConst:
			instr.kind = cpuConst
			instr.bits = n.bits
			if len(n.chain) > 0 {
				instr.chain = n.chain
			} else if n.viewed() {
				instr.views = n.tracker.views
			}
		case kindCoord:
			if n.slot < 0 || n.slot >= len(shape) {
				return cpuProgram{}, ErrAxis
			}
			instr.kind = cpuCoord
			instr.axis = n.slot
			if len(n.chain) > 0 {
				instr.chain = n.chain
			}
		case kindInput:
			instr.kind = cpuLoad
			instr.source = bufIndex[n.buf]
			if len(n.chain) > 0 {
				instr.chain = n.chain
			} else {
				instr.scalar = len(n.tracker.Shape()) == 0
				instr.dense = !instr.scalar && n.tracker.Contiguous() && n.tracker.Shape().Equal(shape)
				if instr.scalar {
					off, ok, err := n.tracker.At(0)
					if err != nil || !ok {
						return cpuProgram{}, ErrIndex
					}
					instr.splatOff = off
				}
				if !instr.dense && !instr.scalar {
					instr.views = n.tracker.views
				}
			}
		case kindOp:
			instr.kind = cpuALU
			instr.alu = n.op
			instr.inType = n.sources[0].dtype
			instr.a = reg[n.sources[0]]
			if len(n.sources) > 1 {
				instr.b = reg[n.sources[1]]
			}
			if len(n.sources) > 2 {
				instr.c = reg[n.sources[2]]
			}
		case kindGather:
			instr.kind = cpuGather
			instr.source = bufIndex[n.buf]
			instr.a = reg[n.sources[0]]
		default:
			return cpuProgram{}, ErrOp
		}
		code = append(code, instr)
	}
	return cpuProgram{code: code, registers: len(code), root: reg[order[len(order)-1]]}, nil
}

func lowerLoop(order []*node, bufs []*buffer, shape Shape) (cpuProgram, error) {
	loop, prefix, inside, suffix, err := splitLoop(order)
	if err != nil || loop == nil {
		return cpuProgram{}, err
	}
	bufIndex := make(map[*buffer]int, len(bufs))
	for i, b := range bufs {
		bufIndex[b] = i
	}
	reg := make(map[*node]int, len(order))
	for i, n := range order {
		reg[n] = i
	}
	encode := func(n *node) (instruction, error) {
		instr := instruction{dest: reg[n], dtype: n.dtype}
		switch n.kind {
		case kindConst:
			instr.kind = cpuConst
			instr.bits = n.bits
			if len(n.chain) > 0 {
				instr.chain = n.chain
			} else if n.viewed() {
				instr.views = n.tracker.views
			}
		case kindCoord:
			if n.slot < 0 || n.slot >= len(shape) {
				return instruction{}, ErrAxis
			}
			instr.kind = cpuCoord
			instr.axis = n.slot
			if len(n.chain) > 0 {
				instr.chain = n.chain
			}
		case kindInput:
			instr.kind = cpuLoad
			instr.source = bufIndex[n.buf]
			if len(n.chain) > 0 {
				instr.chain = n.chain
			} else {
				instr.scalar = len(n.tracker.Shape()) == 0
				instr.dense = !instr.scalar && n.tracker.Contiguous() && n.tracker.Shape().Equal(shape)
				if instr.scalar {
					off, ok, err := n.tracker.At(0)
					if err != nil || !ok {
						return instruction{}, ErrIndex
					}
					instr.splatOff = off
				}
				if !instr.dense && !instr.scalar {
					instr.views = n.tracker.views
				}
			}
		case kindOp:
			instr.kind = cpuALU
			instr.alu = n.op
			instr.inType = n.sources[0].dtype
			instr.a = reg[n.sources[0]]
			if len(n.sources) > 1 {
				instr.b = reg[n.sources[1]]
			}
			if len(n.sources) > 2 {
				instr.c = reg[n.sources[2]]
			}
		case kindGather:
			instr.kind = cpuGather
			instr.source = bufIndex[n.buf]
			instr.a = reg[n.sources[0]]
		case kindLoopIndex:
			instr.kind = cpuIndex
		case kindCarry:
			instr.kind = cpuCarry
		default:
			return instruction{}, ErrOp
		}
		return instr, nil
	}
	var code []instruction
	push := func(nodes []*node) error {
		for _, n := range nodes {
			instr, err := encode(n)
			if err != nil {
				return err
			}
			code = append(code, instr)
		}
		return nil
	}
	if err := push(prefix); err != nil {
		return cpuProgram{}, err
	}
	begin := len(code)
	if err := push(inside); err != nil {
		return cpuProgram{}, err
	}
	end := len(code)
	if err := push(suffix); err != nil {
		return cpuProgram{}, err
	}
	indexNode, carryNode := loopEnds(inside)
	if indexNode == nil || carryNode == nil {
		return cpuProgram{}, ErrOp
	}
	return cpuProgram{
		code:      code,
		registers: len(order),
		root:      reg[order[len(order)-1]],
		loopCount: loop.slot,
		loopBegin: begin,
		loopEnd:   end,
		loopInit:  reg[loop.sources[0]],
		loopIndex: reg[indexNode],
		loopCarry: reg[carryNode],
		loopBody:  reg[loop.sources[1]],
		loopOut:   reg[loop],
		loopChain: loop.chain,
	}, nil
}

func loopEnds(inside []*node) (index, carry *node) {
	for _, n := range inside {
		switch n.kind {
		case kindLoopIndex:
			index = n
		case kindCarry:
			carry = n
		}
	}
	return index, carry
}

type cpuJob struct {
	program  cpuProgram
	kernel   *Kernel
	shape    Shape
	outType  DType
	bufs     []*buffer
	output   []byte
	lo, hi   int
	maskRoot bool
}

type cpuScratch struct {
	registers []uint32
	coords    []int
	scratch   []int
	loopStep  int
	index     int
	carry     uint32
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

func runCPU(program cpuProgram, k *Kernel, output []byte) {
	workers := min(runtime.GOMAXPROCS(0), k.size)
	if workers < 2 || k.size < cpuMinParallel {
		cpuJob{program: program, kernel: k, shape: k.shape, outType: k.outType, bufs: k.bufs, output: output, hi: k.size, maskRoot: k.root.viewed()}.run()
		return
	}
	evalParallel(program, k, output, workers)
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

func evalParallel(program cpuProgram, k *Kernel, output []byte, workers int) {
	startCPUWorkers()
	if workers > len(cpuReady) {
		workers = len(cpuReady)
	}
	chunk := (k.size + workers - 1) / workers
	base := cpuJob{program: program, kernel: k, shape: k.shape, outType: k.outType, bufs: k.bufs, output: output, maskRoot: k.root.viewed()}
	n := 0
	for w := range workers {
		lo := w * chunk
		hi := min(lo+chunk, k.size)
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
		s.index = i
		unravelInto(j.shape, i, s.coords)
		if j.program.loopCount > 0 {
			j.exec(s, j.program.code[:j.program.loopBegin])
			acc := s.registers[j.program.loopInit]
			for step := 0; step < j.program.loopCount; step++ {
				s.loopStep = step
				s.carry = acc
				j.exec(s, j.program.code[j.program.loopBegin:j.program.loopEnd])
				acc = s.registers[j.program.loopBody]
			}
			s.registers[j.program.loopOut] = acc
			if len(j.program.loopChain) > 0 {
				if _, ok := applyChain(j.program.loopChain, s.coords, &s.scratch); !ok {
					s.registers[j.program.loopOut] = 0
				}
			}
			j.exec(s, j.program.code[j.program.loopEnd:])
		} else {
			j.exec(s, j.program.code)
		}
		root := s.registers[j.program.root]
		if j.maskRoot {
			if _, ok := indexViews(j.kernel.root.tracker.views, s.coords, &s.scratch); !ok {
				root = 0
			}
		}
		switch j.outType {
		case U8:
			j.output[i] = uint8(root)
		default:
			binary.LittleEndian.PutUint32(j.output[i*4:], root)
		}
	}
}

func (j cpuJob) exec(s *cpuScratch, code []instruction) {
	for _, instr := range code {
		switch instr.kind {
		case cpuConst:
			if len(instr.chain) > 0 {
				if _, ok := applyChain(instr.chain, s.coords, &s.scratch); !ok {
					s.registers[instr.dest] = 0
					continue
				}
			} else if len(instr.views) > 0 {
				if _, ok := indexViews(instr.views, s.coords, &s.scratch); !ok {
					s.registers[instr.dest] = 0
					continue
				}
			}
			s.registers[instr.dest] = instr.bits
		case cpuCoord:
			coords := s.coords
			if len(instr.chain) > 0 {
				var ok bool
				coords, ok = chainCoords(instr.chain, s.coords, &s.scratch)
				if !ok {
					s.registers[instr.dest] = 0
					continue
				}
			}
			if instr.axis < 0 || instr.axis >= len(coords) {
				s.registers[instr.dest] = 0
				continue
			}
			s.registers[instr.dest] = uint32(int32(coords[instr.axis]))
		case cpuLoad:
			s.registers[instr.dest] = instr.load(s.index, s.coords, j.bufs, &s.scratch)
		case cpuGather:
			off := int32(s.registers[instr.a])
			if instr.source < 0 || instr.source >= len(j.bufs) {
				s.registers[instr.dest] = 0
				continue
			}
			s.registers[instr.dest] = j.bufs[instr.source].word(int(off))
		case cpuIndex:
			s.registers[instr.dest] = uint32(int32(s.loopStep))
		case cpuCarry:
			s.registers[instr.dest] = s.carry
		case cpuALU:
			s.registers[instr.dest] = instr.evalALU(s.registers)
		}
	}
}

func applyChain(chain []stStep, coords []int, scratch *[]int) (int, bool) {
	c := coords
	ok := true
	var off int
	for i, s := range chain {
		var vok bool
		off, vok = indexViews(s.tr.views, c, scratch)
		ok = ok && vok
		if i == len(chain)-1 {
			return off, ok
		}
		if len(s.origin) == 0 {
			return off, ok
		}
		need := len(s.origin)
		if cap(*scratch) < need {
			*scratch = make([]int, need)
		}
		mid := make([]int, need)
		unravelInto(s.origin, off, mid)
		c = mid
	}
	return 0, false
}

func chainCoords(chain []stStep, coords []int, scratch *[]int) ([]int, bool) {
	c := coords
	ok := true
	for _, s := range chain {
		off, vok := indexViews(s.tr.views, c, scratch)
		ok = ok && vok
		if len(s.origin) == 0 {
			return c, ok
		}
		mid := make([]int, len(s.origin))
		unravelInto(s.origin, off, mid)
		c = mid
	}
	return c, ok
}

func (instr instruction) load(i int, coords []int, bufs []*buffer, scratch *[]int) uint32 {
	var off int
	ok := true
	switch {
	case len(instr.chain) > 0:
		off, ok = applyChain(instr.chain, coords, scratch)
	case instr.scalar:
		off = instr.splatOff
	case instr.dense:
		off = i
	default:
		off, ok = indexViews(instr.views, coords, scratch)
	}
	if !ok || instr.source < 0 || instr.source >= len(bufs) || bufs[instr.source] == nil {
		return 0
	}
	return bufs[instr.source].word(off)
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

func unravelInto(shape Shape, i int, coords []int) {
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
	packU8 := func(v int32) uint32 {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return uint32(v)
	}
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
		switch instr.dtype {
		case U8:
			if asFloat {
				return packU8(int32(fa))
			}
			if instr.inType == U8 {
				return a
			}
			return packU8(ia)
		case I32:
			if asFloat {
				return packInt(int32(fa))
			}
			if instr.inType == U8 {
				return packInt(int32(uint8(a)))
			}
			return a
		default:
			if asFloat {
				return a
			}
			if instr.inType == U8 {
				return packFloat(float32(uint8(a)))
			}
			return packFloat(float32(ia))
		}
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
	case MultiplyAccumulate:
		if asFloat {
			return packFloat(fa*fb + fc)
		}
		return packInt(ia*ib + ic)
	default:
		return 0
	}
}
