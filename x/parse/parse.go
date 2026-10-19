package parse

import (
	"sort"
)

const (
	maxHeads   = 4096
	maxEpsilon = 64
	// maxStall is how many reductions a head may make without shifting.
	// A nullable cycle otherwise sits in one state until the step limit,
	// after another head has already accepted the file.
	maxStall = 8192
	// maxPaths bounds distinct stack ancestors while reducing one production.
	// Combinations of those ancestors are not materialized. Passing the
	// bound fails the parse.
	maxPaths = 8192
)

// pathLimit is maxPaths. Tests lower it to prove the bound is an error.
var pathLimit = maxPaths

const (
	redOK = iota
	redStall
	redSyntax
	redBound
)

type link struct {
	state  uint32
	parent int32
	node   uint32
	dyn    int32
	more   int32 // alternate predecessor, or -1
}

// extra is another way to reach the same stack node: a different parent,
// the tree built on that path, and its dynamic precedence.
type extra struct {
	parent int32
	node   uint32
	dyn    int32
	more   int32
}

type head struct {
	link  int32
	pos   int
	has   bool
	sym   SymbolID
	a, b  int
	eps   int
	stall int
	acc   bool
	node  uint32
}

type tok struct {
	id   SymbolID
	a, b int
}

// Parse lexes src and builds one tree. A scanner-based language reads the
// token stream once. The default lexer asks the current state which tokens
// are legal. The stack is a graph: a fork shares the tail it forked from.
func (l *Language) Parse(src []byte) (*Tree, error) {
	if l == nil {
		return nil, &Error{Line: 1, Col: 1}
	}
	if src == nil {
		src = []byte{}
	}
	p := &pr{
		lang:  l,
		src:   src,
		nodes: []node{{}},
		lines: lineStarts(src),
	}
	var err error
	if l.scan != nil {
		p.toks, err = l.prelex(src)
		if err != nil {
			return nil, err
		}
		p.tokMode = true
	}
	p.links = []link{{state: l.start, parent: -1, more: -1}}
	p.heads = []head{{link: 0}}
	limit := len(src)*80 + 5000
	if limit < 5000 {
		limit = 5000
	}
	var best *Error
	var bestState uint32
	for range limit {
		if len(p.heads) == 0 {
			break
		}
		// Act only at the earliest input position. A head that has already
		// shifted waits, so every stack shares the token it is on.
		minPos := -1
		for i := range p.heads {
			h := &p.heads[i]
			if h.acc {
				continue
			}
			if minPos < 0 || h.pos < minPos {
				minPos = h.pos
			}
		}
		if minPos < 0 {
			break
		}
		var next []head
		for _, h := range p.heads {
			if h.acc || h.pos != minPos {
				next = append(next, h)
				continue
			}
			grew, dead := p.act(h)
			if dead != nil && (best == nil || dead.Offset > best.Offset) {
				best = dead
				bestState = p.errState
			}
			next = append(next, grew...)
		}
		merged, ambig := p.merge(next)
		if ambig != nil {
			return nil, p.withWant(ambig, p.errState)
		}
		if p.bound {
			return nil, p.boundError()
		}
		p.heads = merged
	}
	var won *head
	for i := range p.heads {
		h := &p.heads[i]
		if !h.acc {
			continue
		}
		if won == nil || p.better(*h, *won) {
			won = h
		}
	}
	if won == nil {
		if best == nil {
			best = p.errAt(0, nil)
			bestState = p.errState
		}
		return nil, p.withWant(best, bestState)
	}
	return concreteTree(l, src, p.nodes, p.kids, p.lines, won.node), nil
}

type pr struct {
	lang      *Language
	src       []byte
	toks      []tok
	tokMode   bool
	nodes     []node
	kids      []uint32
	links     []link
	extras    []extra
	heads     []head
	lines     []int
	wantBuf   []string
	errState  uint32
	bound     bool
	boundPos  int
	boundHead head
}

func (l *Language) prelex(src []byte) ([]tok, error) {
	st := &LexState{Symbol: l.Symbol, SemiAt: -1}
	valid := func(SymbolID) bool { return true }
	var toks []tok
	zeros := 0
	limit := len(src) + 8
	if limit < 8 {
		limit = 8
	}
	for len(toks) < limit {
		prev := st.At
		id, a, b, ok := l.scan(src, st.At, valid, st)
		if !ok || a < 0 || b < a || b > len(src) {
			off := st.At
			if off < 0 || off > len(src) {
				off = 0
			}
			return nil, syntaxAt(src, off, nil)
		}
		if st.At == prev {
			st.At = b
		}
		if id != SymEOF && a == b {
			zeros++
			if zeros > 4 {
				return nil, syntaxAt(src, a, nil)
			}
		} else {
			zeros = 0
		}
		toks = append(toks, tok{id: id, a: a, b: b})
		if id == SymEOF {
			return toks, nil
		}
	}
	return nil, syntaxAt(src, len(src), nil)
}

