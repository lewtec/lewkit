package elf

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
	if f.Class != Class32 && f.Class != Class64 {
		f.Class = Class64
	}
	if f.Endian != Little && f.Endian != Big {
		f.Endian = Little
	}
	if len(f.Sections) == 0 || f.Sections[0].Name != "" || f.Sections[0].Kind != SectNone {
		f.Sections = append([]*Section{{}}, f.Sections...)
	}
	if err := f.synthesizeSymbols(); err != nil {
		return err
	}
	f.synthesizeRelocs()
	f.synthesizeNames()
	f.place()
	return nil
}

func (f *File) synthesizeSymbols() error {
	if f.hasKind(SectSymTab) || !f.hasStaticSymbols() {
		return nil
	}
	base := len(f.Sections)
	strtab, symtab := f.buildSymtab(false)
	symtab.Link = uint32(base)
	f.Sections = append(f.Sections, strtab, symtab)
	return nil
}

func (f *File) hasStaticSymbols() bool {
	for _, section := range f.Sections {
		for _, symbol := range section.Symbols {
			if !symbol.Dynamic {
				return true
			}
		}
	}
	for _, symbol := range f.Symbols {
		if !symbol.Dynamic {
			return true
		}
	}
	return false
}

func (f *File) hasKind(kind uint32) bool {
	for _, section := range f.Sections {
		if section.Kind == kind {
			return true
		}
	}
	return false
}

type symRef struct {
	symbol *Symbol
	shndx  uint16
}

func (f *File) buildSymtab(dynamic bool) (*Section, *Section) {
	refs := []symRef{{}}
	for index, section := range f.Sections {
		for _, symbol := range section.Symbols {
			if symbol.Dynamic != dynamic {
				continue
			}
			shndx := symbol.Section
			if shndx == 0 {
				shndx = uint16(index)
			}
			refs = append(refs, symRef{symbol: symbol, shndx: shndx})
		}
	}
	for _, symbol := range f.Symbols {
		if symbol.Dynamic != dynamic {
			continue
		}
		shndx := symbol.Section
		refs = append(refs, symRef{symbol: symbol, shndx: shndx})
	}

	names := []byte{0}
	offset := map[string]uint32{"": 0}
	for _, ref := range refs {
		if ref.symbol == nil {
			continue
		}
		name := ref.symbol.Name
		if _, ok := offset[name]; ok {
			continue
		}
		offset[name] = uint32(len(names))
		names = append(names, name...)
		names = append(names, 0)
	}

	str := &Section{
		Name:  ".strtab",
		Kind:  SectStrTab,
		Align: 1,
		Bytes: names,
		Size:  uint64(len(names)),
	}
	if dynamic {
		str.Name = ".dynstr"
	}
	entsize := uint64(24)
	align := uint64(8)
	if f.Class == Class32 {
		entsize = 16
		align = 4
	}
	body := make([]byte, int(entsize)*len(refs))
	order := f.order()
	for i, ref := range refs {
		if ref.symbol == nil {
			continue
		}
		entry := body[i*int(entsize) : (i+1)*int(entsize)]
		name := offset[ref.symbol.Name]
		if f.Class == Class64 {
			order.PutUint32(entry[0:4], name)
			entry[4] = ref.symbol.Info
			entry[5] = ref.symbol.Other
			order.PutUint16(entry[6:8], ref.shndx)
			order.PutUint64(entry[8:16], ref.symbol.Value)
			order.PutUint64(entry[16:24], ref.symbol.Size)
			continue
		}
		order.PutUint32(entry[0:4], name)
		order.PutUint32(entry[4:8], uint32(ref.symbol.Value))
		order.PutUint32(entry[8:12], uint32(ref.symbol.Size))
		entry[12] = ref.symbol.Info
		entry[13] = ref.symbol.Other
		order.PutUint16(entry[14:16], ref.shndx)
	}
	info := uint32(0)
	for i, ref := range refs {
		bind := uint8(0)
		if ref.symbol != nil {
			bind = ref.symbol.Info >> 4
		}
		if bind != BindLocal {
			info = uint32(i)
			break
		}
		info = uint32(i + 1)
	}
	kind := SectSymTab
	name := ".symtab"
	if dynamic {
		kind = SectDynSym
		name = ".dynsym"
	}
	sym := &Section{
		Name:    name,
		Kind:    kind,
		Align:   align,
		Entsize: entsize,
		Info:    info,
		Bytes:   body,
		Size:    uint64(len(body)),
	}
	return str, sym
}

