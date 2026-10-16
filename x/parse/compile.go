package parse

import (
	"encoding/binary"
	"sort"

	"github.com/lewtec/lewkit/x/errs"
)

const (
	actShift  uint8 = 1
	actReduce uint8 = 2
	actAccept uint8 = 3
)

type action struct {
	kind uint8
	val  uint32
}

type symInfo struct {
	name      string
	terminal  bool
	named     bool
	hidden    bool
	extra     bool
	immediate bool
	external  bool
	match     Match
	lexPrec   int
}

type prodInfo struct {
	lhs     SymbolID
	rhs     []SymbolID
	prec    int
	hasPrec bool
	assoc   Assoc
	dyn     int
	hidden  bool
}

// Language is a compiled grammar.
type Language struct {
	name   string
	syms   []symInfo
	prods  []prodInfo
	act    map[uint64][]action
	gto    map[uint64]uint32
	start  uint32
	scan   ScanFunc
	extras []SymbolID
	byName map[string]SymbolID
	states int
	forks  int
}

// States returns the number of LR(1) states.
func (l *Language) States() int {
	if l == nil {
		return 0
	}
	return l.states
}

// Symbol resolves a terminal or rule name in the compiled language.
func (l *Language) Symbol(name string) (SymbolID, bool) {
	if l == nil {
		return 0, false
	}
	id, ok := l.byName[name]
	return id, ok
}

func cellKey(state uint32, sym SymbolID) uint64 {
	return uint64(state)<<16 | uint64(sym)
}

type item struct {
	prod uint16
	dot  uint16
	look SymbolID
}

type rawAct struct {
	kind  uint8
	val   uint32
	prec  int
	has   bool
	assoc Assoc
}

