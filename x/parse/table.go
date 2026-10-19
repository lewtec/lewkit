package parse

import (
	"bytes"
	"encoding/binary"
	"sort"

	"github.com/lewtec/lewkit/x/errs"
)

const tableMagic = "LK01"

// MarshalTable encodes the automaton. Matchers are not in the encoding.
// [Grammar.Load] binds them from the declaration.
func (l *Language) MarshalTable() ([]byte, error) {
	if l == nil {
		return nil, errs.Wrap(errs.ErrInvalid, "parse: nil language")
	}
	var buf bytes.Buffer
	buf.WriteString(tableMagic)
	if err := writeString(&buf, l.name); err != nil {
		return nil, err
	}
	if err := writeCount(&buf, len(l.syms)); err != nil {
		return nil, err
	}
	for _, sym := range l.syms {
		if err := writeString(&buf, sym.name); err != nil {
			return nil, err
		}
		buf.WriteByte(symFlags(sym))
		if sym.lexPrec != int(int16(sym.lexPrec)) {
			return nil, errs.Wrap(errs.ErrInvalid, "parse: lexical precedence does not fit")
		}
		if err := binary.Write(&buf, binary.LittleEndian, int16(sym.lexPrec)); err != nil {
			return nil, err
		}
	}
	if err := writeCount(&buf, len(l.prods)); err != nil {
		return nil, err
	}
	for _, prod := range l.prods {
		if err := writeID(&buf, prod.lhs); err != nil {
			return nil, err
		}
		if err := writeCount(&buf, len(prod.rhs)); err != nil {
			return nil, err
		}
		for _, id := range prod.rhs {
			if err := writeID(&buf, id); err != nil {
				return nil, err
			}
		}
		if prod.prec != int(int32(prod.prec)) || prod.dyn != int(int32(prod.dyn)) {
			return nil, errs.Wrap(errs.ErrInvalid, "parse: precedence does not fit")
		}
		var flags uint8
		if prod.hasPrec {
			flags |= 1
		}
		if prod.hidden {
			flags |= 4
		}
		buf.WriteByte(flags)
		buf.WriteByte(byte(prod.assoc))
		if err := binary.Write(&buf, binary.LittleEndian, int32(prod.prec)); err != nil {
			return nil, err
		}
		if err := binary.Write(&buf, binary.LittleEndian, int32(prod.dyn)); err != nil {
			return nil, err
		}
	}
	if err := writeCount(&buf, len(l.extras)); err != nil {
		return nil, err
	}
	for _, id := range l.extras {
		if err := writeID(&buf, id); err != nil {
			return nil, err
		}
	}
	if err := binary.Write(&buf, binary.LittleEndian, l.start); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, uint32(l.states)); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, uint32(l.forks)); err != nil {
		return nil, err
	}
	acts := actionCells(l)
	if err := writeCount(&buf, len(acts)); err != nil {
		return nil, err
	}
	for _, cell := range acts {
		if err := binary.Write(&buf, binary.LittleEndian, cell.state); err != nil {
			return nil, err
		}
		if err := writeID(&buf, cell.sym); err != nil {
			return nil, err
		}
		if err := writeCount(&buf, len(cell.acts)); err != nil {
			return nil, err
		}
		for _, act := range cell.acts {
			buf.WriteByte(act.kind)
			if err := binary.Write(&buf, binary.LittleEndian, act.val); err != nil {
				return nil, err
			}
		}
	}
	gotos := gotoCells(l)
	if err := writeCount(&buf, len(gotos)); err != nil {
		return nil, err
	}
	for _, cell := range gotos {
		if err := binary.Write(&buf, binary.LittleEndian, cell.state); err != nil {
			return nil, err
		}
		if err := writeID(&buf, cell.sym); err != nil {
			return nil, err
		}
		if err := binary.Write(&buf, binary.LittleEndian, cell.dest); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// Load binds matchers and the scanner from g onto a table from [Language.MarshalTable].
// The table has to name the same symbols and productions as g.
func (g *Grammar) Load(table []byte) (*Language, error) {
	c, err := g.prepareCompiler()
	if err != nil {
		return nil, err
	}
	lang := &Language{
		name:   g.name,
		syms:   c.syms,
		prods:  c.prods,
		act:    map[uint64][]action{},
		gto:    map[uint64]uint32{},
		scan:   g.scan,
		extras: append([]SymbolID(nil), g.extras...),
		byName: make(map[string]SymbolID, len(c.syms)),
	}
	for i, sym := range c.syms {
		if i == 0 || sym.name == "" {
			continue
		}
		lang.byName[sym.name] = SymbolID(i)
	}
	decoded, err := decodeTable(table)
	if err != nil {
		return nil, err
	}
	if err := decoded.matches(lang); err != nil {
		return nil, err
	}
	lang.act = decoded.act
	lang.gto = decoded.gto
	lang.start = decoded.start
	lang.states = decoded.states
	lang.forks = decoded.forks
	return lang, nil
}

type tableData struct {
	name   string
	syms   []symInfo
	prods  []prodInfo
	extras []SymbolID
	start  uint32
	states int
	forks  int
	act    map[uint64][]action
	gto    map[uint64]uint32
}

func (t *tableData) matches(lang *Language) error {
	if t.name != lang.name || len(t.syms) != len(lang.syms) || len(t.prods) != len(lang.prods) || len(t.extras) != len(lang.extras) {
		return errs.Wrap(errs.ErrInvalid, "parse: table does not match the grammar")
	}
	for i := range t.syms {
		got, want := t.syms[i], lang.syms[i]
		if got.name != want.name || symFlags(got) != symFlags(want) || got.lexPrec != want.lexPrec {
			return errs.Wrap(errs.ErrInvalid, "parse: table does not match the grammar")
		}
	}
	for i := range t.prods {
		got, want := t.prods[i], lang.prods[i]
		if got.lhs != want.lhs || got.prec != want.prec || got.hasPrec != want.hasPrec || got.assoc != want.assoc || got.dyn != want.dyn || got.hidden != want.hidden || len(got.rhs) != len(want.rhs) {
			return errs.Wrap(errs.ErrInvalid, "parse: table does not match the grammar")
		}
		for j := range got.rhs {
			if got.rhs[j] != want.rhs[j] {
				return errs.Wrap(errs.ErrInvalid, "parse: table does not match the grammar")
			}
		}
	}
	for i := range t.extras {
		if t.extras[i] != lang.extras[i] {
			return errs.Wrap(errs.ErrInvalid, "parse: table does not match the grammar")
		}
	}
	return nil
}

func symFlags(sym symInfo) uint8 {
	var flags uint8
	if sym.terminal {
		flags |= 1
	}
	if sym.named {
		flags |= 2
	}
	if sym.hidden {
		flags |= 4
	}
	if sym.extra {
		flags |= 8
	}
	if sym.immediate {
		flags |= 16
	}
	if sym.external {
		flags |= 32
	}
	return flags
}

func symFromFlags(flags uint8) symInfo {
	return symInfo{
		terminal:  flags&1 != 0,
		named:     flags&2 != 0,
		hidden:    flags&4 != 0,
		extra:     flags&8 != 0,
		immediate: flags&16 != 0,
		external:  flags&32 != 0,
	}
}

type actCell struct {
	state uint32
	sym   SymbolID
	acts  []action
}

type gtoCell struct {
	state uint32
	sym   SymbolID
	dest  uint32
}

func actionCells(l *Language) []actCell {
	out := make([]actCell, 0, len(l.act))
	for key, acts := range l.act {
		out = append(out, actCell{state: uint32(key >> 16), sym: SymbolID(key), acts: acts})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].state != out[j].state {
			return out[i].state < out[j].state
		}
		return out[i].sym < out[j].sym
	})
	return out
}