func (f *File) synthesizeRelocs() {
	if f.hasKind(SectRela) || f.hasKind(SectRel) {
		return
	}
	strIndex, symIndex := 0, 0
	for i, section := range f.Sections {
		switch section.Kind {
		case SectStrTab:
			if section.Name == ".strtab" {
				strIndex = i
			}
		case SectSymTab:
			symIndex = i
		}
	}
	if symIndex == 0 {
		return
	}
	f.Sections[symIndex].Link = uint32(strIndex)
	names := f.symNames(false)
	var extra []*Section
	for index, section := range f.Sections {
		if len(section.Relocs) == 0 {
			continue
		}
		body := f.encodeRelocs(section.Relocs, names)
		extra = append(extra, &Section{
			Name:    ".rela" + section.Name,
			Kind:    SectRela,
			Align:   8,
			Link:    uint32(symIndex),
			Info:    uint32(index),
			Entsize: defaultEntsize(f.Class, true),
			Bytes:   body,
			Size:    uint64(len(body)),
		})
	}
	f.Sections = append(f.Sections, extra...)
}

func (f *File) symNames(dynamic bool) map[string]uint32 {
	names := map[string]uint32{}
	var index uint32 = 1
	for _, section := range f.Sections {
		for _, symbol := range section.Symbols {
			if symbol.Dynamic != dynamic {
				continue
			}
			if _, ok := names[symbol.Name]; ok {
				index++
				continue
			}
			names[symbol.Name] = index
			index++
		}
	}
	return names
}

func (f *File) encodeRelocs(relocs []*Reloc, names map[string]uint32) []byte {
	entsize := defaultEntsize(f.Class, true)
	body := make([]byte, int(entsize)*len(relocs))
	order := f.order()
	for i, reloc := range relocs {
		entry := body[i*int(entsize) : (i+1)*int(entsize)]
		sym := reloc.Sym
		if sym == 0 && reloc.SymName != "" {
			sym = names[reloc.SymName]
		}
		if f.Class == Class64 {
			order.PutUint64(entry[0:8], reloc.Offset)
			info := uint64(sym)<<32 | uint64(reloc.Kind)
			order.PutUint64(entry[8:16], info)
			order.PutUint64(entry[16:24], uint64(reloc.Addend))
			continue
		}
		order.PutUint32(entry[0:4], uint32(reloc.Offset))
		info := sym<<8 | reloc.Kind&0xff
		order.PutUint32(entry[4:8], info)
		order.PutUint32(entry[8:12], uint32(int32(reloc.Addend)))
	}
	return body
}

func (f *File) synthesizeNames() {
	for _, section := range f.Sections {
		if section.Name == ".shstrtab" {
			f.ShStr = 0
			for i, item := range f.Sections {
				if item == section {
					f.ShStr = uint16(i)
				}
			}
			return
		}
	}
	section := &Section{Name: ".shstrtab", Kind: SectStrTab, Align: 1}
	f.Sections = append(f.Sections, section)
	f.ShStr = uint16(len(f.Sections) - 1)
	buf := []byte{0}
	for _, item := range f.Sections {
		if item.Name == "" {
			item.NameOff = 0
			continue
		}
		item.NameOff = uint32(len(buf))
		buf = append(buf, item.Name...)
		buf = append(buf, 0)
	}
	section.Bytes = buf
	section.Size = uint64(len(buf))
}

