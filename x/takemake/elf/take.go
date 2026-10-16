package elf

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

func take(r io.Reader) (*File, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFormat, err)
	}
	if len(data) < 64 || string(data[:4]) != elf.ELFMAG {
		return nil, ErrFormat
	}
	parsed, err := elf.NewFile(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFormat, err)
	}
	defer parsed.Close()

	file := &File{
		Header: Header{
			Class:   Class(parsed.Class),
			Endian:  Endian(parsed.Data),
			OSABI:   uint8(parsed.OSABI),
			ABI:     parsed.ABIVersion,
			Kind:    Kind(parsed.Type),
			Machine: Machine(parsed.Machine),
			Entry:   parsed.Entry,
		},
		Span: uint64(len(data)),
		cur:  -1,
		seg:  -1,
	}
	order := file.order()
	eh, _, sh := file.dims()
	if len(data) < eh {
		return nil, ErrFormat
	}
	file.Flags = order.Uint32(data[eh-16 : eh-12])
	switch file.Class {
	case Class32:
		file.PhOff = uint64(order.Uint32(data[28:32]))
		file.ShOff = uint64(order.Uint32(data[32:36]))
		file.ShStr = order.Uint16(data[50:52])
	case Class64:
		file.PhOff = order.Uint64(data[32:40])
		file.ShOff = order.Uint64(data[40:48])
		file.ShStr = order.Uint16(data[62:64])
	default:
		return nil, ErrFormat
	}

	file.Sections = make([]*Section, len(parsed.Sections))
	for i, section := range parsed.Sections {
		item := &Section{
			Name:    section.Name,
			NameOff: nameOff(data, order, sh, file.ShOff, i),
			Kind:    uint32(section.Type),
			Flags:   uint64(section.Flags),
			Addr:    section.Addr,
			Offset:  section.Offset,
			Size:    section.Size,
			Link:    section.Link,
			Info:    section.Info,
			Align:   section.Addralign,
			Entsize: section.Entsize,
		}
		if section.Type != elf.SHT_NOBITS && section.FileSize > 0 {
			raw, err := slice(data, section.Offset, section.FileSize)
			if err != nil {
				return nil, err
			}
			item.Bytes = raw
		}
		file.Sections[i] = item
	}
	if int(file.ShStr) >= len(file.Sections) {
		return nil, ErrFormat
	}

	if err := file.attachSymbols(parsed); err != nil {
		return nil, err
	}
	file.attachRelocs()

	file.Segments = make([]*Segment, len(parsed.Progs))
	for i, prog := range parsed.Progs {
		item := &Segment{
			Type:   uint32(prog.Type),
			Flags:  uint32(prog.Flags),
			Off:    prog.Off,
			Vaddr:  prog.Vaddr,
			Paddr:  prog.Paddr,
			Filesz: prog.Filesz,
			Memsz:  prog.Memsz,
			Align:  prog.Align,
		}
		for index, section := range file.Sections {
			if index == 0 {
				continue
			}
			if covers(item, section) {
				item.Covers = append(item.Covers, index)
			}
		}
		file.Segments[i] = item
	}
	file.loaded = true
	return file, nil
}

func nameOff(data []byte, order binary.ByteOrder, shentsize int, shoff uint64, index int) uint32 {
	off := shoff + uint64(index*shentsize)
	raw, err := slice(data, off, 4)
	if err != nil {
		return 0
	}
	return order.Uint32(raw)
}

func slice(data []byte, off, n uint64) ([]byte, error) {
	if n == 0 {
		return nil, nil
	}
	if off > uint64(len(data)) || n > uint64(len(data))-off {
		return nil, ErrFormat
	}
	return append([]byte(nil), data[off:off+n]...), nil
}

