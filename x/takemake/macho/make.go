package macho

import (
	"encoding/binary"
	"io"
)

func (f *File) emit(w io.Writer) error {
	if err := f.prepare(); err != nil {
		return err
	}
	buf, err := f.image()
	if err != nil {
		return err
	}
	_, err = w.Write(buf)
	return err
}

func (f *File) prepare() error {
	if f.loaded {
		return nil
	}
	if f.Endian == 0 {
		f.Endian = Little
	}
	if f.Header.CPU == 0 {
		f.Header.CPU = Amd64
	}
	if f.Magic == 0 {
		if f.Header.CPU&CPU(cpuArch64) != 0 {
			f.Magic = magic64
		} else {
			f.Magic = magic32
		}
	}
	if f.Kind == 0 {
		f.Kind = TypeObj
	}
	if f.Kind == TypeDylib && f.idDylib() == nil {
		id := &Dylib{Cmd: lcIDDylib, Name: "lib.dylib", Current: 0x10000, Compat: 0x10000}
		f.Dylibs = append([]*Dylib{id}, f.Dylibs...)
	}

	wide := f.wide()
	hdr := header32
	segSize := segment32Size
	secSize := section32Size
	nlist := nlist32Size
	if wide {
		hdr = header64
		segSize = segment64Size
		secSize = section64Size
		nlist = nlist64Size
	}
	hasSym := f.hasSymbol()
	emitMain := f.entry || f.Kind == TypeExec || f.Kind == TypeDylib
	cmdSize := 0
	for _, seg := range f.Segments {
		cmdSize += segSize + len(seg.Sections)*secSize
	}
	if hasSym {
		cmdSize += segSize + symtabCmdSize
	}
	for _, item := range f.Dylibs {
		cmdSize += dylibSize(item.Name)
	}
	if emitMain {
		cmdSize += mainCmdSize
		f.entry = true
	}

	off := alignUp(uint32(hdr+cmdSize), 8)
	var vm uint64
	for _, seg := range f.Segments {
		if seg.MaxProt == 0 && seg.Prot == 0 {
			seg.MaxProt = 7
			seg.Prot = 7
		}
		seg.Offset = uint64(off)
		start := off
		addr := seg.Addr
		for _, section := range seg.Sections {
			section.Seg = seg.Name
			section.Offset = off
			section.Size = uint64(len(section.Bytes))
			section.Addr = addr
			off += uint32(section.Size)
			addr += section.Size
		}
		seg.Filesz = uint64(off - start)
		if seg.Memsz < seg.Filesz {
			seg.Memsz = seg.Filesz
		}
		if seg.Filesz == 0 {
			seg.Bytes = nil
		} else {
			seg.Bytes = make([]byte, seg.Filesz)
			for _, section := range seg.Sections {
				rel := uint64(section.Offset) - seg.Offset
				copy(seg.Bytes[rel:], section.Bytes)
			}
		}
		if end := seg.Addr + seg.Memsz; end > vm {
			vm = end
		}
	}

	plan := f.symbolPlan()
	if len(plan) > 0 {
		sym, str := encodeSymbols(plan, nlist, f.order())
		f.SymOff = off
		f.SymBytes = sym
		off += uint32(len(sym))
		f.StrOff = off
		f.StrBytes = str
		off += uint32(len(str))
		blob := append(append([]byte(nil), sym...), str...)
		f.Segments = append(f.Segments, &Segment{
			Name:    "__LINKEDIT",
			Addr:    vm,
			Memsz:   uint64(len(blob)),
			Offset:  uint64(f.SymOff),
			Filesz:  uint64(len(blob)),
			MaxProt: 1,
			Prot:    1,
			Bytes:   blob,
		})
	}

	f.Commands = f.Commands[:0]
	for _, seg := range f.Segments {
		raw := encodeSegment(seg, wide, f.order())
		cmd := lcSegment
		if wide {
			cmd = lcSegment64
		}
		f.Commands = append(f.Commands, Command{Cmd: cmd, Data: raw})
	}
	if len(plan) > 0 {
		raw := encodeSymtab(f.SymOff, uint32(len(plan)), f.StrOff, uint32(len(f.StrBytes)), f.order())
		f.Commands = append(f.Commands, Command{Cmd: lcSymtab, Data: raw})
	}
	for _, item := range f.Dylibs {
		raw := encodeDylib(item, f.order())
		f.Commands = append(f.Commands, Command{Cmd: item.Cmd, Data: raw})
	}
	if emitMain {
		raw := encodeMain(f.EntryOff, f.Stack, f.order())
		f.Commands = append(f.Commands, Command{Cmd: lcMain, Data: raw})
	}
	got := 0
	for _, cmd := range f.Commands {
		got += len(cmd.Data)
	}
	f.NCmd = uint32(len(f.Commands))
	f.CmdSz = uint32(got)
	f.Span = uint64(off)
	if uint64(hdr+got) > f.Span {
		f.Span = uint64(hdr + got)
	}
	return nil
}

type symRef struct {
	sym  *Symbol
	sect uint8
	addr uint64
}

