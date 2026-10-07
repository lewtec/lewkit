package ndarray

import (
	"fmt"
	"math"
	"strconv"
)

// Code is the scheduled body of one kernel.
// Threads is the 1D workgroup size. Out is binding 0.
// In is the element type of each input, in binding order.
// A backend defines gi, n, and d0..d3, then renders Stmts.
// The body defines i. This package does not spell a shading language.
type Code struct {
	Threads int
	Out     DType
	In      []DType
	Stmts   []Stmt
}

// StmtOp is one scheduled step.
type StmtOp uint8

const (
	StmtDecl StmtOp = iota
	StmtSet
	StmtIfSet
	StmtAdd
	StmtStore
	StmtReturn
)

// Stmt is one step of the schedule.
// StmtDecl binds Name. Bool is a predicate; otherwise DType is the value type.
// StmtSet replaces Name with RHS. StmtIfSet does that when Cond holds.
// StmtAdd adds RHS into Name.
// StmtStore writes RHS to output element i.
// StmtReturn ends the invocation when Cond holds.
type Stmt struct {
	Op    StmtOp
	DType DType
	Bool  bool
	Name  string
	RHS   Expr
	Cond  Expr
}

// ExprOp is one scheduled value.
type ExprOp uint8

const (
	ExprRef ExprOp = iota
	ExprInt
	ExprUint
	ExprTrue
	ExprConst
	ExprLoad
	ExprNeg
	ExprAdd
	ExprMul
	ExprDiv
	ExprMod
	ExprBitAnd
	ExprBitOr
	ExprBitXor
	ExprShl
	ExprShr
	ExprRel
	ExprAnd
	ExprSel
	ExprCast
	ExprFn
)

// Rel is a predicate over two values.
type Rel uint8

const (
	RelLT Rel = iota + 1
	RelLE
	RelGT
	RelGE
	RelEQ
	RelNE
)

// Fn is a math function the schedule calls by meaning.
type Fn uint8

const (
	FnExp2 Fn = iota + 1
	FnLog2
	FnSin
	FnSqrt
	FnMax
	FnClamp
)

// Expr is one value in the schedule.
// ExprRef is Name. ExprInt and ExprUint are immediates. ExprTrue is true.
// ExprConst is DType and Bits. ExprLoad reads Name at Args[0].
// ExprNeg through ExprShr are raw arithmetic on Args.
// ExprRel compares Args under Rel. ExprAnd is logical and.
// ExprSel is Args[0], then Args[1], else Args[2].
// ExprCast converts Args[0] to DType. ExprFn calls Fn.
type Expr struct {
	Op    ExprOp
	DType DType
	Rel   Rel
	Fn    Fn
	Name  string
	Int   int64
	Bits  uint32
	Args  []Expr
}

// Code schedules the kernel. Rank above 4 is ErrShape.
func (k *Kernel) Code() (Code, error) {
	if k == nil || k.root == nil {
		return Code{}, ErrOp
	}
	if len(k.built) > maxPushRank {
		return Code{}, fmt.Errorf("%w: rank %d", ErrShape, len(k.built))
	}
	w := &codeWriter{
		root:     k.root,
		order:    k.order,
		bufs:     k.bufs,
		outShape: k.built,
		bufBind:  make(map[*buffer]int, len(k.bufs)),
		names:    make(map[*node]string, len(k.order)),
	}
	if err := w.schedule(); err != nil {
		return Code{}, err
	}
	in := make([]DType, len(k.bufs))
	for i, b := range k.bufs {
		in[i] = F32
		if b != nil {
			in[i] = b.dtype
		}
	}
	return Code{Threads: localSize, Out: k.root.dtype, In: in, Stmts: w.body}, nil
}

type codeWriter struct {
	next     int
	coords   []string
	outShape Shape
	bufBind  map[*buffer]int
	names    map[*node]string
	root     *node
	order    []*node
	bufs     []*buffer
	body     []Stmt
}

func (w *codeWriter) emit(s Stmt) { w.body = append(w.body, s) }

