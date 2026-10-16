package pe

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
		if f.Optional == 0 {
			f.Optional = uint16(f.encodedOptional())
		}
		return nil
	}
	if f.Header.Machine == MachineNone {
		f.Header.Machine = Amd64
	}
	if f.Magic == 0 {
		if f.Header.Machine == I386 {
			f.Magic = magic32
		} else {
			f.Magic = magic64
		}
	}
	if f.SectionAlign == 0 {
		f.SectionAlign = 0x1000
	}
	if f.FileAlign == 0 {
		f.FileAlign = 0x200
	}
	if f.Lfanew == 0 {
		f.Lfanew = 0x80
	}
	if len(f.Directories) < dirCount {
		next := make([]Directory, dirCount)
		copy(next, f.Directories)
		f.Directories = next
	}
	f.noteSpecialSections()
	f.placeSections()
	if err := f.fillImports(); err != nil {
		return err
	}
	if err := f.fillExports(); err != nil {
		return err
	}
	if f.placeSections() {
		f.clearSynthDirs()
		if err := f.fillImports(); err != nil {
			return err
		}
		if err := f.fillExports(); err != nil {
			return err
		}
	}
	f.placeFiles()
	f.Optional = uint16(f.encodedOptional())
	return nil
}

func (f *File) noteSpecialSections() {
	if f.directory(dirImport).VirtualAddress == 0 && len(f.Imports) > 0 && f.sectionNamed(".idata") == nil {
		f.Sections = append(f.Sections, &Section{
			Name:            ".idata",
			Characteristics: scnInit | scnRead,
			auto:            true,
		})
		copy(f.Sections[len(f.Sections)-1].RawName[:], ".idata")
	}
	if f.directory(dirExport).VirtualAddress == 0 && len(f.Exports) > 0 && f.sectionNamed(".edata") == nil {
		f.Sections = append(f.Sections, &Section{
			Name:            ".edata",
			Characteristics: scnInit | scnRead,
			auto:            true,
		})
		copy(f.Sections[len(f.Sections)-1].RawName[:], ".edata")
	}
}

func (f *File) clearSynthDirs() {
	if section := f.sectionNamed(".idata"); section != nil && section.auto {
		f.setDir(dirImport, 0, 0)
		f.setDir(dirIAT, 0, 0)
	}
	if section := f.sectionNamed(".edata"); section != nil && section.auto {
		f.setDir(dirExport, 0, 0)
	}
}

func (f *File) sectionNamed(name string) *Section {
	for _, section := range f.Sections {
		if section.Name == name {
			return section
		}
	}
	return nil
}

func (f *File) placeSections() bool {
	align := f.SectionAlign
	if align == 0 {
		align = 0x1000
	}
	next := align
	changed := false
	for _, section := range f.Sections {
		size := section.VirtualSize
		if uint32(len(section.Bytes)) > size {
			size = uint32(len(section.Bytes))
			section.VirtualSize = size
		}
		stride := size
		if stride == 0 {
			stride = align
		}
		va := section.VirtualAddress
		if va == 0 || section.auto {
			va = next
			if section.VirtualAddress != va {
				changed = true
			}
			section.VirtualAddress = va
			section.auto = true
		}
		end := alignUp(va+stride, align)
		if end <= va {
			end = va + align
		}
		if end > next {
			next = end
		}
	}
	return changed
}

func (f *File) placeFiles() {
	opt := f.encodedOptional()
	headers := alignUp(f.Lfanew+4+20+uint32(opt)+uint32(len(f.Sections))*40, f.FileAlign)
	off := headers
	var code, init, uninit uint32
	for _, section := range f.Sections {
		if len(section.Bytes) == 0 {
			section.Offset = 0
			section.Size = 0
			section.Characteristics |= scnUninit
			uninit += alignUp(section.VirtualSize, f.FileAlign)
			continue
		}
		rawSize := alignUp(uint32(len(section.Bytes)), f.FileAlign)
		if rawSize != uint32(len(section.Bytes)) {
			padded := make([]byte, rawSize)
			copy(padded, section.Bytes)
			section.Bytes = padded
		}
		section.Size = rawSize
		section.Offset = off
		off += rawSize
		if section.Characteristics&scnCode != 0 {
			code += rawSize
		} else {
			init += rawSize
		}
	}
	f.SizeOfHeaders = headers
	f.SizeOfCode = code
	f.SizeOfInitialized = init
	f.SizeOfUninitialized = uninit
	var image uint32
	for _, section := range f.Sections {
		size := section.VirtualSize
		if section.Size > size && section.Characteristics&scnUninit == 0 {
			size = section.Size
		}
		end := alignUp(section.VirtualAddress+size, f.SectionAlign)
		if end > image {
			image = end
		}
	}
	if image == 0 {
		image = f.SectionAlign
	}
	f.SizeOfImage = image
	f.Span = uint64(off)
	f.Optional = uint16(opt)
}

