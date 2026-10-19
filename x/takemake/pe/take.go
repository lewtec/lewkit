package pe

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"io"
)

func take(r io.Reader) (*File, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFormat, err)
	}
	parsed, err := pe.NewFile(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFormat, err)
	}
	defer parsed.Close()

	file := &File{
		Header: Header{
			Machine:         Machine(parsed.Machine),
			TimeDateStamp:   parsed.TimeDateStamp,
			Characteristics: parsed.Characteristics,
			SymbolCount:     parsed.NumberOfSymbols,
			SymbolOff:       parsed.PointerToSymbolTable,
			Optional:        parsed.SizeOfOptionalHeader,
		},
		Span: uint64(len(data)),
		cur:  -1,
		imp:  -1,
		exp:  -1,
	}
	if len(data) >= 0x40 && data[0] == 'M' && data[1] == 'Z' {
		file.Lfanew = binary.LittleEndian.Uint32(data[0x3c:])
	}
	if err := file.readOptional(parsed.OptionalHeader); err != nil {
		return nil, err
	}
	file.Sections = make([]*Section, len(parsed.Sections))
	for i, section := range parsed.Sections {
		item := &Section{
			Name:            section.Name,
			VirtualSize:     section.VirtualSize,
			VirtualAddress:  section.VirtualAddress,
			Size:            section.Size,
			Offset:          section.Offset,
			RelocOff:        section.PointerToRelocations,
			LineOff:         section.PointerToLineNumbers,
			Relocs:          section.NumberOfRelocations,
			Lines:           section.NumberOfLineNumbers,
			Characteristics: section.Characteristics,
		}
		if section.Offset != 0 && section.Size != 0 {
			raw, err := slice(data, uint64(section.Offset), uint64(section.Size))
			if err != nil {
				return nil, err
			}
			item.Bytes = raw
		}
		if file.Lfanew != 0 {
			item.RawName = rawName(data, file, i)
		}
		file.Sections[i] = item
	}
	if file.SymbolCount > 0 && file.SymbolOff != 0 {
		size := uint64(file.SymbolCount) * 18
		if end := uint64(file.SymbolOff) + size + 4; end <= uint64(len(data)) {
			strSize := uint64(binary.LittleEndian.Uint32(data[file.SymbolOff+uint32(size):]))
			if strSize >= 4 && uint64(file.SymbolOff)+size+strSize <= uint64(len(data)) {
				size += strSize
			}
		}
		raw, err := slice(data, uint64(file.SymbolOff), size)
		if err != nil {
			return nil, err
		}
		file.Symbols = raw
	}
	if err := file.readImports(); err != nil {
		return nil, err
	}
	if err := file.readExports(); err != nil {
		return nil, err
	}
	file.loaded = true
	return file, nil
}

func rawName(data []byte, file *File, index int) [8]byte {
	var name [8]byte
	if file.Optional == 0 {
		return name
	}
	off := uint64(file.Lfanew) + 4 + 20 + uint64(file.Optional) + uint64(index)*40
	if off+8 > uint64(len(data)) {
		return name
	}
	copy(name[:], data[off:off+8])
	return name
}