func gotoCells(l *Language) []gtoCell {
	out := make([]gtoCell, 0, len(l.gto))
	for key, dest := range l.gto {
		out = append(out, gtoCell{state: uint32(key >> 16), sym: SymbolID(key), dest: dest})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].state != out[j].state {
			return out[i].state < out[j].state
		}
		return out[i].sym < out[j].sym
	})
	return out
}

func decodeTable(table []byte) (*tableData, error) {
	if len(table) < len(tableMagic) || string(table[:len(tableMagic)]) != tableMagic {
		return nil, errs.Wrap(errs.ErrInvalid, "parse: bad table")
	}
	r := bytes.NewReader(table[len(tableMagic):])
	name, err := readString(r)
	if err != nil {
		return nil, err
	}
	nsym, err := readCount(r)
	if err != nil {
		return nil, err
	}
	syms := make([]symInfo, nsym)
	for i := range syms {
		symName, err := readString(r)
		if err != nil {
			return nil, err
		}
		flags, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		var prec int16
		if err := binary.Read(r, binary.LittleEndian, &prec); err != nil {
			return nil, err
		}
		syms[i] = symFromFlags(flags)
		syms[i].name = symName
		syms[i].lexPrec = int(prec)
	}
	nprod, err := readCount(r)
	if err != nil {
		return nil, err
	}
	prods := make([]prodInfo, nprod)
	for i := range prods {
		lhs, err := readID(r)
		if err != nil {
			return nil, err
		}
		nrhs, err := readCount(r)
		if err != nil {
			return nil, err
		}
		rhs := make([]SymbolID, nrhs)
		for j := range rhs {
			rhs[j], err = readID(r)
			if err != nil {
				return nil, err
			}
		}
		flags, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		assoc, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		var prec, dyn int32
		if err := binary.Read(r, binary.LittleEndian, &prec); err != nil {
			return nil, err
		}
		if err := binary.Read(r, binary.LittleEndian, &dyn); err != nil {
			return nil, err
		}
		prods[i] = prodInfo{
			lhs: lhs, rhs: rhs,
			prec: int(prec), hasPrec: flags&1 != 0,
			assoc:  Assoc(assoc),
			dyn:    int(dyn),
			hidden: flags&4 != 0,
		}
	}
	nextra, err := readCount(r)
	if err != nil {
		return nil, err
	}
	extras := make([]SymbolID, nextra)
	for i := range extras {
		extras[i], err = readID(r)
		if err != nil {
			return nil, err
		}
	}
	var start, states, forks uint32
	if err := binary.Read(r, binary.LittleEndian, &start); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &states); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &forks); err != nil {
		return nil, err
	}
	nact, err := readCount(r)
	if err != nil {
		return nil, err
	}
	act := map[uint64][]action{}
	gotForks := 0
	for range nact {
		var state uint32
		if err := binary.Read(r, binary.LittleEndian, &state); err != nil {
			return nil, err
		}
		sym, err := readID(r)
		if err != nil {
			return nil, err
		}
		n, err := readCount(r)
		if err != nil {
			return nil, err
		}
		acts := make([]action, n)
		for i := range acts {
			kind, err := r.ReadByte()
			if err != nil {
				return nil, err
			}
			var val uint32
			if err := binary.Read(r, binary.LittleEndian, &val); err != nil {
				return nil, err
			}
			acts[i] = action{kind: kind, val: val}
		}
		if len(acts) > 1 {
			gotForks++
		}
		act[cellKey(state, sym)] = acts
	}
	ngoto, err := readCount(r)
	if err != nil {
		return nil, err
	}
	gto := map[uint64]uint32{}
	for range ngoto {
		var state, dest uint32
		if err := binary.Read(r, binary.LittleEndian, &state); err != nil {
			return nil, err
		}
		sym, err := readID(r)
		if err != nil {
			return nil, err
		}
		if err := binary.Read(r, binary.LittleEndian, &dest); err != nil {
			return nil, err
		}
		gto[cellKey(state, sym)] = dest
	}
	if r.Len() != 0 || int(forks) != gotForks {
		return nil, errs.Wrap(errs.ErrInvalid, "parse: bad table")
	}
	return &tableData{
		name: name, syms: syms, prods: prods, extras: extras,
		start: start, states: int(states), forks: int(forks),
		act: act, gto: gto,
	}, nil
}