// Compile builds LR(1) tables. Precedence removes a shift/reduce action when
// one side wins. An unresolved pair stays, and Parse forks on it.
func (g *Grammar) Compile() (*Language, error) {
	if g == nil {
		return nil, errs.Wrap(errs.ErrInvalid, "compile: nil grammar")
	}
	start := g.start
	if !g.startSet {
		for _, p := range g.prods {
			if !g.syms[p.lhs].hidden {
				start = p.lhs
				break
			}
		}
	}
	if start == 0 || int(start) >= len(g.syms) {
		return nil, errs.Wrap(errs.ErrInvalid, "compile: no start rule")
	}
	if g.syms[start].kind == kindTerm {
		return nil, errs.Wrap(errs.ErrInvalid, "compile: start is a token")
	}

	c := &compiler{
		index: map[string]uint32{},
	}
	c.syms = make([]symInfo, len(g.syms)+1)
	for i, s := range g.syms {
		c.syms[i] = symInfo{
			name:      s.name,
			terminal:  s.kind == kindTerm,
			named:     s.named,
			hidden:    s.hidden,
			extra:     s.extra,
			immediate: s.immediate,
			external:  s.external,
			match:     s.match,
			lexPrec:   s.lexPrec,
		}
	}
	acceptID := SymbolID(len(g.syms))
	c.syms[acceptID] = symInfo{name: "$accept", hidden: true}

	c.prods = make([]prodInfo, len(g.prods)+1)
	c.prods[0] = prodInfo{lhs: acceptID, rhs: []SymbolID{start}, hidden: true}
	lhsCount := map[SymbolID]int{}
	if len(g.prods)+1 > int(^uint16(0)) {
		return nil, errs.Wrap(errs.ErrInvalid, "compile: too many alternatives")
	}
	for i, p := range g.prods {
		if p.lhs == 0 {
			return nil, errs.Wrap(errs.ErrInvalid, "compile: production without a rule")
		}
		if g.syms[p.lhs].kind == kindTerm {
			return nil, errs.Wrap(errs.ErrInvalid, "compile: token and rule share the name "+g.syms[p.lhs].name)
		}
		info := prodInfo{
			lhs: p.lhs, rhs: append([]SymbolID(nil), p.rhs...),
			prec: p.prec, hasPrec: p.hasPrec, assoc: p.assoc, dyn: p.dyn,
			hidden: g.syms[p.lhs].hidden,
		}
		c.prods[i+1] = info
		lhsCount[p.lhs]++
		for _, s := range p.rhs {
			if s == 0 || int(s) >= len(g.syms) {
				return nil, errs.Wrap(errs.ErrInvalid, "compile: empty symbol in a rule")
			}
			if g.syms[s].extra {
				return nil, errs.Wrap(errs.ErrInvalid, "compile: skipped token "+g.syms[s].name+" is in a rule")
			}
		}
	}
	if lhsCount[start] == 0 {
		return nil, errs.Wrap(errs.ErrInvalid, "compile: start has no alternative")
	}
	used := map[SymbolID]bool{}
	var walk func(SymbolID)
	walk = func(id SymbolID) {
		if used[id] {
			return
		}
		used[id] = true
		for i := range c.prods {
			if c.prods[i].lhs != id {
				continue
			}
			for _, s := range c.prods[i].rhs {
				if !c.syms[s].terminal {
					walk(s)
				} else {
					used[s] = true
				}
			}
		}
	}
	walk(acceptID)
	for id, s := range c.syms {
		if id == 0 || s.terminal || s.extra || !used[SymbolID(id)] {
			continue
		}
		if lhsCount[SymbolID(id)] == 0 && SymbolID(id) != acceptID {
			return nil, errs.Wrap(errs.ErrInvalid, "compile: "+s.name+" has no alternative")
		}
	}
	if g.scan == nil {
		for id, s := range c.syms {
			if s.terminal && s.extra && s.match == nil && SymbolID(id) != SymEOF {
				return nil, errs.Wrap(errs.ErrInvalid, "compile: skipped token "+s.name+" has no matcher")
			}
			if s.terminal && !s.extra && !s.external && used[SymbolID(id)] && s.match == nil && SymbolID(id) != SymEOF {
				return nil, errs.Wrap(errs.ErrInvalid, "compile: token "+s.name+" has no matcher")
			}
		}
	}

	c.nullableFirst()
	c.byLHS = make([][]uint16, len(c.syms))
	for i := range c.prods {
		lhs := c.prods[i].lhs
		c.byLHS[lhs] = append(c.byLHS[lhs], uint16(i))
	}

	startItems := c.closure([]item{{prod: 0, dot: 0, look: SymEOF}})
	c.intern(startItems)
	for si := 0; si < len(c.states); si++ {
		seen := map[SymbolID]bool{}
		var syms []SymbolID
		for _, it := range c.states[si] {
			p := c.prods[it.prod]
			if int(it.dot) >= len(p.rhs) {
				continue
			}
			s := p.rhs[it.dot]
			if !seen[s] {
				seen[s] = true
				syms = append(syms, s)
			}
		}
		for _, s := range syms {
			var moved []item
			in := map[item]struct{}{}
			for _, it := range c.states[si] {
				p := c.prods[it.prod]
				if int(it.dot) >= len(p.rhs) || p.rhs[it.dot] != s {
					continue
				}
				ni := item{prod: it.prod, dot: it.dot + 1, look: it.look}
				if _, ok := in[ni]; ok {
					continue
				}
				in[ni] = struct{}{}
				moved = append(moved, ni)
			}
			dest := c.intern(c.closure(moved))
			c.edge[si][s] = dest
		}
	}

	lang := &Language{
		name:   g.name,
		syms:   c.syms,
		prods:  c.prods,
		act:    map[uint64][]action{},
		gto:    map[uint64]uint32{},
		start:  0,
		scan:   g.scan,
		extras: append([]SymbolID(nil), g.extras...),
		byName: make(map[string]SymbolID, len(c.syms)),
		states: len(c.states),
	}
	for i, s := range c.syms {
		if i == 0 || s.name == "" {
			continue
		}
		lang.byName[s.name] = SymbolID(i)
	}
	for si := range c.states {
		for sym, dest := range c.edge[si] {
			lang.gto[cellKey(uint32(si), sym)] = dest
		}
		raw := map[SymbolID][]rawAct{}
		for _, it := range c.states[si] {
			p := c.prods[it.prod]
			if int(it.dot) >= len(p.rhs) {
				if it.prod == 0 && it.look == SymEOF {
					raw[SymEOF] = append(raw[SymEOF], rawAct{kind: actAccept})
					continue
				}
				raw[it.look] = append(raw[it.look], rawAct{
					kind: actReduce, val: uint32(it.prod),
					prec: p.prec, has: p.hasPrec, assoc: p.assoc,
				})
				continue
			}
			sym := p.rhs[it.dot]
			if !c.syms[sym].terminal {
				continue
			}
			dest, ok := c.edge[si][sym]
			if !ok {
				continue
			}
			raw[sym] = append(raw[sym], rawAct{
				kind: actShift, val: dest,
				prec: p.prec, has: p.hasPrec, assoc: p.assoc,
			})
		}
		for sym, list := range raw {
			kept := resolveCell(list)
			if len(kept) == 0 {
				continue
			}
			acts := make([]action, len(kept))
			for i, a := range kept {
				acts[i] = action{kind: a.kind, val: a.val}
			}
			lang.act[cellKey(uint32(si), sym)] = acts
			if len(acts) > 1 {
				lang.forks++
			}
		}
	}
	return lang, nil
}