func (f *File) readOptional(header any) error {
	switch opt := header.(type) {
	case *pe.OptionalHeader64:
		f.Magic = opt.Magic
		f.LinkerMajor = opt.MajorLinkerVersion
		f.LinkerMinor = opt.MinorLinkerVersion
		f.SizeOfCode = opt.SizeOfCode
		f.SizeOfInitialized = opt.SizeOfInitializedData
		f.SizeOfUninitialized = opt.SizeOfUninitializedData
		f.Header.Entry = opt.AddressOfEntryPoint
		f.BaseOfCode = opt.BaseOfCode
		f.Header.ImageBase = opt.ImageBase
		f.SectionAlign = opt.SectionAlignment
		f.FileAlign = opt.FileAlignment
		f.OSMajor = opt.MajorOperatingSystemVersion
		f.OSMinor = opt.MinorOperatingSystemVersion
		f.ImageMajor = opt.MajorImageVersion
		f.ImageMinor = opt.MinorImageVersion
		f.SubsystemMajor = opt.MajorSubsystemVersion
		f.SubsystemMinor = opt.MinorSubsystemVersion
		f.Win32Version = opt.Win32VersionValue
		f.SizeOfImage = opt.SizeOfImage
		f.SizeOfHeaders = opt.SizeOfHeaders
		f.CheckSum = opt.CheckSum
		f.Header.Subsystem = Subsystem(opt.Subsystem)
		f.DllCharacteristics = opt.DllCharacteristics
		f.StackReserve = opt.SizeOfStackReserve
		f.StackCommit = opt.SizeOfStackCommit
		f.HeapReserve = opt.SizeOfHeapReserve
		f.HeapCommit = opt.SizeOfHeapCommit
		f.LoaderFlags = opt.LoaderFlags
		f.Directories = dirs(directorySlice(opt.DataDirectory[:], opt.NumberOfRvaAndSizes))
	case *pe.OptionalHeader32:
		f.Magic = opt.Magic
		f.LinkerMajor = opt.MajorLinkerVersion
		f.LinkerMinor = opt.MinorLinkerVersion
		f.SizeOfCode = opt.SizeOfCode
		f.SizeOfInitialized = opt.SizeOfInitializedData
		f.SizeOfUninitialized = opt.SizeOfUninitializedData
		f.Header.Entry = opt.AddressOfEntryPoint
		f.BaseOfCode = opt.BaseOfCode
		f.BaseOfData = opt.BaseOfData
		f.Header.ImageBase = uint64(opt.ImageBase)
		f.SectionAlign = opt.SectionAlignment
		f.FileAlign = opt.FileAlignment
		f.OSMajor = opt.MajorOperatingSystemVersion
		f.OSMinor = opt.MinorOperatingSystemVersion
		f.ImageMajor = opt.MajorImageVersion
		f.ImageMinor = opt.MinorImageVersion
		f.SubsystemMajor = opt.MajorSubsystemVersion
		f.SubsystemMinor = opt.MinorSubsystemVersion
		f.Win32Version = opt.Win32VersionValue
		f.SizeOfImage = opt.SizeOfImage
		f.SizeOfHeaders = opt.SizeOfHeaders
		f.CheckSum = opt.CheckSum
		f.Header.Subsystem = Subsystem(opt.Subsystem)
		f.DllCharacteristics = opt.DllCharacteristics
		f.StackReserve = uint64(opt.SizeOfStackReserve)
		f.StackCommit = uint64(opt.SizeOfStackCommit)
		f.HeapReserve = uint64(opt.SizeOfHeapReserve)
		f.HeapCommit = uint64(opt.SizeOfHeapCommit)
		f.LoaderFlags = opt.LoaderFlags
		f.Directories = dirs(directorySlice(opt.DataDirectory[:], opt.NumberOfRvaAndSizes))
	case nil:
		return ErrFormat
	default:
		return ErrFormat
	}
	return nil
}

func directorySlice(in []pe.DataDirectory, n uint32) []pe.DataDirectory {
	if n > uint32(len(in)) {
		n = uint32(len(in))
	}
	return in[:n]
}

func dirs(in []pe.DataDirectory) []Directory {
	if len(in) == 0 {
		return nil
	}
	out := make([]Directory, len(in))
	for i, item := range in {
		out[i] = Directory{VirtualAddress: item.VirtualAddress, Size: item.Size}
	}
	return out
}

func (f *File) readImports() error {
	data := f.directory(dirImport)
	if data.VirtualAddress == 0 || data.Size == 0 {
		return nil
	}
	table, ok := f.rva(data.VirtualAddress)
	if !ok {
		return nil
	}
	for len(table) >= 20 {
		original := binary.LittleEndian.Uint32(table[0:4])
		nameRVA := binary.LittleEndian.Uint32(table[12:16])
		first := binary.LittleEndian.Uint32(table[16:20])
		table = table[20:]
		if original == 0 && nameRVA == 0 && first == 0 {
			break
		}
		if nameRVA == 0 {
			continue
		}
		library, ok := f.cstring(nameRVA)
		if !ok || library == "" {
			continue
		}
		item := &Import{Library: library}
		thunk := original
		if thunk == 0 {
			thunk = first
		}
		symbols, err := f.readThunks(thunk)
		if err != nil {
			return err
		}
		item.Symbols = symbols
		f.Imports = append(f.Imports, item)
	}
	return nil
}