func covers(segment *Segment, section *Section) bool {
	if section == nil {
		return false
	}
	if len(section.Bytes) > 0 && overlap(section.Offset, uint64(len(section.Bytes)), segment.Off, segment.Filesz) {
		return true
	}
	if section.Flags&FlagAlloc != 0 && section.Size > 0 && overlap(section.Addr, section.Size, segment.Vaddr, segment.Memsz) {
		return true
	}
	return false
}

func overlap(start, size, other, otherSize uint64) bool {
	if size == 0 || otherSize == 0 {
		return false
	}
	return start < other+otherSize && other < start+size
}

func (f *File) attachSymbols(parsed *elf.File) error {
	static, err := parsed.Symbols()
	if err != nil && !errors.Is(err, elf.ErrNoSymbols) {
		return fmt.Errorf("%w: %v", ErrFormat, err)
	}
	f.placeSymbols(static, false)
	dynamic, err := parsed.DynamicSymbols()
	if err != nil && !errors.Is(err, elf.ErrNoSymbols) {
		return fmt.Errorf("%w: %v", ErrFormat, err)
	}
	f.placeSymbols(dynamic, true)
	return nil
}

func (f *File) placeSymbols(symbols []elf.Symbol, dynamic bool) {
	for _, symbol := range symbols {
		item := &Symbol{
			Name:    symbol.Name,
			Info:    symbol.Info,
			Other:   symbol.Other,
			Section: uint16(symbol.Section),
			Value:   symbol.Value,
			Size:    symbol.Size,
			Dynamic: dynamic,
		}
		index := int(symbol.Section)
		if symbol.Section != elf.SHN_UNDEF && symbol.Section < elf.SHN_LORESERVE && index < len(f.Sections) {
			f.Sections[index].Symbols = append(f.Sections[index].Symbols, item)
			continue
		}
		f.Symbols = append(f.Symbols, item)
	}
}

func (f *File) attachRelocs() {
	for _, section := range f.Sections {
		if section.Kind != SectRel && section.Kind != SectRela {
			continue
		}
		entsize := section.Entsize
		rela := section.Kind == SectRela
		if entsize == 0 {
			entsize = defaultEntsize(f.Class, rela)
		}
		if entsize == 0 || uint64(len(section.Bytes))%entsize != 0 {
			continue
		}
		dynamic := false
		if int(section.Link) < len(f.Sections) && f.Sections[section.Link].Kind == SectDynSym {
			dynamic = true
		}
		relocs := decodeRelocs(section.Bytes, f.order(), f.Class, rela, entsize, dynamic)
		target := int(section.Info)
		if target > 0 && target < len(f.Sections) {
			f.Sections[target].Relocs = append(f.Sections[target].Relocs, relocs...)
			continue
		}
		f.Relocs = append(f.Relocs, relocs...)
	}
}

func defaultEntsize(class Class, rela bool) uint64 {
	if class == Class64 {
		if rela {
			return 24
		}
		return 16
	}
	if rela {
		return 12
	}
	return 8
}

func decodeRelocs(raw []byte, order binary.ByteOrder, class Class, rela bool, entsize uint64, dynamic bool) []*Reloc {
	var out []*Reloc
	for off := uint64(0); off+entsize <= uint64(len(raw)); off += entsize {
		entry := raw[off : off+entsize]
		item := &Reloc{Dynamic: dynamic, HasAddend: rela}
		if class == Class64 {
			item.Offset = order.Uint64(entry[0:8])
			info := order.Uint64(entry[8:16])
			item.Sym = uint32(info >> 32)
			item.Kind = uint32(info)
			if rela && len(entry) >= 24 {
				item.Addend = int64(order.Uint64(entry[16:24]))
			}
		} else {
			item.Offset = uint64(order.Uint32(entry[0:4]))
			info := order.Uint32(entry[4:8])
			item.Sym = info >> 8
			item.Kind = info & 0xff
			if rela && len(entry) >= 12 {
				item.Addend = int64(int32(order.Uint32(entry[8:12])))
			}
		}
		out = append(out, item)
	}
	return out
}