func (f *File) place() {
	eh, ph, sh := f.dims()
	off := uint64(eh)
	if len(f.Segments) > 0 {
		f.PhOff = off
		off += uint64(len(f.Segments)) * uint64(ph)
	}
	for _, section := range f.Sections {
		if len(section.Bytes) == 0 {
			section.Offset = 0
			continue
		}
		align := section.Align
		if align == 0 {
			align = 1
		}
		if align&(align-1) == 0 {
			off = (off + align - 1) &^ (align - 1)
		}
		section.Offset = off
		off += uint64(len(section.Bytes))
	}
	off = (off + 7) &^ 7
	f.ShOff = off
	f.Span = off + uint64(len(f.Sections))*uint64(sh)
	for _, segment := range f.Segments {
		if segment.Off != 0 || segment.Filesz != 0 {
			continue
		}
		var min uint64
		var max uint64
		set := false
		for _, index := range segment.Covers {
			if index < 0 || index >= len(f.Sections) {
				continue
			}
			section := f.Sections[index]
			if len(section.Bytes) == 0 {
				continue
			}
			end := section.Offset + uint64(len(section.Bytes))
			if !set || section.Offset < min {
				min = section.Offset
			}
			if end > max {
				max = end
			}
			set = true
		}
		if set {
			segment.Off = min
			segment.Filesz = max - min
		}
		if segment.Memsz == 0 {
			segment.Memsz = segment.Filesz
		}
		if segment.Align == 0 {
			segment.Align = 1
		}
	}
}

func (f *File) image() ([]byte, error) {
	if f.Span == 0 || f.ShOff == 0 {
		return nil, ErrFormat
	}
	eh, ph, sh := f.dims()
	need := f.ShOff + uint64(len(f.Sections))*uint64(sh)
	span := f.Span
	if need > span {
		span = need
	}
	buf := make([]byte, span)
	for _, section := range f.Sections {
		if len(section.Bytes) == 0 {
			continue
		}
		end := section.Offset + uint64(len(section.Bytes))
		if section.Offset >= uint64(len(buf)) || end > uint64(len(buf)) {
			return nil, ErrFormat
		}
		copy(buf[section.Offset:end], section.Bytes)
	}
	order := f.order()
	if len(f.Segments) > 0 {
		for i, segment := range f.Segments {
			off := f.PhOff + uint64(i*ph)
			if off+uint64(ph) > uint64(len(buf)) {
				return nil, ErrFormat
			}
			entry := buf[off : off+uint64(ph)]
			f.putProg(entry, order, segment)
		}
	}
	for i, section := range f.Sections {
		off := f.ShOff + uint64(i*sh)
		if off+uint64(sh) > uint64(len(buf)) {
			return nil, ErrFormat
		}
		f.putSect(buf[off:off+uint64(sh)], order, section)
	}
	if eh > len(buf) {
		return nil, ErrFormat
	}
	f.putEhdr(buf[:eh], order, eh, ph, sh)
	return buf, nil
}

func (f *File) putEhdr(buf []byte, order binary.ByteOrder, eh, ph, sh int) {
	buf[0], buf[1], buf[2], buf[3] = 0x7f, 'E', 'L', 'F'
	buf[4] = byte(f.Class)
	buf[5] = byte(f.Endian)
	buf[6] = 1
	buf[7] = f.OSABI
	buf[8] = f.ABI
	order.PutUint16(buf[16:18], uint16(f.Kind))
	order.PutUint16(buf[18:20], uint16(f.Header.Machine))
	order.PutUint32(buf[20:24], 1)
	if f.Class == Class64 {
		order.PutUint64(buf[24:32], f.Header.Entry)
		order.PutUint64(buf[32:40], f.PhOff)
		order.PutUint64(buf[40:48], f.ShOff)
		order.PutUint32(buf[48:52], f.Flags)
	} else {
		order.PutUint32(buf[24:28], uint32(f.Header.Entry))
		order.PutUint32(buf[28:32], uint32(f.PhOff))
		order.PutUint32(buf[32:36], uint32(f.ShOff))
		order.PutUint32(buf[36:40], f.Flags)
	}
	tail := eh - 12
	order.PutUint16(buf[tail:tail+2], uint16(eh))
	order.PutUint16(buf[tail+2:tail+4], uint16(ph))
	order.PutUint16(buf[tail+4:tail+6], uint16(len(f.Segments)))
	order.PutUint16(buf[tail+6:tail+8], uint16(sh))
	order.PutUint16(buf[tail+8:tail+10], uint16(len(f.Sections)))
	order.PutUint16(buf[tail+10:tail+12], f.ShStr)
}