func (p *pr) act(h head) ([]head, *Error) {
	if !p.ensure(&h) {
		return nil, p.errAt(h.pos, &h)
	}
	st := p.links[h.link].state
	acts := p.lang.act[cellKey(st, h.sym)]
	if len(acts) == 0 {
		return nil, p.errAt(h.pos, &h)
	}
	out := make([]head, 0, len(acts))
	for _, a := range acts {
		h2 := h
		switch a.kind {
		case actShift:
			if !p.doShift(&h2, a.val) {
				return out, p.errAt(h.pos, &h)
			}
		case actReduce:
			reduced, kind := p.doReduce(h2, a.val)
			if kind == redStall {
				continue
			}
			if kind == redBound {
				p.noteBound(h)
				return nil, nil
			}
			if kind == redSyntax {
				return out, p.errAt(h.pos, &h)
			}
			out = append(out, reduced...)
			// h2 is still the pre-reduce head. The reduced heads carry
			// the new links.
			continue
		case actAccept:
			h2.acc = true
			h2.node = p.bestNode(h2.link)
		default:
			continue
		}
		out = append(out, h2)
	}
	return out, nil
}

func (p *pr) ensure(h *head) bool {
	if h.has {
		return true
	}
	if p.tokMode {
		if h.pos < 0 || h.pos >= len(p.toks) {
			return false
		}
		t := p.toks[h.pos]
		h.sym, h.a, h.b = t.id, t.a, t.b
		h.has = true
		return true
	}
	if h.pos > len(p.src) {
		return false
	}
	id, a, b, ok := p.lang.lex(p.src, h.pos, p.links[h.link].state, p.prevSymbol(h.link))
	if !ok {
		// a is the byte the lexer stopped on, after extras.
		h.pos = a
		h.a, h.b = a, b
		return false
	}
	h.sym, h.a, h.b = id, a, b
	h.has = true
	return true
}

// prevSymbol is the previous terminal on the best path of this stack.
// A token whose text is only spaces or tabs is skipped, so After still
// sees the token before them. An abandoned predecessor does not win the
// lookup: the lexer follows the same node the accept path would keep.
func (p *pr) prevSymbol(link int32) SymbolID {
	for link >= 0 {
		nodeID := p.bestNode(link)
		if nodeID == 0 {
			return 0
		}
		kind := p.nodes[nodeID].kind
		next := p.predParent(link, nodeID)
		if int(kind) >= len(p.lang.syms) || !p.lang.syms[kind].terminal {
			link = next
			continue
		}
		n := p.nodes[nodeID]
		if !onlySpace(p.src, n.start, n.end) {
			return kind
		}
		link = next
	}
	return 0
}

// predParent is the stack link under the predecessor that holds node.
func (p *pr) predParent(lk int32, node uint32) int32 {
	if lk < 0 {
		return -1
	}
	L := p.links[lk]
	if L.node == node {
		return L.parent
	}
	for m := L.more; m >= 0; m = p.extras[m].more {
		if p.extras[m].node == node {
			return p.extras[m].parent
		}
	}
	return L.parent
}

func onlySpace(src []byte, start, end uint32) bool {
	if end <= start || int(end) > len(src) {
		return false
	}
	for i := start; i < end; i++ {
		if src[i] != ' ' && src[i] != '\t' {
			return false
		}
	}
	return true
}

func (p *pr) doShift(h *head, state uint32) bool {
	leaf := p.add(node{
		kind:  h.sym,
		flags: p.leafFlags(h.sym),
		start: uint32(h.a),
		end:   uint32(h.b),
	})
	p.links = append(p.links, link{
		state:  state,
		parent: h.link,
		node:   leaf,
		dyn:    0,
		more:   -1,
	})
	h.link = int32(len(p.links) - 1)
	if p.tokMode {
		h.pos++
	} else {
		h.pos = h.b
	}
	h.has = false
	h.eps = 0
	h.stall = 0
	return true
}

