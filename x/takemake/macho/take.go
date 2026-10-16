package macho

import (
	"bytes"
	stdmacho "debug/macho"
	"encoding/binary"
	"fmt"
	"io"
)

func take(r io.Reader) (*File, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFormat, err)
	}
	parsed, err := stdmacho.NewFile(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFormat, err)
	}
	order := parsed.ByteOrder
	file := &File{
		Header: Header{
			Magic:  parsed.Magic,
			CPU:    CPU(parsed.Cpu),
			SubCPU: parsed.SubCpu,
			Kind:   Kind(parsed.Type),
			NCmd:   parsed.Ncmd,
			CmdSz:  parsed.Cmdsz,
			Flags:  parsed.Flags,
			Endian: endianOf(order),
		},
		Span: uint64(len(data)),
		cur:  -1,
		sec:  -1,
	}
	if parsed.Magic == stdmacho.Magic64 && len(data) >= header64 {
		file.Reserved = order.Uint32(data[28:32])
	}
	sections := make([]*Section, 0, len(parsed.Sections))
	next := 0
	for _, load := range parsed.Loads {
		raw := append([]byte(nil), load.Raw()...)
		cmd := uint32(0)
		if len(raw) >= 4 {
			cmd = order.Uint32(raw[:4])
		}
		file.Commands = append(file.Commands, Command{Cmd: cmd, Data: raw})
		if segment, ok := load.(*stdmacho.Segment); ok {
			item, err := segmentFrom(segment, data)
			if err != nil {
				return nil, err
			}
			for range int(segment.Nsect) {
				if next >= len(parsed.Sections) {
					return nil, ErrFormat
				}
				section, err := sectionFrom(parsed.Sections[next], data)
				if err != nil {
					return nil, err
				}
				next++
				item.Sections = append(item.Sections, section)
				sections = append(sections, section)
			}
			file.Segments = append(file.Segments, item)
		}
		if err := file.noteDylib(cmd, raw, order); err != nil {
			return nil, err
		}
		if err := file.noteSymtab(cmd, raw, data, order); err != nil {
			return nil, err
		}
		file.noteMain(cmd, raw, order)
		if err := file.noteDysym(cmd, raw, data, order); err != nil {
			return nil, err
		}
	}
	if parsed.Symtab != nil {
		for _, sym := range parsed.Symtab.Syms {
			item := &Symbol{
				Name:  sym.Name,
				Type:  sym.Type,
				Sect:  sym.Sect,
				Desc:  sym.Desc,
				Value: sym.Value,
			}
			if sym.Sect > 0 && int(sym.Sect) <= len(sections) {
				sections[sym.Sect-1].Symbols = append(sections[sym.Sect-1].Symbols, item)
				continue
			}
			file.Symbols = append(file.Symbols, item)
		}
	}
	file.loaded = true
	return file, nil
}

func segmentFrom(segment *stdmacho.Segment, data []byte) (*Segment, error) {
	item := &Segment{
		Name:    segment.Name,
		Addr:    segment.Addr,
		Memsz:   segment.Memsz,
		Offset:  segment.Offset,
		Filesz:  segment.Filesz,
		MaxProt: segment.Maxprot,
		Prot:    segment.Prot,
		Flag:    segment.Flag,
	}
	raw, err := sliceCopy(data, segment.Offset, segment.Filesz)
	if err != nil {
		return nil, err
	}
	item.Bytes = raw
	return item, nil
}

func sectionFrom(section *stdmacho.Section, data []byte) (*Section, error) {
	item := &Section{
		Name:   section.Name,
		Seg:    section.Seg,
		Addr:   section.Addr,
		Size:   section.Size,
		Offset: section.Offset,
		Align:  section.Align,
		Reloff: section.Reloff,
		Nreloc: section.Nreloc,
		Flags:  section.Flags,
	}
	if zerofill(section.Flags) || section.Size == 0 {
		return item, nil
	}
	raw, err := sliceCopy(data, uint64(section.Offset), section.Size)
	if err != nil {
		return nil, err
	}
	item.Bytes = raw
	return item, nil
}

func zerofill(flags uint32) bool {
	switch flags & sectionTypeMask {
	case sectionZerofill, sectionGBZerofill, sectionTLSZerofill:
		return true
	default:
		return false
	}
}

func (f *File) noteDylib(cmd uint32, raw []byte, order binary.ByteOrder) error {
	if !dylibCmd(cmd) || len(raw) < dylibHead {
		return nil
	}
	nameAt := order.Uint32(raw[8:12])
	if nameAt >= uint32(len(raw)) {
		return ErrFormat
	}
	f.Dylibs = append(f.Dylibs, &Dylib{
		Cmd:     cmd,
		Name:    cstr(raw[nameAt:]),
		Time:    order.Uint32(raw[12:16]),
		Current: order.Uint32(raw[16:20]),
		Compat:  order.Uint32(raw[20:24]),
	})
	return nil
}

func dylibCmd(cmd uint32) bool {
	switch cmd {
	case lcLoadDylib, lcIDDylib, lcLazyDylib, lcWeakDylib, lcReexportDylib, lcUpwardDylib:
		return true
	default:
		return false
	}
}

func (f *File) noteSymtab(cmd uint32, raw, data []byte, order binary.ByteOrder) error {
	if cmd != lcSymtab || len(raw) < symtabCmdSize {
		return nil
	}
	symAt := order.Uint32(raw[8:12])
	nsyms := order.Uint32(raw[12:16])
	strAt := order.Uint32(raw[16:20])
	strSize := order.Uint32(raw[20:24])
	width := uint64(nlist32Size)
	if f.Magic == magic64 || f.Magic == stdmacho.Magic64 {
		width = nlist64Size
	}
	sym, err := sliceCopy(data, uint64(symAt), uint64(nsyms)*width)
	if err != nil {
		return err
	}
	str, err := sliceCopy(data, uint64(strAt), uint64(strSize))
	if err != nil {
		return err
	}
	f.SymOff = symAt
	f.StrOff = strAt
	f.SymBytes = sym
	f.StrBytes = str
	return nil
}

func (f *File) noteMain(cmd uint32, raw []byte, order binary.ByteOrder) {
	if cmd != lcMain || len(raw) < 16 {
		return
	}
	f.EntryOff = order.Uint64(raw[8:16])
	f.entry = true
	if len(raw) >= mainCmdSize {
		f.Stack = order.Uint64(raw[16:24])
	}
}

func (f *File) noteDysym(cmd uint32, raw, data []byte, order binary.ByteOrder) error {
	if cmd != lcDysymtab || len(raw) < dysymtabSize {
		return nil
	}
	at := order.Uint32(raw[56:60])
	n := order.Uint32(raw[60:64])
	indirect, err := sliceCopy(data, uint64(at), uint64(n)*4)
	if err != nil {
		return err
	}
	f.IndirectAt = at
	f.Indirect = indirect
	return nil
}

func endianOf(order binary.ByteOrder) Endian {
	if order == binary.BigEndian {
		return Big
	}
	return Little
}

func cstr(raw []byte) string {
	if i := bytes.IndexByte(raw, 0); i >= 0 {
		return string(raw[:i])
	}
	return string(raw)
}

func sliceCopy(data []byte, off, n uint64) ([]byte, error) {
	if n == 0 || off >= uint64(len(data)) {
		return nil, nil
	}
	if n > uint64(len(data))-off {
		n = uint64(len(data)) - off
	}
	return append([]byte(nil), data[off:off+n]...), nil
}