func (f *File) hasSymbol() bool {
	if len(f.Symbols) > 0 {
		return true
	}
	for _, seg := range f.Segments {
		for _, section := range seg.Sections {
			if len(section.Symbols) > 0 {
				return true
			}
		}
	}
	return false
}

func (f *File) symbolPlan() []symRef {
	var out []symRef
	var n uint8 = 1
	for _, seg := range f.Segments {
		for _, section := range seg.Sections {
			for _, sym := range section.Symbols {
				out = append(out, symRef{sym: sym, sect: n, addr: section.Addr})
			}
			n++
		}
	}
	for _, sym := range f.Symbols {
		out = append(out, symRef{sym: sym, sect: sym.Sect, addr: sym.Value})
	}
	return out
}

func encodeSymbols(plan []symRef, nlist int, order binary.ByteOrder) ([]byte, []byte) {
	str := []byte{0}
	buf := make([]byte, len(plan)*nlist)
	for i, item := range plan {
		at := uint32(len(str))
		str = append(str, item.sym.Name...)
		str = append(str, 0)
		entry := buf[i*nlist : (i+1)*nlist]
		order.PutUint32(entry[0:4], at)
		kind := item.sym.Type
		if kind == 0 {
			kind = symSectExt
			if item.sect == 0 {
				kind = symExternal
			}
		}
		entry[4] = kind
		entry[5] = item.sect
		order.PutUint16(entry[6:8], item.sym.Desc)
		val := item.sym.Value
		if val == 0 {
			val = item.addr
		}
		if nlist == nlist64Size {
			order.PutUint64(entry[8:16], val)
			continue
		}
		order.PutUint32(entry[8:12], uint32(val))
	}
	return buf, str
}

func encodeSegment(seg *Segment, wide bool, order binary.ByteOrder) []byte {
	nsect := len(seg.Sections)
	size := segment32Size + nsect*section32Size
	cmd := lcSegment
	if wide {
		size = segment64Size + nsect*section64Size
		cmd = lcSegment64
	}
	buf := make([]byte, size)
	order.PutUint32(buf[0:4], cmd)
	order.PutUint32(buf[4:8], uint32(size))
	copy(buf[8:24], seg.Name)
	if wide {
		order.PutUint64(buf[24:32], seg.Addr)
		order.PutUint64(buf[32:40], seg.Memsz)
		order.PutUint64(buf[40:48], seg.Offset)
		order.PutUint64(buf[48:56], seg.Filesz)
		order.PutUint32(buf[56:60], seg.MaxProt)
		order.PutUint32(buf[60:64], seg.Prot)
		order.PutUint32(buf[64:68], uint32(nsect))
		order.PutUint32(buf[68:72], seg.Flag)
		at := segment64Size
		for _, section := range seg.Sections {
			putSection(buf[at:at+section64Size], section, true, order)
			at += section64Size
		}
		return buf
	}
	order.PutUint32(buf[24:28], uint32(seg.Addr))
	order.PutUint32(buf[28:32], uint32(seg.Memsz))
	order.PutUint32(buf[32:36], uint32(seg.Offset))
	order.PutUint32(buf[36:40], uint32(seg.Filesz))
	order.PutUint32(buf[40:44], seg.MaxProt)
	order.PutUint32(buf[44:48], seg.Prot)
	order.PutUint32(buf[48:52], uint32(nsect))
	order.PutUint32(buf[52:56], seg.Flag)
	at := segment32Size
	for _, section := range seg.Sections {
		putSection(buf[at:at+section32Size], section, false, order)
		at += section32Size
	}
	return buf
}

func putSection(buf []byte, section *Section, wide bool, order binary.ByteOrder) {
	copy(buf[0:16], section.Name)
	copy(buf[16:32], section.Seg)
	if wide {
		order.PutUint64(buf[32:40], section.Addr)
		order.PutUint64(buf[40:48], section.Size)
		order.PutUint32(buf[48:52], section.Offset)
		order.PutUint32(buf[52:56], section.Align)
		order.PutUint32(buf[56:60], section.Reloff)
		order.PutUint32(buf[60:64], section.Nreloc)
		order.PutUint32(buf[64:68], section.Flags)
		return
	}
	order.PutUint32(buf[32:36], uint32(section.Addr))
	order.PutUint32(buf[36:40], uint32(section.Size))
	order.PutUint32(buf[40:44], section.Offset)
	order.PutUint32(buf[44:48], section.Align)
	order.PutUint32(buf[48:52], section.Reloff)
	order.PutUint32(buf[52:56], section.Nreloc)
	order.PutUint32(buf[56:60], section.Flags)
}

func encodeSymtab(symAt, nsyms, strAt, strSize uint32, order binary.ByteOrder) []byte {
	buf := make([]byte, symtabCmdSize)
	order.PutUint32(buf[0:4], lcSymtab)
	order.PutUint32(buf[4:8], symtabCmdSize)
	order.PutUint32(buf[8:12], symAt)
	order.PutUint32(buf[12:16], nsyms)
	order.PutUint32(buf[16:20], strAt)
	order.PutUint32(buf[20:24], strSize)
	return buf
}