func (p *pr) doReduce(h head, prod uint32) ([]head, int) {
	if int(prod) >= len(p.lang.prods) {
		return nil, redSyntax
	}
	info := p.lang.prods[prod]
	n := len(info.rhs)
	h.stall++
	if h.stall > maxStall {
		return nil, redStall
	}
	if n == 0 {
		h.eps++
		if h.eps > maxEpsilon {
			return nil, redSyntax
		}
		return p.finishReduce(h, info, prod, h.link, nil, int32(info.dyn))
	}
	h.eps = 0
	// One predecessor along the whole right-hand side is the common case.
	// The slice lives only until the node is committed.
	if raw, anc, dyn, ok := p.onePath(h.link, n, int32(info.dyn)); ok {
		return p.finishReduce(h, info, prod, anc, raw, dyn)
	}
	// Several predecessors: one best path per ancestor. Each step points
	// at the step it extends, so a shared tail is not copied. The node
	// list is written once, when that path commits a record.
	steps := make([]pathStep, 0, n)
	cur := []pathCursor{{step: -1, link: h.link, dyn: int32(info.dyn)}}
	for range n {
		next := make([]pathCursor, 0, len(cur))
		best := make(map[int32]int, len(cur))
		overflow := false
		for _, part := range cur {
			p.eachPred(part.link, func(parent int32, node uint32, d int32) {
				if overflow || parent < 0 {
					return
				}
				dyn := part.dyn + d
				if i, ok := best[parent]; ok {
					if dyn <= next[i].dyn {
						return
					}
					steps = append(steps, pathStep{tail: part.step, node: node})
					next[i] = pathCursor{step: int32(len(steps) - 1), link: parent, dyn: dyn}
					return
				}
				if len(next) >= pathLimit {
					overflow = true
					return
				}
				steps = append(steps, pathStep{tail: part.step, node: node})
				best[parent] = len(next)
				next = append(next, pathCursor{step: int32(len(steps) - 1), link: parent, dyn: dyn})
			})
			if overflow {
				return nil, redBound
			}
		}
		if len(next) == 0 {
			return nil, redSyntax
		}
		cur = next
	}
	var out []head
	saw := false
	raw := make([]uint32, n)
	for _, part := range cur {
		flattenPath(steps, part.step, raw)
		got, kind := p.finishReduce(h, info, prod, part.link, raw, part.dyn)
		if kind != redOK {
			continue
		}
		saw = true
		out = append(out, got...)
	}
	if !saw {
		return nil, redSyntax
	}
	return out, redOK
}

// pathStep is one popped edge. tail is the previous edge, or -1.
// The step list is scratch for one reduction. A kept path points at its
// tail; a path that loses is not appended.
type pathStep struct {
	tail int32
	node uint32
}

// pathCursor is the best path found for one ancestor so far.
type pathCursor struct {
	step int32
	link int32
	dyn  int32
}

// flattenPath writes the path leftmost child first. The cursor sits on the
// last pop, which is the left edge, and each tail is the edge to its right.
func flattenPath(steps []pathStep, step int32, raw []uint32) {
	for i := range raw {
		s := steps[step]
		raw[i] = s.node
		step = s.tail
	}
}

// onePath pops n stack links when each has a single predecessor.
// The returned slice is owned by the caller for this reduction.
func (p *pr) onePath(lk int32, n int, dyn int32) ([]uint32, int32, int32, bool) {
	raw := make([]uint32, n)
	for i := n - 1; i >= 0; i-- {
		if lk < 0 || p.links[lk].more >= 0 {
			return nil, 0, 0, false
		}
		raw[i] = p.links[lk].node
		dyn += p.links[lk].dyn
		lk = p.links[lk].parent
	}
	if lk < 0 {
		return nil, 0, 0, false
	}
	return raw, lk, dyn, true
}

func (p *pr) finishReduce(h head, info prodInfo, prod uint32, ancestor int32, raw []uint32, dyn int32) ([]head, int) {
	if ancestor < 0 {
		return nil, redSyntax
	}
	id, _, _ := p.parent(info, uint16(prod), raw, dyn, h.a)
	dest, ok := p.lang.gto[cellKey(p.links[ancestor].state, info.lhs)]
	if !ok {
		return nil, redSyntax
	}
	p.links = append(p.links, link{
		state:  dest,
		parent: ancestor,
		node:   id,
		dyn:    dyn,
		more:   -1,
	})
	h.link = int32(len(p.links) - 1)
	return []head{h}, redOK
}