func (f *File) readThunks(rva uint32) ([]Imported, error) {
	thunks, ok := f.rva(rva)
	if !ok {
		return nil, nil
	}
	var out []Imported
	if f.is64() {
		for len(thunks) >= 8 {
			value := binary.LittleEndian.Uint64(thunks[0:8])
			thunks = thunks[8:]
			if value == 0 {
				break
			}
			if value&0x8000000000000000 != 0 {
				out = append(out, Imported{Ordinal: uint16(value), ByOrdinal: true})
				continue
			}
			name, hint, ok := f.hintName(uint32(value))
			if !ok {
				return nil, ErrFormat
			}
			out = append(out, Imported{Name: name, Hint: hint})
		}
		return out, nil
	}
	for len(thunks) >= 4 {
		value := binary.LittleEndian.Uint32(thunks[0:4])
		thunks = thunks[4:]
		if value == 0 {
			break
		}
		if value&0x80000000 != 0 {
			out = append(out, Imported{Ordinal: uint16(value), ByOrdinal: true})
			continue
		}
		name, hint, ok := f.hintName(value)
		if !ok {
			return nil, ErrFormat
		}
		out = append(out, Imported{Name: name, Hint: hint})
	}
	return out, nil
}

func (f *File) hintName(rva uint32) (string, uint16, bool) {
	raw, ok := f.rva(rva)
	if !ok || len(raw) < 3 {
		return "", 0, false
	}
	hint := binary.LittleEndian.Uint16(raw[:2])
	name, _ := cstring(raw[2:])
	return name, hint, name != ""
}

func (f *File) readExports() error {
	data := f.directory(dirExport)
	if data.VirtualAddress == 0 || data.Size == 0 {
		return nil
	}
	dir, ok := f.rva(data.VirtualAddress)
	if !ok || len(dir) < 40 {
		return nil
	}
	f.OrdinalBase = binary.LittleEndian.Uint32(dir[16:20])
	count := binary.LittleEndian.Uint32(dir[20:24])
	names := binary.LittleEndian.Uint32(dir[24:28])
	funcRVA := binary.LittleEndian.Uint32(dir[28:32])
	nameRVA := binary.LittleEndian.Uint32(dir[32:36])
	ordRVA := binary.LittleEndian.Uint32(dir[36:40])
	if dll, ok := f.cstring(binary.LittleEndian.Uint32(dir[12:16])); ok {
		f.DllName = dll
	}
	functions, ok := f.rva(funcRVA)
	if !ok {
		return nil
	}
	nameTable, _ := f.rva(nameRVA)
	ordinals, _ := f.rva(ordRVA)
	used := make([]bool, count)
	for i := uint32(0); i < names && len(nameTable) >= 4 && len(ordinals) >= 2; i++ {
		ordinal := uint32(binary.LittleEndian.Uint16(ordinals[:2]))
		ordinals = ordinals[2:]
		symbolRVA := binary.LittleEndian.Uint32(nameTable[:4])
		nameTable = nameTable[4:]
		name, ok := f.cstring(symbolRVA)
		if !ok {
			continue
		}
		if ordinal >= count || len(functions) < int((ordinal+1)*4) {
			continue
		}
		used[ordinal] = true
		rva := binary.LittleEndian.Uint32(functions[ordinal*4:])
		item := Export{Name: name, Ordinal: f.OrdinalBase + ordinal, RVA: rva}
		if rva >= data.VirtualAddress && rva < data.VirtualAddress+data.Size {
			if forwarder, ok := f.cstring(rva); ok {
				item.Forwarder = forwarder
				item.RVA = 0
			}
		}
		f.Exports = append(f.Exports, item)
	}
	for ordinal, seen := range used {
		if seen || len(functions) < (ordinal+1)*4 {
			continue
		}
		rva := binary.LittleEndian.Uint32(functions[ordinal*4:])
		if rva == 0 {
			continue
		}
		f.Exports = append(f.Exports, Export{Ordinal: f.OrdinalBase + uint32(ordinal), RVA: rva})
	}
	return nil
}

func (f *File) directory(index int) Directory {
	if index < 0 || index >= len(f.Directories) {
		return Directory{}
	}
	return f.Directories[index]
}

func (f *File) rva(rva uint32) ([]byte, bool) {
	if rva == 0 {
		return nil, false
	}
	for _, section := range f.Sections {
		size := section.VirtualSize
		if uint32(len(section.Bytes)) > size {
			size = uint32(len(section.Bytes))
		}
		if section.VirtualAddress > rva || rva-section.VirtualAddress >= size {
			continue
		}
		off := rva - section.VirtualAddress
		if off > uint32(len(section.Bytes)) {
			return nil, false
		}
		return section.Bytes[off:], true
	}
	return nil, false
}

func (f *File) cstring(rva uint32) (string, bool) {
	raw, ok := f.rva(rva)
	if !ok {
		return "", false
	}
	return cstring(raw)
}

func cstring(raw []byte) (string, bool) {
	for i, b := range raw {
		if b == 0 {
			return string(raw[:i]), true
		}
	}
	return "", false
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