func (w *codeWriter) name(prefix string) string {
	w.next++
	return prefix + strconv.Itoa(w.next)
}

func (w *codeWriter) schedule() error {
	for i, b := range w.bufs {
		w.bufBind[b] = i + 1
	}
	w.emit(Stmt{Op: StmtReturn, Cond: rel(RelGE, ref("gi"), ref("n"))})
	w.emit(decl(I32, "i", cast(I32, ref("gi"))))
	w.coords = w.unravelPush(len(w.outShape))
	for _, n := range w.order {
		name, err := w.node(n)
		if err != nil {
			return err
		}
		w.names[n] = name
	}
	out := w.names[w.root]
	if w.root.viewed() {
		_, valid := w.indexAt(w.root.tracker, w.coords)
		w.emit(Stmt{Op: StmtStore, RHS: sel(asBool(valid), ref(out), zeroExpr(w.root.dtype))})
		return nil
	}
	w.emit(Stmt{Op: StmtStore, RHS: ref(out)})
	return nil
}

func (w *codeWriter) node(n *node) (string, error) {
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
			w.emit(decl(n.dtype, id, zeroExpr(n.dtype)))
			w.emit(Stmt{Op: StmtIfSet, Cond: ref(valid), Name: id, RHS: constExpr(n)})
			return id, nil
		}
		w.emit(decl(n.dtype, id, constExpr(n)))
		return id, nil
	case kindCoord:
		coords := w.coords
		valid := "true"
		if len(n.chain) > 0 {
			var err error
			coords, valid, err = w.chainCoords(n.chain)
			if err != nil {
				return "", err
			}
		}
		if n.slot < 0 || n.slot >= len(coords) {
			return "", ErrAxis
		}
		if valid == "true" {
			w.emit(decl(I32, id, refOrInt(coords[n.slot])))
			return id, nil
		}
		w.emit(decl(I32, id, iconst(0)))
		w.emit(Stmt{Op: StmtIfSet, Cond: ref(valid), Name: id, RHS: refOrInt(coords[n.slot])})
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
		cells := 0
		if n.buf != nil {
			cells = n.buf.cells()
		}
		w.emit(decl(n.dtype, id, zeroExpr(n.dtype)))
		cond := land(asBool(valid), land(
			rel(RelGE, refOrInt(off), iconst(0)),
			rel(RelLT, refOrInt(off), iconst(cells)),
		))
		w.emit(Stmt{
			Op:   StmtIfSet,
			Cond: cond,
			Name: id,
			RHS:  load("x"+strconv.Itoa(w.bufBind[n.buf]), refOrInt(off)),
		})
		return id, nil
	case kindOp:
		args := make([]string, len(n.sources))
		for i, s := range n.sources {
			args[i] = w.names[s]
		}
		w.emit(decl(n.dtype, id, n.exprALU(args)))
		return id, nil
	default:
		return "", ErrOp
	}
}

func (w *codeWriter) index(tracker Tracker) (off, valid string) {
	return w.indexAt(tracker, w.coords)
}

func (w *codeWriter) indexAt(tracker Tracker, coords []string) (off, valid string) {
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
		w.emit(declBool(both, land(asBool(valid), asBool(val2))))
		off, valid = off2, both
	}
	return off, valid
}

