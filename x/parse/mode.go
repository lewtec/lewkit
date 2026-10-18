package parse

// Mode is one scanner memory. Zero is no mode.
type Mode int

// ModeRow is one transition or one guarded token.
// On fires after that token is read. Enter and Leave with Depth count a
// nesting. Without Depth, Enter sets the mode and Leave clears it.
// In emits Emit when the mode is active, Not is inactive, and When matches.
// Span reads the rest of that token and may set [LexState.At].
type ModeRow[Term interface {
	comparable
	String() string
}] struct {
	On    Term
	Enter Mode
	Leave Mode
	Depth bool
	In    Mode
	Not   Mode
	When  Pattern
	Emit  Term
	Span  func(src []byte, at int, st *LexState) (end int, ok bool)
}

// Modes is the scanner memory for one language.
type Modes[Term interface {
	comparable
	String() string
}] []ModeRow[Term]

// ModeMemory is the mode table for one parse.
type ModeMemory[Term interface {
	comparable
	String() string
}] struct {
	modes []ModeRow[Term]
	ready []readyRow[Term]
	depth map[Mode]int
	on    map[Mode]bool
}

type readyRow[Term interface {
	comparable
	String() string
}] struct {
	row   ModeRow[Term]
	match Match
}

// Remember returns the memory stored on st, creating it from modes on the
// first call.
func Remember[Term interface {
	comparable
	String() string
}](st *LexState, modes Modes[Term]) *ModeMemory[Term] {
	if st != nil {
		if mem, ok := st.Data.(*ModeMemory[Term]); ok {
			return mem
		}
	}
	mem := &ModeMemory[Term]{
		modes: append([]ModeRow[Term](nil), modes...),
		depth: map[Mode]int{},
		on:    map[Mode]bool{},
	}
	for i := range mem.modes {
		row := mem.modes[i]
		if row.Emit.String() == "" || row.When.kind == 0 {
			continue
		}
		built := row.When.build(nil)
		mem.ready = append(mem.ready, readyRow[Term]{row: row, match: func(src []byte, at int) int {
			return built.eval(src, at, 0)
		}})
	}
	if st != nil {
		st.Data = mem
	}
	return mem
}

// Note applies every On row for term.
func (m *ModeMemory[Term]) Note(term Term) {
	if m == nil || term.String() == "" {
		return
	}
	for i := range m.modes {
		row := &m.modes[i]
		if row.On != term {
			continue
		}
		if row.Enter != 0 {
			m.Enter(row.Enter, row.Depth)
		}
		if row.Leave != 0 {
			m.Leave(row.Leave, row.Depth)
		}
	}
}

// Enter sets mode, or adds one level when depth is set.
func (m *ModeMemory[Term]) Enter(mode Mode, depth bool) {
	if m == nil || mode == 0 {
		return
	}
	if depth {
		m.depth[mode]++
		return
	}
	m.on[mode] = true
}

// Leave clears mode, or removes one level when depth is set.
func (m *ModeMemory[Term]) Leave(mode Mode, depth bool) {
	if m == nil || mode == 0 {
		return
	}
	if depth {
		if m.depth[mode] > 0 {
			m.depth[mode]--
		}
		return
	}
	m.on[mode] = false
}

// Depth is the nesting of a depth mode.
func (m *ModeMemory[Term]) Depth(mode Mode) int {
	if m == nil {
		return 0
	}
	return m.depth[mode]
}

// Active reports a set flag or a depth above zero.
func (m *ModeMemory[Term]) Active(mode Mode) bool {
	if m == nil || mode == 0 {
		return false
	}
	return m.on[mode] || m.depth[mode] > 0
}

// lead reports a guarded token. matched is false when no row applies.
// ok is false when a row matches and its span fails.
func (m *ModeMemory[Term]) lead(src []byte, at int, st *LexState) (term Term, start, end int, matched, ok bool) {
	if m == nil {
		return term, at, at, false, true
	}
	for i := range m.ready {
		row := &m.ready[i].row
		if row.In != 0 && !m.Active(row.In) {
			continue
		}
		if row.Not != 0 && m.Active(row.Not) {
			continue
		}
		n := m.ready[i].match(src, at)
		if n <= 0 {
			continue
		}
		end = at + n
		if row.Span != nil {
			var spanOK bool
			end, spanOK = row.Span(src, at, st)
			if !spanOK {
				return term, at, at, true, false
			}
		}
		return row.Emit, at, end, true, true
	}
	return term, at, at, false, true
}

// Drive is the scanner. A guarded Emit row wins before read. read's term is
// then noted. An empty term at the end of src is EOF.
func Drive[Term interface {
	comparable
	String() string
}](modes Modes[Term], read func(src []byte, at int, st *LexState) (Term, int, int, bool)) ScanFunc {
	return func(src []byte, at int, _ func(SymbolID) bool, st *LexState) (SymbolID, int, int, bool) {
		if st == nil {
			return 0, at, at, false
		}
		mem := Remember(st, modes)
		if term, start, end, matched, ok := mem.lead(src, at, st); matched {
			if !ok {
				return 0, start, end, false
			}
			id, found := st.Symbol(term.String())
			if !found {
				return 0, start, end, false
			}
			return id, start, end, true
		}
		if read == nil {
			return 0, at, at, false
		}
		term, start, end, ok := read(src, at, st)
		if !ok {
			return 0, start, end, false
		}
		if term.String() == "" {
			if start >= len(src) {
				return SymEOF, start, start, true
			}
			return 0, start, end, false
		}
		mem.Note(term)
		id, found := st.Symbol(term.String())
		if !found {
			return 0, start, end, false
		}
		return id, start, end, true
	}
}