func (f *File) fillImports() error {
	if len(f.Imports) == 0 || f.directory(dirImport).VirtualAddress != 0 {
		return nil
	}
	section := f.sectionNamed(".idata")
	if section == nil || section.VirtualAddress == 0 {
		return nil
	}
	blob, dirOff, dirSize, iatOff, iatSize := encodeImports(f.Imports, section.VirtualAddress, f.is64())
	section.Bytes = blob
	section.VirtualSize = uint32(len(blob))
	section.Size = section.VirtualSize
	f.setDir(dirImport, section.VirtualAddress+dirOff, dirSize)
	f.setDir(dirIAT, section.VirtualAddress+iatOff, iatSize)
	return nil
}

func encodeImports(imports []*Import, base uint32, pe64 bool) (blob []byte, dirOff, dirSize, iatOff, iatSize uint32) {
	thunk := 4
	if pe64 {
		thunk = 8
	}
	dirSize = uint32(20 * (len(imports) + 1))
	pos := int(dirSize)
	type plan struct {
		ilt, iat int
		hints    []int
		dll      int
	}
	plans := make([]plan, len(imports))
	for i, item := range imports {
		plans[i].ilt = pos
		pos += thunk * (len(item.Symbols) + 1)
	}
	iatOff = uint32(pos)
	for i, item := range imports {
		plans[i].iat = pos
		pos += thunk * (len(item.Symbols) + 1)
	}
	iatSize = uint32(pos) - iatOff
	for i, item := range imports {
		plans[i].hints = make([]int, len(item.Symbols))
		for s, symbol := range item.Symbols {
			if symbol.ByOrdinal {
				plans[i].hints[s] = -1
				continue
			}
			plans[i].hints[s] = pos
			pos += 2 + len(symbol.Name) + 1
			if pos%2 != 0 {
				pos++
			}
		}
		plans[i].dll = pos
		pos += len(item.Library) + 1
		if pos%2 != 0 {
			pos++
		}
	}
	buf := make([]byte, pos)
	for i, item := range imports {
		entry := buf[i*20 : (i+1)*20]
		plan := plans[i]
		binary.LittleEndian.PutUint32(entry[0:4], base+uint32(plan.ilt))
		binary.LittleEndian.PutUint32(entry[12:16], base+uint32(plan.dll))
		binary.LittleEndian.PutUint32(entry[16:20], base+uint32(plan.iat))
		copy(buf[plan.dll:], item.Library)
		for s, symbol := range item.Symbols {
			var value uint64
			if symbol.ByOrdinal {
				value = uint64(symbol.Ordinal)
				if pe64 {
					value |= 0x8000000000000000
				} else {
					value |= 0x80000000
				}
			} else {
				binary.LittleEndian.PutUint16(buf[plan.hints[s]:], symbol.Hint)
				copy(buf[plan.hints[s]+2:], symbol.Name)
				value = uint64(base + uint32(plan.hints[s]))
			}
			putThunk(buf[plan.ilt+s*thunk:], value, pe64)
			putThunk(buf[plan.iat+s*thunk:], value, pe64)
		}
	}
	return buf, 0, dirSize, iatOff, iatSize
}

func putThunk(buf []byte, value uint64, pe64 bool) {
	if pe64 {
		binary.LittleEndian.PutUint64(buf[:8], value)
		return
	}
	binary.LittleEndian.PutUint32(buf[:4], uint32(value))
}