func (w *codeWriter) indexChain(chain []stStep) (off, valid string) {
	coords := w.coords
	valid = "true"
	for i, s := range chain {
		o, v := w.indexAt(s.tr, coords)
		both := w.name("ok")
		w.emit(declBool(both, land(asBool(valid), asBool(v))))
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

func (w *codeWriter) chainCoords(chain []stStep) ([]string, string, error) {
	coords := w.coords
	valid := "true"
	for _, s := range chain {
		o, v := w.indexAt(s.tr, coords)
		both := w.name("ok")
		w.emit(declBool(both, land(asBool(valid), asBool(v))))
		valid = both
		if len(s.origin) == 0 {
			return coords, valid, nil
		}
		coords = w.unravel(s.origin, o)
	}
	return coords, valid, nil
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

func (w *codeWriter) view(v view, coords []string) (off, valid string) {
	off = w.name("p")
	w.emit(decl(I32, off, iconst(v.offset)))
	if len(v.shape) == 0 {
		return off, "true"
	}
	for i, c := range coords {
		if i >= len(v.strides) || v.strides[i] == 0 {
			continue
		}
		w.emit(Stmt{Op: StmtAdd, Name: off, RHS: mul(refOrInt(c), iconst(v.strides[i]))})
	}
	valid = "true"
	if v.mask != nil {
		valid = w.name("m")
		w.emit(declBool(valid, Expr{Op: ExprTrue}))
		for i, c := range coords {
			if i >= len(v.mask) {
				break
			}
			rhs := land(ref(valid), rel(RelGE, refOrInt(c), iconst(v.mask[i][0])))
			rhs = land(rhs, rel(RelLT, refOrInt(c), iconst(v.mask[i][1])))
			w.emit(Stmt{Op: StmtSet, Name: valid, RHS: rhs})
		}
	}
	return off, valid
}

func (w *codeWriter) unravelPush(rank int) []string {
	if rank == 0 {
		return nil
	}
	coords := make([]string, rank)
	rem := "i"
	for d := rank - 1; d >= 0; d-- {
		dim := ref("d" + strconv.Itoa(d))
		c := w.name("c")
		cond := rel(RelLE, dim, uintc(1))
		w.emit(decl(I32, c, sel(cond, iconst(0), mod(ref(rem), cast(I32, dim)))))
		coords[d] = c
		if d > 0 {
			nxt := w.name("u")
			w.emit(decl(I32, nxt, sel(cond, ref(rem), div(ref(rem), cast(I32, dim)))))
			rem = nxt
		}
	}
	return coords
}

func (w *codeWriter) unravel(shape Shape, idx string) []string {
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
		w.emit(decl(I32, c, mod(ref(rem), iconst(s))))
		coords[d] = c
		if d > 0 {
			nxt := w.name("u")
			w.emit(decl(I32, nxt, div(ref(rem), iconst(s))))
			rem = nxt
		}
	}
	return coords
}

func decl(d DType, name string, rhs Expr) Stmt {
	return Stmt{Op: StmtDecl, DType: d, Name: name, RHS: rhs}
}

func declBool(name string, rhs Expr) Stmt {
	return Stmt{Op: StmtDecl, Bool: true, Name: name, RHS: rhs}
}

func ref(name string) Expr { return Expr{Op: ExprRef, Name: name} }

func iconst(v int) Expr { return Expr{Op: ExprInt, Int: int64(v)} }

func uintc(v int) Expr { return Expr{Op: ExprUint, Int: int64(v)} }

func refOrInt(s string) Expr {
	if s == "" {
		return iconst(0)
	}
	if s[0] == '-' || (s[0] >= '0' && s[0] <= '9') {
		if n, err := strconv.Atoi(s); err == nil {
			return iconst(n)
		}
	}
	return ref(s)
}

func asBool(s string) Expr {
	if s == "true" {
		return Expr{Op: ExprTrue}
	}
	return ref(s)
}

func fconst(bits uint32) Expr { return Expr{Op: ExprConst, DType: F32, Bits: bits} }

func neg(a Expr) Expr { return Expr{Op: ExprNeg, Args: []Expr{a}} }

func add(a, b Expr) Expr { return Expr{Op: ExprAdd, Args: []Expr{a, b}} }

func mul(a, b Expr) Expr { return Expr{Op: ExprMul, Args: []Expr{a, b}} }

func div(a, b Expr) Expr { return Expr{Op: ExprDiv, Args: []Expr{a, b}} }

func mod(a, b Expr) Expr { return Expr{Op: ExprMod, Args: []Expr{a, b}} }

func band(a, b Expr) Expr { return Expr{Op: ExprBitAnd, Args: []Expr{a, b}} }

func bor(a, b Expr) Expr { return Expr{Op: ExprBitOr, Args: []Expr{a, b}} }

func bxor(a, b Expr) Expr { return Expr{Op: ExprBitXor, Args: []Expr{a, b}} }

func shl(a, b Expr) Expr { return Expr{Op: ExprShl, Args: []Expr{a, b}} }

func shr(a, b Expr) Expr { return Expr{Op: ExprShr, Args: []Expr{a, b}} }

func rel(r Rel, a, b Expr) Expr { return Expr{Op: ExprRel, Rel: r, Args: []Expr{a, b}} }

func land(a, b Expr) Expr { return Expr{Op: ExprAnd, Args: []Expr{a, b}} }

func sel(c, a, b Expr) Expr { return Expr{Op: ExprSel, Args: []Expr{c, a, b}} }

func cast(d DType, a Expr) Expr { return Expr{Op: ExprCast, DType: d, Args: []Expr{a}} }

func callFn(f Fn, args ...Expr) Expr { return Expr{Op: ExprFn, Fn: f, Args: args} }

func load(name string, index Expr) Expr {
	return Expr{Op: ExprLoad, Name: name, Args: []Expr{index}}
}

func zeroExpr(d DType) Expr { return Expr{Op: ExprConst, DType: d, Bits: 0} }

func constExpr(n *node) Expr { return Expr{Op: ExprConst, DType: n.dtype, Bits: n.bits} }

func (n *node) exprALU(args []string) Expr {
	a := make([]Expr, len(args))
	for i, s := range args {
		a[i] = ref(s)
	}
	switch n.op {
	case EXP2:
		return callFn(FnExp2, a[0])
	case LOG2:
		return callFn(FnLog2, a[0])
	case SIN:
		return callFn(FnSin, a[0])
	case SQRT:
		return callFn(FnSqrt, a[0])
	case RECIP:
		return div(fconst(math.Float32bits(1)), a[0])
	case NEG:
		return neg(a[0])
	case CAST:
		from := F32
		if len(n.sources) > 0 {
			from = n.sources[0].dtype
		}
		switch n.dtype {
		case U8:
			if from == F32 {
				return cast(U8, callFn(FnClamp, a[0], fconst(0), fconst(math.Float32bits(255))))
			}
			return cast(U8, callFn(FnClamp, a[0], iconst(0), iconst(255)))
		default:
			return cast(n.dtype, a[0])
		}
	case ADD:
		return add(a[0], a[1])
	case MUL:
		return mul(a[0], a[1])
	case IDIV:
		return divGuard(a[0], a[1])
	case MAX:
		return callFn(FnMax, a[0], a[1])
	case MOD:
		return modGuard(a[0], a[1])
	case CMPLT:
		return cast(I32, rel(RelLT, a[0], a[1]))
	case CMPNE:
		return cast(I32, rel(RelNE, a[0], a[1]))
	case XOR:
		return bxor(a[0], a[1])
	case SHL:
		return shiftGuard(a[0], a[1], false)
	case SHR:
		return shiftGuard(a[0], a[1], true)
	case OR:
		return bor(a[0], a[1])
	case AND:
		return band(a[0], a[1])
	case WHERE:
		return sel(rel(RelNE, a[0], iconst(0)), a[1], a[2])
	case MultiplyAccumulate:
		return add(mul(a[0], a[1]), a[2])
	default:
		return iconst(0)
	}
}

func divGuard(a, b Expr) Expr {
	bz := rel(RelEQ, b, iconst(0))
	return sel(bz, iconst(0), div(a, sel(bz, iconst(1), b)))
}

func modGuard(a, b Expr) Expr {
	bz := rel(RelEQ, b, iconst(0))
	return sel(bz, iconst(0), mod(a, sel(bz, iconst(1), b)))
}

func shiftGuard(a, b Expr, arith bool) Expr {
	ge := rel(RelGE, cast(U8, b), uintc(32))
	shamt := cast(I32, band(cast(U8, b), uintc(31)))
	op := shl(a, shamt)
	if arith {
		op = shr(a, shamt)
	}
	if !arith {
		return sel(ge, iconst(0), op)
	}
	neg1 := sel(rel(RelLT, a, iconst(0)), iconst(-1), iconst(0))
	return sel(ge, neg1, op)
}