func encodeDylib(item *Dylib, order binary.ByteOrder) []byte {
	size := dylibSize(item.Name)
	buf := make([]byte, size)
	cmd := item.Cmd
	if cmd == 0 {
		cmd = lcLoadDylib
	}
	order.PutUint32(buf[0:4], cmd)
	order.PutUint32(buf[4:8], uint32(size))
	order.PutUint32(buf[8:12], dylibHead)
	order.PutUint32(buf[12:16], item.Time)
	current := item.Current
	compat := item.Compat
	if current == 0 {
		current = 0x10000
	}
	if compat == 0 {
		compat = 0x10000
	}
	order.PutUint32(buf[16:20], current)
	order.PutUint32(buf[20:24], compat)
	copy(buf[dylibHead:], item.Name)
	return buf
}

func encodeMain(entry, stack uint64, order binary.ByteOrder) []byte {
	buf := make([]byte, mainCmdSize)
	order.PutUint32(buf[0:4], lcMain)
	order.PutUint32(buf[4:8], mainCmdSize)
	order.PutUint64(buf[8:16], entry)
	order.PutUint64(buf[16:24], stack)
	return buf
}

func dylibSize(name string) int {
	return int(alignUp(uint32(dylibHead+len(name)+1), 8))
}

func (f *File) image() ([]byte, error) {
	buf := make([]byte, f.Span)
	for _, seg := range f.Segments {
		if err := copyAt(buf, seg.Offset, seg.Bytes); err != nil {
			return nil, err
		}
	}
	if err := copyAt(buf, uint64(f.SymOff), f.SymBytes); err != nil {
		return nil, err
	}
	if err := copyAt(buf, uint64(f.StrOff), f.StrBytes); err != nil {
		return nil, err
	}
	if err := copyAt(buf, uint64(f.IndirectAt), f.Indirect); err != nil {
		return nil, err
	}
	for _, seg := range f.Segments {
		for _, section := range seg.Sections {
			if section.Offset == 0 || len(section.Bytes) == 0 {
				continue
			}
			if err := copyAt(buf, uint64(section.Offset), section.Bytes); err != nil {
				return nil, err
			}
		}
	}
	if err := f.putHeader(buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func (f *File) putHeader(buf []byte) error {
	hdr := header32
	if f.wide() {
		hdr = header64
	}
	need := hdr
	for _, cmd := range f.Commands {
		need += len(cmd.Data)
	}
	if need > len(buf) {
		return ErrFormat
	}
	order := f.order()
	order.PutUint32(buf[0:4], f.Magic)
	order.PutUint32(buf[4:8], uint32(f.Header.CPU))
	order.PutUint32(buf[8:12], f.SubCPU)
	order.PutUint32(buf[12:16], uint32(f.Kind))
	order.PutUint32(buf[16:20], f.NCmd)
	order.PutUint32(buf[20:24], f.CmdSz)
	order.PutUint32(buf[24:28], f.Flags)
	if hdr == header64 {
		order.PutUint32(buf[28:32], f.Reserved)
	}
	off := hdr
	for _, cmd := range f.Commands {
		copy(buf[off:], cmd.Data)
		off += len(cmd.Data)
	}
	return nil
}

func (f *File) order() binary.ByteOrder {
	if f.Endian == Big {
		return binary.BigEndian
	}
	return binary.LittleEndian
}

func copyAt(buf []byte, off uint64, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if off > uint64(len(buf)) || uint64(len(data)) > uint64(len(buf))-off {
		return ErrFormat
	}
	copy(buf[off:], data)
	return nil
}

func alignUp(value, align uint32) uint32 {
	if align == 0 {
		return value
	}
	return (value + align - 1) &^ (align - 1)
}

func (f *File) clone() *File {
	out := *f
	out.bad = nil
	out.Commands = make([]Command, len(f.Commands))
	for i, cmd := range f.Commands {
		cmd.Data = append([]byte(nil), cmd.Data...)
		out.Commands[i] = cmd
	}
	out.Segments = make([]*Segment, len(f.Segments))
	for i, seg := range f.Segments {
		item := *seg
		item.Bytes = append([]byte(nil), seg.Bytes...)
		item.Sections = make([]*Section, len(seg.Sections))
		for j, section := range seg.Sections {
			sec := *section
			sec.Bytes = append([]byte(nil), section.Bytes...)
			sec.Symbols = make([]*Symbol, len(section.Symbols))
			for k, sym := range section.Symbols {
				next := *sym
				sec.Symbols[k] = &next
			}
			item.Sections[j] = &sec
		}
		out.Segments[i] = &item
	}
	out.Symbols = make([]*Symbol, len(f.Symbols))
	for i, sym := range f.Symbols {
		next := *sym
		out.Symbols[i] = &next
	}
	out.Dylibs = make([]*Dylib, len(f.Dylibs))
	for i, item := range f.Dylibs {
		next := *item
		out.Dylibs[i] = &next
	}
	out.SymBytes = append([]byte(nil), f.SymBytes...)
	out.StrBytes = append([]byte(nil), f.StrBytes...)
	out.Indirect = append([]byte(nil), f.Indirect...)
	return &out
}