func (f *File) fillExports() error {
	if len(f.Exports) == 0 || f.directory(dirExport).VirtualAddress != 0 {
		return nil
	}
	section := f.sectionNamed(".edata")
	if section == nil || section.VirtualAddress == 0 {
		return nil
	}
	if f.OrdinalBase == 0 {
		f.OrdinalBase = 1
	}
	if f.DllName == "" {
		f.DllName = "export.dll"
	}
	blob := encodeExports(f.Exports, f.DllName, f.OrdinalBase, section.VirtualAddress)
	section.Bytes = blob
	section.VirtualSize = uint32(len(blob))
	section.Size = section.VirtualSize
	f.setDir(dirExport, section.VirtualAddress, uint32(len(blob)))
	return nil
}

func encodeExports(exports []Export, dll string, base, section uint32) []byte {
	n := len(exports)
	funcOff := 40
	namePtr := funcOff + n*4
	ordOff := namePtr + n*4
	strOff := ordOff + n*2
	if strOff%2 != 0 {
		strOff++
	}
	names := make([]string, n)
	for i, item := range exports {
		if item.Forwarder != "" {
			names[i] = item.Forwarder
			continue
		}
		names[i] = ""
	}
	// strings: dll name, export names, forwarders
	type str struct {
		off  int
		text string
	}
	var strs []str
	add := func(text string) int {
		if text == "" {
			return 0
		}
		off := strOff
		strOff += len(text) + 1
		if strOff%2 != 0 {
			strOff++
		}
		strs = append(strs, str{off: off, text: text})
		return off
	}
	dllOff := add(dll)
	nameOff := make([]int, n)
	forwardOff := make([]int, n)
	for i, item := range exports {
		nameOff[i] = add(item.Name)
		if item.Forwarder != "" {
			forwardOff[i] = add(item.Forwarder)
		}
	}
	buf := make([]byte, strOff)
	for _, item := range strs {
		copy(buf[item.off:], item.text)
	}
	binary.LittleEndian.PutUint32(buf[12:16], section+uint32(dllOff))
	binary.LittleEndian.PutUint32(buf[16:20], base)
	binary.LittleEndian.PutUint32(buf[20:24], uint32(n))
	binary.LittleEndian.PutUint32(buf[24:28], uint32(n))
	binary.LittleEndian.PutUint32(buf[28:32], section+uint32(funcOff))
	binary.LittleEndian.PutUint32(buf[32:36], section+uint32(namePtr))
	binary.LittleEndian.PutUint32(buf[36:40], section+uint32(ordOff))
	for i, item := range exports {
		rva := item.RVA
		if item.Forwarder != "" {
			rva = section + uint32(forwardOff[i])
		}
		binary.LittleEndian.PutUint32(buf[funcOff+i*4:], rva)
		if item.Name != "" {
			binary.LittleEndian.PutUint32(buf[namePtr+i*4:], section+uint32(nameOff[i]))
			binary.LittleEndian.PutUint16(buf[ordOff+i*2:], uint16(i))
		}
	}
	return buf
}

func (f *File) setDir(index int, rva, size uint32) {
	if len(f.Directories) <= index {
		next := make([]Directory, index+1)
		copy(next, f.Directories)
		f.Directories = next
	}
	f.Directories[index] = Directory{VirtualAddress: rva, Size: size}
}

func (f *File) encodedOptional() int {
	n := len(f.Directories)
	if n == 0 {
		n = dirCount
	}
	if f.is64() {
		return 112 + 8*n
	}
	return 96 + 8*n
}