func (p *pr) eachPred(lk int32, fn func(parent int32, node uint32, dyn int32)) {
	if lk < 0 {
		return
	}
	L := p.links[lk]
	fn(L.parent, L.node, L.dyn)
	for m := L.more; m >= 0; m = p.extras[m].more {
		e := p.extras[m]
		fn(e.parent, e.node, e.dyn)
	}
}

// addPred records another parent of lk. The same parent keeps the higher
// dynamic precedence.
func (p *pr) addPred(lk, parent int32, node uint32, dyn int32) {
	L := &p.links[lk]
	if L.parent == parent {
		if dyn > L.dyn || (dyn == L.dyn && node < L.node) {
			L.dyn = dyn
			L.node = node
		}
		return
	}
	for m := L.more; m >= 0; m = p.extras[m].more {
		e := &p.extras[m]
		if e.parent != parent {
			continue
		}
		if dyn > e.dyn || (dyn == e.dyn && node < e.node) {
			e.dyn = dyn
			e.node = node
		}
		return
	}
	p.extras = append(p.extras, extra{parent: parent, node: node, dyn: dyn, more: L.more})
	p.links[lk].more = int32(len(p.extras) - 1)
}

func (p *pr) union(dst, src int32) {
	if dst == src {
		return
	}
	p.eachPred(src, func(parent int32, node uint32, dyn int32) {
		p.addPred(dst, parent, node, dyn)
	})
}

func (p *pr) leafFlags(sym SymbolID) uint8 {
	if int(sym) < len(p.lang.syms) && p.lang.syms[sym].named {
		return flagNamed
	}
	return 0
}

func (p *pr) expand(raw []uint32) []uint32 {
	var out []uint32
	var walk func(uint32)
	walk = func(id uint32) {
		if id == 0 {
			return
		}
		n := p.nodes[id]
		if n.flags&flagHidden != 0 {
			for i := uint32(0); i < n.nchild; i++ {
				walk(p.kids[n.child+i])
			}
			return
		}
		out = append(out, id)
	}
	for _, id := range raw {
		walk(id)
	}
	return out
}

func (p *pr) parent(pr prodInfo, prod uint16, raw []uint32, dyn int32, at int) (uint32, uint32, uint32) {
	// Hidden rules keep the direct children. Flattening a repeat here copies
	// the whole list on every element. A visible parent flattens once.
	nkeep := 0
	for _, id := range raw {
		if id != 0 {
			raw[nkeep] = id
			nkeep++
		}
	}
	direct := raw[:nkeep]
	start, end := uint32(at), uint32(at)
	if len(direct) > 0 {
		start = p.nodes[direct[0]].start
		end = p.nodes[direct[0]].end
		for _, id := range direct[1:] {
			n := p.nodes[id]
			if n.start < start {
				start = n.start
			}
			if n.end > end {
				end = n.end
			}
		}
	}
	flags := uint8(0)
	if pr.hidden {
		flags |= flagHidden
	}
	if int(pr.lhs) < len(p.lang.syms) && p.lang.syms[pr.lhs].named {
		flags |= flagNamed
	}
	if pr.hidden && len(direct) == 0 {
		return 0, start, end
	}
	if pr.hidden && len(direct) == 1 {
		// A hidden alternative with one child is that child. The link still
		// carries the summed dynamic precedence.
		return direct[0], start, end
	}
	kids := direct
	if !pr.hidden {
		kids = p.expand(direct)
		if len(kids) > 0 {
			start = p.nodes[kids[0]].start
			end = p.nodes[kids[len(kids)-1]].end
		}
	}
	base := uint32(len(p.kids))
	p.kids = append(p.kids, kids...)
	id := p.add(node{
		kind:   pr.lhs,
		flags:  flags,
		start:  start,
		end:    end,
		child:  base,
		nchild: uint32(len(kids)),
		dyn:    dyn,
		prod:   prod,
	})
	return id, start, end
}

func (p *pr) add(n node) uint32 {
	id := uint32(len(p.nodes))
	p.nodes = append(p.nodes, n)
	return id
}

func (p *pr) noteBound(h head) {
	if p.bound {
		return
	}
	p.bound = true
	p.boundPos = h.pos
	p.boundHead = h
}

func (p *pr) boundError() *Error {
	e := p.errAt(p.boundPos, &p.boundHead)
	e.Detail = "too many stacks"
	e.Want = nil
	return e
}