func (f *File) putProg(buf []byte, order binary.ByteOrder, segment *Segment) {
	if f.Class == Class64 {
		order.PutUint32(buf[0:4], segment.Type)
		order.PutUint32(buf[4:8], segment.Flags)
		order.PutUint64(buf[8:16], segment.Off)
		order.PutUint64(buf[16:24], segment.Vaddr)
		order.PutUint64(buf[24:32], segment.Paddr)
		order.PutUint64(buf[32:40], segment.Filesz)
		order.PutUint64(buf[40:48], segment.Memsz)
		order.PutUint64(buf[48:56], segment.Align)
		return
	}
	order.PutUint32(buf[0:4], segment.Type)
	order.PutUint32(buf[4:8], uint32(segment.Off))
	order.PutUint32(buf[8:12], uint32(segment.Vaddr))
	order.PutUint32(buf[12:16], uint32(segment.Paddr))
	order.PutUint32(buf[16:20], uint32(segment.Filesz))
	order.PutUint32(buf[20:24], uint32(segment.Memsz))
	order.PutUint32(buf[24:28], segment.Flags)
	order.PutUint32(buf[28:32], uint32(segment.Align))
}

func (f *File) putSect(buf []byte, order binary.ByteOrder, section *Section) {
	size := section.Size
	if size == 0 {
		size = uint64(len(section.Bytes))
	}
	order.PutUint32(buf[0:4], section.NameOff)
	order.PutUint32(buf[4:8], section.Kind)
	if f.Class == Class64 {
		order.PutUint64(buf[8:16], section.Flags)
		order.PutUint64(buf[16:24], section.Addr)
		order.PutUint64(buf[24:32], section.Offset)
		order.PutUint64(buf[32:40], size)
		order.PutUint32(buf[40:44], section.Link)
		order.PutUint32(buf[44:48], section.Info)
		order.PutUint64(buf[48:56], section.Align)
		order.PutUint64(buf[56:64], section.Entsize)
		return
	}
	order.PutUint32(buf[8:12], uint32(section.Flags))
	order.PutUint32(buf[12:16], uint32(section.Addr))
	order.PutUint32(buf[16:20], uint32(section.Offset))
	order.PutUint32(buf[20:24], uint32(size))
	order.PutUint32(buf[24:28], section.Link)
	order.PutUint32(buf[28:32], section.Info)
	order.PutUint32(buf[32:36], uint32(section.Align))
	order.PutUint32(buf[36:40], uint32(section.Entsize))
}

func (f *File) clone() *File {
	if f == nil {
		return nil
	}
	out := *f
	out.sym = nil
	out.bad = nil
	out.Sections = make([]*Section, len(f.Sections))
	for i, section := range f.Sections {
		out.Sections[i] = cloneSection(section)
	}
	out.Segments = make([]*Segment, len(f.Segments))
	for i, segment := range f.Segments {
		item := *segment
		item.Covers = append([]int(nil), segment.Covers...)
		out.Segments[i] = &item
	}
	out.Symbols = cloneSymbols(f.Symbols)
	out.Relocs = cloneRelocs(f.Relocs)
	return &out
}

func cloneSection(section *Section) *Section {
	if section == nil {
		return nil
	}
	out := *section
	out.Bytes = append([]byte(nil), section.Bytes...)
	out.Symbols = cloneSymbols(section.Symbols)
	out.Relocs = cloneRelocs(section.Relocs)
	return &out
}

func cloneSymbols(symbols []*Symbol) []*Symbol {
	if len(symbols) == 0 {
		return nil
	}
	out := make([]*Symbol, len(symbols))
	for i, symbol := range symbols {
		item := *symbol
		out[i] = &item
	}
	return out
}

func cloneRelocs(relocs []*Reloc) []*Reloc {
	if len(relocs) == 0 {
		return nil
	}
	out := make([]*Reloc, len(relocs))
	for i, reloc := range relocs {
		item := *reloc
		out[i] = &item
	}
	return out
}