func (f *File) image() ([]byte, error) {
	optSize := int(f.Optional)
	if optSize < f.encodedOptional() {
		optSize = f.encodedOptional()
	}
	if f.Lfanew < 0x40 {
		return nil, ErrFormat
	}
	headers := uint64(f.Lfanew) + 4 + 20 + uint64(optSize) + uint64(len(f.Sections))*40
	span := f.Span
	if headers > span {
		span = headers
	}
	for _, section := range f.Sections {
		if section.Offset == 0 || len(section.Bytes) == 0 {
			continue
		}
		end := uint64(section.Offset) + uint64(len(section.Bytes))
		if end > span {
			span = end
		}
	}
	if f.SymbolOff != 0 && len(f.Symbols) > 0 {
		end := uint64(f.SymbolOff) + uint64(len(f.Symbols))
		if end > span {
			span = end
		}
	}
	buf := make([]byte, span)
	if f.SymbolOff != 0 && len(f.Symbols) > 0 {
		copy(buf[f.SymbolOff:], f.Symbols)
	}
	for _, section := range f.Sections {
		if section.Offset == 0 || len(section.Bytes) == 0 {
			continue
		}
		copy(buf[section.Offset:], section.Bytes)
	}
	buf[0], buf[1] = 'M', 'Z'
	binary.LittleEndian.PutUint32(buf[0x3c:], f.Lfanew)
	pe := buf[f.Lfanew:]
	pe[0], pe[1], pe[2], pe[3] = 'P', 'E', 0, 0
	coff := pe[4:]
	binary.LittleEndian.PutUint16(coff[0:2], uint16(f.Header.Machine))
	binary.LittleEndian.PutUint16(coff[2:4], uint16(len(f.Sections)))
	binary.LittleEndian.PutUint32(coff[4:8], f.TimeDateStamp)
	binary.LittleEndian.PutUint32(coff[8:12], f.SymbolOff)
	binary.LittleEndian.PutUint32(coff[12:16], f.SymbolCount)
	binary.LittleEndian.PutUint16(coff[16:18], uint16(optSize))
	binary.LittleEndian.PutUint16(coff[18:20], f.Characteristics)
	opt := coff[20 : 20+optSize]
	f.putOptional(opt)
	sect := coff[20+optSize:]
	for i, section := range f.Sections {
		entry := sect[i*40 : (i+1)*40]
		name := section.RawName
		if name == [8]byte{} {
			copy(name[:], section.Name)
		}
		copy(entry[0:8], name[:])
		binary.LittleEndian.PutUint32(entry[8:12], section.VirtualSize)
		binary.LittleEndian.PutUint32(entry[12:16], section.VirtualAddress)
		binary.LittleEndian.PutUint32(entry[16:20], section.Size)
		binary.LittleEndian.PutUint32(entry[20:24], section.Offset)
		binary.LittleEndian.PutUint32(entry[24:28], section.RelocOff)
		binary.LittleEndian.PutUint32(entry[28:32], section.LineOff)
		binary.LittleEndian.PutUint16(entry[32:34], section.Relocs)
		binary.LittleEndian.PutUint16(entry[34:36], section.Lines)
		binary.LittleEndian.PutUint32(entry[36:40], section.Characteristics)
	}
	return buf, nil
}