func writeString(buf *bytes.Buffer, text string) error {
	if err := binary.Write(buf, binary.LittleEndian, uint32(len(text))); err != nil {
		return err
	}
	_, err := buf.WriteString(text)
	return err
}

func writeCount(buf *bytes.Buffer, n int) error {
	if n < 0 || n > int(^uint32(0)) {
		return errs.Wrap(errs.ErrInvalid, "parse: table does not fit")
	}
	return binary.Write(buf, binary.LittleEndian, uint32(n))
}

func writeID(buf *bytes.Buffer, id SymbolID) error {
	return binary.Write(buf, binary.LittleEndian, uint16(id))
}

func readString(r *bytes.Reader) (string, error) {
	var n uint32
	if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
		return "", errs.Wrap(errs.ErrInvalid, "parse: bad table")
	}
	if uint32(r.Len()) < n {
		return "", errs.Wrap(errs.ErrInvalid, "parse: bad table")
	}
	b := make([]byte, n)
	if _, err := r.Read(b); err != nil {
		return "", errs.Wrap(errs.ErrInvalid, "parse: bad table")
	}
	return string(b), nil
}

func readCount(r *bytes.Reader) (int, error) {
	var n uint32
	if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
		return 0, errs.Wrap(errs.ErrInvalid, "parse: bad table")
	}
	return int(n), nil
}

func readID(r *bytes.Reader) (SymbolID, error) {
	var id uint16
	if err := binary.Read(r, binary.LittleEndian, &id); err != nil {
		return 0, errs.Wrap(errs.ErrInvalid, "parse: bad table")
	}
	return SymbolID(id), nil
}