// Forks is the number of table cells that still have more than one action.
func (l *Language) Forks() int {
	if l == nil {
		return 0
	}
	return l.forks
}

func resolveCell(list []rawAct) []rawAct {
	var shifts []rawAct
	var reds []rawAct
	accept := false
	for _, a := range list {
		switch a.kind {
		case actAccept:
			accept = true
		case actShift:
			shifts = append(shifts, a)
		case actReduce:
			reds = append(reds, a)
		}
	}
	if accept {
		return []rawAct{{kind: actAccept}}
	}
	shift, shiftOK := mergeShifts(shifts)
	if !shiftOK && len(reds) == 0 {
		return nil
	}
	if !shiftOK {
		return dedupReduce(reds)
	}
	if len(reds) == 0 {
		return []rawAct{shift}
	}
	keepShift := false
	var keepRed []rawAct
	undecided := false
	for _, r := range reds {
		ks, kr := resolvePair(shift, r)
		if ks && kr {
			undecided = true
		}
		if ks {
			keepShift = true
		}
		if kr {
			keepRed = append(keepRed, r)
		}
	}
	if undecided {
		out := []rawAct{shift}
		return append(out, dedupReduce(reds)...)
	}
	var out []rawAct
	if keepShift {
		out = append(out, shift)
	}
	out = append(out, dedupReduce(keepRed)...)
	if len(out) == 0 {
		out = append(out, shift)
		out = append(out, dedupReduce(reds)...)
	}
	return out
}

func mergeShifts(list []rawAct) (rawAct, bool) {
	if len(list) == 0 {
		return rawAct{}, false
	}
	s := list[0]
	ambiguous := false
	for _, a := range list[1:] {
		if a.val != s.val {
			s.val = a.val
		}
		if a.has && s.has && a.prec != s.prec {
			ambiguous = true
		}
		if a.has && !s.has {
			s.prec = a.prec
			s.has = true
			s.assoc = a.assoc
		}
	}
	if ambiguous {
		s.has = false
	}
	return s, true
}

func dedupReduce(list []rawAct) []rawAct {
	if len(list) == 0 {
		return nil
	}
	seen := map[uint32]bool{}
	var out []rawAct
	for _, a := range list {
		if seen[a.val] {
			continue
		}
		seen[a.val] = true
		out = append(out, a)
	}
	return out
}

// resolvePair reports which side of a shift/reduce conflict survives.
// A side with no declared precedence is treated as 0 when the other side
// declared one. When neither declared one, both survive.
func resolvePair(shift, red rawAct) (keepShift, keepRed bool) {
	if !shift.has && !red.has {
		return true, true
	}
	sp, rp := 0, 0
	if shift.has {
		sp = shift.prec
	}
	if red.has {
		rp = red.prec
	}
	if sp > rp {
		return true, false
	}
	if rp > sp {
		return false, true
	}
	switch red.assoc {
	case AssocLeft:
		return false, true
	case AssocRight:
		return true, false
	default:
		return true, true
	}
}

type compiler struct {
	syms     []symInfo
	prods    []prodInfo
	nullable []bool
	first    [][]SymbolID
	byLHS    [][]uint16
	states   [][]item
	index    map[string]uint32
	edge     []map[SymbolID]uint32
	buf      []byte
}