func (f *File) putOptional(buf []byte) {
	n := len(f.Directories)
	if f.is64() {
		binary.LittleEndian.PutUint16(buf[0:2], f.Magic)
		buf[2] = f.LinkerMajor
		buf[3] = f.LinkerMinor
		binary.LittleEndian.PutUint32(buf[4:8], f.SizeOfCode)
		binary.LittleEndian.PutUint32(buf[8:12], f.SizeOfInitialized)
		binary.LittleEndian.PutUint32(buf[12:16], f.SizeOfUninitialized)
		binary.LittleEndian.PutUint32(buf[16:20], f.Header.Entry)
		binary.LittleEndian.PutUint32(buf[20:24], f.BaseOfCode)
		binary.LittleEndian.PutUint64(buf[24:32], f.Header.ImageBase)
		binary.LittleEndian.PutUint32(buf[32:36], f.SectionAlign)
		binary.LittleEndian.PutUint32(buf[36:40], f.FileAlign)
		binary.LittleEndian.PutUint16(buf[40:42], f.OSMajor)
		binary.LittleEndian.PutUint16(buf[42:44], f.OSMinor)
		binary.LittleEndian.PutUint16(buf[44:46], f.ImageMajor)
		binary.LittleEndian.PutUint16(buf[46:48], f.ImageMinor)
		binary.LittleEndian.PutUint16(buf[48:50], f.SubsystemMajor)
		binary.LittleEndian.PutUint16(buf[50:52], f.SubsystemMinor)
		binary.LittleEndian.PutUint32(buf[52:56], f.Win32Version)
		binary.LittleEndian.PutUint32(buf[56:60], f.SizeOfImage)
		binary.LittleEndian.PutUint32(buf[60:64], f.SizeOfHeaders)
		binary.LittleEndian.PutUint32(buf[64:68], f.CheckSum)
		binary.LittleEndian.PutUint16(buf[68:70], uint16(f.Header.Subsystem))
		binary.LittleEndian.PutUint16(buf[70:72], f.DllCharacteristics)
		binary.LittleEndian.PutUint64(buf[72:80], f.StackReserve)
		binary.LittleEndian.PutUint64(buf[80:88], f.StackCommit)
		binary.LittleEndian.PutUint64(buf[88:96], f.HeapReserve)
		binary.LittleEndian.PutUint64(buf[96:104], f.HeapCommit)
		binary.LittleEndian.PutUint32(buf[104:108], f.LoaderFlags)
		binary.LittleEndian.PutUint32(buf[108:112], uint32(n))
		for i, dir := range f.Directories {
			off := 112 + i*8
			binary.LittleEndian.PutUint32(buf[off:off+4], dir.VirtualAddress)
			binary.LittleEndian.PutUint32(buf[off+4:off+8], dir.Size)
		}
		return
	}
	binary.LittleEndian.PutUint16(buf[0:2], f.Magic)
	buf[2] = f.LinkerMajor
	buf[3] = f.LinkerMinor
	binary.LittleEndian.PutUint32(buf[4:8], f.SizeOfCode)
	binary.LittleEndian.PutUint32(buf[8:12], f.SizeOfInitialized)
	binary.LittleEndian.PutUint32(buf[12:16], f.SizeOfUninitialized)
	binary.LittleEndian.PutUint32(buf[16:20], f.Header.Entry)
	binary.LittleEndian.PutUint32(buf[20:24], f.BaseOfCode)
	binary.LittleEndian.PutUint32(buf[24:28], f.BaseOfData)
	binary.LittleEndian.PutUint32(buf[28:32], uint32(f.Header.ImageBase))
	binary.LittleEndian.PutUint32(buf[32:36], f.SectionAlign)
	binary.LittleEndian.PutUint32(buf[36:40], f.FileAlign)
	binary.LittleEndian.PutUint16(buf[40:42], f.OSMajor)
	binary.LittleEndian.PutUint16(buf[42:44], f.OSMinor)
	binary.LittleEndian.PutUint16(buf[44:46], f.ImageMajor)
	binary.LittleEndian.PutUint16(buf[46:48], f.ImageMinor)
	binary.LittleEndian.PutUint16(buf[48:50], f.SubsystemMajor)
	binary.LittleEndian.PutUint16(buf[50:52], f.SubsystemMinor)
	binary.LittleEndian.PutUint32(buf[52:56], f.Win32Version)
	binary.LittleEndian.PutUint32(buf[56:60], f.SizeOfImage)
	binary.LittleEndian.PutUint32(buf[60:64], f.SizeOfHeaders)
	binary.LittleEndian.PutUint32(buf[64:68], f.CheckSum)
	binary.LittleEndian.PutUint16(buf[68:70], uint16(f.Header.Subsystem))
	binary.LittleEndian.PutUint16(buf[70:72], f.DllCharacteristics)
	binary.LittleEndian.PutUint32(buf[72:76], uint32(f.StackReserve))
	binary.LittleEndian.PutUint32(buf[76:80], uint32(f.StackCommit))
	binary.LittleEndian.PutUint32(buf[80:84], uint32(f.HeapReserve))
	binary.LittleEndian.PutUint32(buf[84:88], uint32(f.HeapCommit))
	binary.LittleEndian.PutUint32(buf[88:92], f.LoaderFlags)
	binary.LittleEndian.PutUint32(buf[92:96], uint32(n))
	for i, dir := range f.Directories {
		off := 96 + i*8
		binary.LittleEndian.PutUint32(buf[off:off+4], dir.VirtualAddress)
		binary.LittleEndian.PutUint32(buf[off+4:off+8], dir.Size)
	}
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
	out.Directories = append([]Directory(nil), f.Directories...)
	out.Sections = make([]*Section, len(f.Sections))
	for i, section := range f.Sections {
		item := *section
		item.Bytes = append([]byte(nil), section.Bytes...)
		out.Sections[i] = &item
	}
	out.Imports = make([]*Import, len(f.Imports))
	for i, item := range f.Imports {
		next := *item
		next.Symbols = append([]Imported(nil), item.Symbols...)
		out.Imports[i] = &next
	}
	out.Exports = append([]Export(nil), f.Exports...)
	out.Symbols = append([]byte(nil), f.Symbols...)
	return &out
}