func (p *pr) merge(heads []head) ([]head, *Error) {
	// One head per state and input position. Other stacks that reached
	// that state stay as predecessors, so a later reduce can follow each
	// of them without parsing the shared span twice.
	type key struct {
		state uint32
		pos   int
		sym   SymbolID
		has   bool
		acc   bool
	}
	idx := make(map[key]int, len(heads))
	out := make([]head, 0, len(heads))
	for _, h := range heads {
		sym := h.sym
		if !h.has {
			sym = 0
		}
		k := key{
			state: p.links[h.link].state,
			pos:   h.pos,
			sym:   sym,
			has:   h.has,
			acc:   h.acc,
		}
		if i, ok := idx[k]; ok {
			if h.acc && p.better(h, out[i]) {
				out[i].node = h.node
			}
			p.union(out[i].link, h.link)
			continue
		}
		idx[k] = len(out)
		out = append(out, h)
	}
	if len(out) <= maxHeads {
		return out, nil
	}
	// The head limit is a search bound. Heads at the furthest position
	// are the ones that can still accept. A frontier wider than the limit
	// is an error: the parse does not guess among them. Heads behind that
	// position are abandoned. They are not a second tree.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].pos != out[j].pos {
			return out[i].pos > out[j].pos
		}
		di := p.links[out[i].link].dyn
		dj := p.links[out[j].link].dyn
		if di != dj {
			return di > dj
		}
		return p.nodeOf(out[i]) < p.nodeOf(out[j])
	})
	best := out[0].pos
	n := 0
	for _, h := range out {
		if h.pos != best {
			break
		}
		n++
	}
	if n > maxHeads {
		e := p.errAt(best, &out[0])
		e.Detail = "too many stacks"
		e.Want = nil
		return nil, e
	}
	return out[:n], nil
}

func (p *pr) better(a, b head) bool {
	da := p.linkDyn(a.link)
	db := p.linkDyn(b.link)
	if da != db {
		return da > db
	}
	return p.nodeOf(a) < p.nodeOf(b)
}

func (p *pr) linkDyn(lk int32) int32 {
	if lk < 0 {
		return 0
	}
	d := p.links[lk].dyn
	for m := p.links[lk].more; m >= 0; m = p.extras[m].more {
		if p.extras[m].dyn > d {
			d = p.extras[m].dyn
		}
	}
	return d
}

func (p *pr) bestNode(lk int32) uint32 {
	if lk < 0 {
		return 0
	}
	bestN := p.links[lk].node
	bestD := p.links[lk].dyn
	for m := p.links[lk].more; m >= 0; m = p.extras[m].more {
		e := p.extras[m]
		if e.dyn > bestD || (e.dyn == bestD && e.node < bestN) {
			bestD = e.dyn
			bestN = e.node
		}
	}
	return bestN
}

func (p *pr) nodeOf(h head) uint32 {
	if h.acc {
		return h.node
	}
	return p.links[h.link].node
}

func (p *pr) errAt(pos int, h *head) *Error {
	off := pos
	var st uint32
	if h != nil {
		st = p.links[h.link].state
		if p.tokMode {
			if h.pos >= 0 && h.pos < len(p.toks) {
				off = p.toks[h.pos].a
			} else if h.has {
				off = h.a
			} else {
				off = len(p.src)
			}
		} else if h.has {
			off = h.a
		} else if h.pos >= 0 && h.pos <= len(p.src) {
			off = h.pos
		}
	}
	p.errState = st
	if off < 0 {
		off = 0
	}
	if off > len(p.src) {
		off = len(p.src)
	}
	line, col := lineCol(p.lines, off)
	return &Error{Offset: off, Line: line, Col: col}
}

func (p *pr) withWant(e *Error, state uint32) *Error {
	if e == nil {
		e = p.errAt(0, nil)
		state = p.errState
	}
	e.Want = p.expected(state)
	return e
}

func (p *pr) expected(state uint32) []string {
	p.wantBuf = p.wantBuf[:0]
	for id, s := range p.lang.syms {
		if id < 2 || s.name == "" || s.extra || s.hidden {
			continue
		}
		if _, ok := p.lang.act[cellKey(state, SymbolID(id))]; ok {
			p.wantBuf = append(p.wantBuf, s.name)
		}
	}
	sort.Strings(p.wantBuf)
	if len(p.wantBuf) > 8 {
		p.wantBuf = p.wantBuf[:8]
	}
	out := make([]string, len(p.wantBuf))
	copy(out, p.wantBuf)
	return out
}

func syntaxAt(src []byte, off int, want []string) *Error {
	if off < 0 {
		off = 0
	}
	if off > len(src) {
		off = len(src)
	}
	line, col := lineCol(lineStarts(src), off)
	return &Error{Offset: off, Line: line, Col: col, Want: want}
}