func (c *compiler) nullableFirst() {
	n := len(c.syms)
	c.nullable = make([]bool, n)
	c.first = make([][]SymbolID, n)
	changed := true
	for changed {
		changed = false
		for i := range c.prods {
			p := c.prods[i]
			if len(p.rhs) == 0 {
				if !c.nullable[p.lhs] {
					c.nullable[p.lhs] = true
					changed = true
				}
				continue
			}
			allNull := true
			for _, s := range p.rhs {
				if c.syms[s].terminal {
					if addSym(&c.first[p.lhs], s) {
						changed = true
					}
					allNull = false
					break
				}
				if addSyms(&c.first[p.lhs], c.first[s]) {
					changed = true
				}
				if !c.nullable[s] {
					allNull = false
					break
				}
			}
			if allNull && !c.nullable[p.lhs] {
				c.nullable[p.lhs] = true
				changed = true
			}
		}
	}
}

func addSym(dst *[]SymbolID, id SymbolID) bool {
	for _, s := range *dst {
		if s == id {
			return false
		}
	}
	*dst = append(*dst, id)
	return true
}

func addSyms(dst *[]SymbolID, ids []SymbolID) bool {
	changed := false
	for _, id := range ids {
		if addSym(dst, id) {
			changed = true
		}
	}
	return changed
}

func (c *compiler) firstSeq(syms []SymbolID, look SymbolID) []SymbolID {
	var out []SymbolID
	for _, s := range syms {
		if c.syms[s].terminal {
			return addOne(out, s)
		}
		out = addMany(out, c.first[s])
		if !c.nullable[s] {
			return out
		}
	}
	return addOne(out, look)
}

func addOne(dst []SymbolID, id SymbolID) []SymbolID {
	for _, s := range dst {
		if s == id {
			return dst
		}
	}
	return append(dst, id)
}

func addMany(dst, ids []SymbolID) []SymbolID {
	for _, id := range ids {
		dst = addOne(dst, id)
	}
	return dst
}

func (c *compiler) closure(seed []item) []item {
	in := make(map[item]struct{}, len(seed)*2)
	q := make([]item, 0, len(seed)*2)
	for _, it := range seed {
		if _, ok := in[it]; ok {
			continue
		}
		in[it] = struct{}{}
		q = append(q, it)
	}
	for head := 0; head < len(q); head++ {
		it := q[head]
		p := c.prods[it.prod]
		if int(it.dot) >= len(p.rhs) {
			continue
		}
		b := p.rhs[it.dot]
		if c.syms[b].terminal {
			continue
		}
		first := c.firstSeq(p.rhs[it.dot+1:], it.look)
		for _, prod := range c.byLHS[b] {
			for _, look := range first {
				ni := item{prod: prod, dot: 0, look: look}
				if _, ok := in[ni]; ok {
					continue
				}
				in[ni] = struct{}{}
				q = append(q, ni)
			}
		}
	}
	out := make([]item, 0, len(in))
	for it := range in {
		out = append(out, it)
	}
	sortItems(out)
	return out
}

func sortItems(items []item) {
	sort.Slice(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if a.prod != b.prod {
			return a.prod < b.prod
		}
		if a.dot != b.dot {
			return a.dot < b.dot
		}
		return a.look < b.look
	})
}

func (c *compiler) intern(items []item) uint32 {
	k := c.pack(items)
	if id, ok := c.index[k]; ok {
		return id
	}
	id := uint32(len(c.states))
	cp := make([]item, len(items))
	copy(cp, items)
	c.states = append(c.states, cp)
	c.index[k] = id
	c.edge = append(c.edge, map[SymbolID]uint32{})
	return id
}

func (c *compiler) pack(items []item) string {
	c.buf = c.buf[:0]
	var tmp [6]byte
	for _, it := range items {
		binary.LittleEndian.PutUint16(tmp[0:2], it.prod)
		binary.LittleEndian.PutUint16(tmp[2:4], it.dot)
		binary.LittleEndian.PutUint16(tmp[4:6], uint16(it.look))
		c.buf = append(c.buf, tmp[:]...)
	}
	return string(c.buf)
}
