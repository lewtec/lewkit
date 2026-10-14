package disasm

import (
	"debug/pe"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
)

var (
	ErrDecodeHexEmptyHex             = errors.New("empty hex")
	ErrDecodeHexHex                  = errors.New("hex")
	ErrELFFileArchitectureElfMachine = errors.New("elf machine")
	ErrELFFileSectionsElfSection     = errors.New("elf section")
	ErrELFFileTextSectionElfSection  = errors.New("elf section")
	ErrLookupNamedUnknown            = errors.New("unknown")
	ErrMachOFileArchitectureMachoCpu = errors.New("macho cpu")
	ErrMachoSection                  = errors.New("macho section")
	ErrOddHexLength                  = errors.New("odd hex length")
	ErrOpenMachOMacho                = errors.New("macho")
	ErrOpenObjectElf                 = errors.New("elf")
	ErrOpenPEPe                      = errors.New("pe")
	ErrPEFileArchitecturePeMachine   = errors.New("pe machine")
	ErrPEFileSectionsPeSection       = errors.New("pe section")
	ErrPEFileTextSectionPeSection    = errors.New("pe section")
	ErrParseModeUnknownMode          = errors.New("unknown mode")
)

// PEFile wraps [pe.File].
type PEFile struct{ File *pe.File }

var _ Object = (*PEFile)(nil)

func openPE(reader io.ReaderAt) (*PEFile, error) {
	file, err := pe.NewFile(reader)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrOpenPEPe, err)
	}
	return &PEFile{File: file}, nil
}

// Close closes the underlying file.
func (file *PEFile) Close() error {
	return file.File.Close()
}

// Architecture maps the PE header to Capstone arch and mode.
func (file *PEFile) Architecture() (Architecture, Mode, error) {
	mode := Mode32
	switch file.File.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		mode = Mode32
	case *pe.OptionalHeader64:
		mode = Mode64
	}
	switch file.File.Machine {
	case pe.IMAGE_FILE_MACHINE_I386:
		return ArchitectureX86, Mode32, nil
	case pe.IMAGE_FILE_MACHINE_AMD64:
		return ArchitectureX86, Mode64, nil
	case pe.IMAGE_FILE_MACHINE_ARM:
		return ArchitectureARM, mode, nil
	case pe.IMAGE_FILE_MACHINE_ARM64:
		return ArchitectureAArch64, mode, nil
	default:
		return 0, 0, fmt.Errorf("%w %#x", ErrPEFileArchitecturePeMachine, file.File.Machine)
	}
}

// TextSection returns the named section, or .text.
func (file *PEFile) TextSection(name string) (uint64, []byte, string, error) {
	if name == "" {
		name = ".text"
	}
	var section *pe.Section
	for _, candidate := range file.File.Sections {
		if candidate.Name == name {
			section = candidate
			break
		}
	}
	if section == nil {
		return 0, nil, "", fmt.Errorf("%w: %s", ErrNoText, name)
	}
	data, err := section.Data()
	if err != nil {
		return 0, nil, "", fmt.Errorf("%w %s: %w", ErrPEFileTextSectionPeSection, section.Name, err)
	}
	return uint64(section.VirtualAddress), data, section.Name, nil
}

const (
	imageSectionExecute = 0x20000000
	imageSectionRead    = 0x40000000
	imageSectionWrite   = 0x80000000
	imageSymbolFunction = 2
)

// Entry is the image base plus the address of the entry point.
func (file *PEFile) Entry() uint64 {
	switch header := file.File.OptionalHeader.(type) {
	case *pe.OptionalHeader64:
		return header.ImageBase + uint64(header.AddressOfEntryPoint)
	case *pe.OptionalHeader32:
		return uint64(header.ImageBase) + uint64(header.AddressOfEntryPoint)
	default:
		return 0
	}
}

// Sections returns PE sections. Addresses include the image base.
// Size is the virtual size. File-alignment bytes past that size are not code.
func (file *PEFile) Sections() ([]Section, error) {
	base := file.imageBase()
	var sections []Section
	for _, section := range file.File.Sections {
		size := uint64(section.VirtualSize)
		var data []byte
		if section.Offset != 0 {
			read, err := section.Data()
			if err != nil {
				return nil, fmt.Errorf("%w %s: %w", ErrPEFileSectionsPeSection, section.Name, err)
			}
			data = read
		}
		if size == 0 {
			size = uint64(len(data))
		}
		if uint64(len(data)) > size {
			data = data[:size]
		}
		if size == 0 {
			continue
		}
		sections = append(sections, Section{
			Name:    section.Name,
			Address: base + uint64(section.VirtualAddress),
			Size:    size,
			Bytes:   data,
			Exec:    section.Characteristics&imageSectionExecute != 0,
			Write:   section.Characteristics&imageSectionWrite != 0,
			Read:    section.Characteristics&imageSectionRead != 0,
		})
	}
	return sections, nil
}

// Import is one imported symbol and the address of its IAT slot.
type Import struct {
	Library string
	Name    string
	Address uint64
}

// Export is one exported symbol. Address is the image address of its target.
type Export struct {
	Name    string
	Address uint64
}

// Imports reads the import directory. The address is the IAT slot.
// The library name has no .dll suffix.
func (file *PEFile) Imports() ([]Import, error) {
	directory, pe64, ok := file.dataDirectory(pe.IMAGE_DIRECTORY_ENTRY_IMPORT)
	if !ok || directory.VirtualAddress == 0 || directory.Size == 0 {
		return nil, nil
	}
	var imports []Import
	for offset := uint32(0); ; offset += 20 {
		entry, ok := file.bytesAt(directory.VirtualAddress+offset, 20)
		if !ok {
			break
		}
		original := binary.LittleEndian.Uint32(entry[0:4])
		nameRVA := binary.LittleEndian.Uint32(entry[12:16])
		first := binary.LittleEndian.Uint32(entry[16:20])
		if original == 0 && first == 0 && nameRVA == 0 {
			break
		}
		if first == 0 {
			continue
		}
		library, _ := file.cstringAt(nameRVA)
		lookup := original
		if lookup == 0 {
			lookup = first
		}
		step := uint32(4)
		ordinalMask := uint64(0x80000000)
		if pe64 {
			step = 8
			ordinalMask = 1 << 63
		}
		for index := uint32(0); ; index++ {
			slotRVA := first + index*step
			value, ok := file.thunkAt(lookup+index*step, pe64)
			if !ok || value == 0 {
				break
			}
			item := Import{
				Library: importLibrary(library),
				Address: file.imageBase() + uint64(slotRVA),
			}
			if value&ordinalMask != 0 {
				item.Name = fmt.Sprintf("#%d", value&0xffff)
			} else if text, ok := file.cstringAt(uint32(value) + 2); ok {
				item.Name = text
			}
			if item.Name == "" {
				continue
			}
			imports = append(imports, item)
		}
	}
	return imports, nil
}

// Exports reads the export directory. Address is the image base plus the export RVA.
func (file *PEFile) Exports() ([]Export, error) {
	directory, _, ok := file.dataDirectory(pe.IMAGE_DIRECTORY_ENTRY_EXPORT)
	if !ok || directory.VirtualAddress == 0 || directory.Size == 0 {
		return nil, nil
	}
	table, ok := file.bytesAt(directory.VirtualAddress, 40)
	if !ok {
		return nil, nil
	}
	count := binary.LittleEndian.Uint32(table[24:28])
	functions := binary.LittleEndian.Uint32(table[28:32])
	names := binary.LittleEndian.Uint32(table[32:36])
	ordinals := binary.LittleEndian.Uint32(table[36:40])
	var exports []Export
	for index := uint32(0); index < count; index++ {
		nameSlot, ok := file.bytesAt(names+index*4, 4)
		ordinalSlot, ordinalOK := file.bytesAt(ordinals+index*2, 2)
		if !ok || !ordinalOK {
			continue
		}
		nameRVA := binary.LittleEndian.Uint32(nameSlot)
		ordinal := binary.LittleEndian.Uint16(ordinalSlot)
		functionSlot, ok := file.bytesAt(functions+uint32(ordinal)*4, 4)
		if !ok {
			continue
		}
		functionRVA := binary.LittleEndian.Uint32(functionSlot)
		if functionRVA == 0 {
			continue
		}
		if functionRVA >= directory.VirtualAddress && functionRVA < directory.VirtualAddress+directory.Size {
			continue
		}
		name, ok := file.cstringAt(nameRVA)
		if !ok || name == "" {
			continue
		}
		exports = append(exports, Export{
			Name:    name,
			Address: file.imageBase() + uint64(functionRVA),
		})
	}
	return exports, nil
}

func importLibrary(name string) string {
	trimmed := strings.TrimSuffix(strings.ToLower(name), ".dll")
	return trimmed
}

func (file *PEFile) dataDirectory(index int) (pe.DataDirectory, bool, bool) {
	switch header := file.File.OptionalHeader.(type) {
	case *pe.OptionalHeader64:
		if header.NumberOfRvaAndSizes <= uint32(index) {
			return pe.DataDirectory{}, true, false
		}
		return header.DataDirectory[index], true, true
	case *pe.OptionalHeader32:
		if header.NumberOfRvaAndSizes <= uint32(index) {
			return pe.DataDirectory{}, false, false
		}
		return header.DataDirectory[index], false, true
	default:
		return pe.DataDirectory{}, false, false
	}
}

func (file *PEFile) thunkAt(rva uint32, pe64 bool) (uint64, bool) {
	if pe64 {
		raw, ok := file.bytesAt(rva, 8)
		if !ok {
			return 0, false
		}
		return binary.LittleEndian.Uint64(raw), true
	}
	raw, ok := file.bytesAt(rva, 4)
	if !ok {
		return 0, false
	}
	return uint64(binary.LittleEndian.Uint32(raw)), true
}

func (file *PEFile) bytesAt(rva uint32, size int) ([]byte, bool) {
	if size < 0 {
		return nil, false
	}
	for _, section := range file.File.Sections {
		if rva < section.VirtualAddress {
			continue
		}
		offset := int(rva - section.VirtualAddress)
		data, err := section.Data()
		if err != nil || offset < 0 || offset+size > len(data) {
			continue
		}
		return data[offset : offset+size], true
	}
	return nil, false
}

func (file *PEFile) cstringAt(rva uint32) (string, bool) {
	for _, section := range file.File.Sections {
		if rva < section.VirtualAddress {
			continue
		}
		offset := int(rva - section.VirtualAddress)
		data, err := section.Data()
		if err != nil || offset < 0 || offset >= len(data) {
			continue
		}
		end := offset
		for end < len(data) && data[end] != 0 {
			end++
		}
		return string(data[offset:end]), true
	}
	return "", false
}

func (file *PEFile) imageBase() uint64 {
	switch header := file.File.OptionalHeader.(type) {
	case *pe.OptionalHeader64:
		return header.ImageBase
	case *pe.OptionalHeader32:
		return uint64(header.ImageBase)
	default:
		return 0
	}
}

// Symbols returns COFF symbols with section-relative addresses resolved.
func (file *PEFile) Symbols() Symbols {
	if len(file.File.COFFSymbols) == 0 {
		return nil
	}
	var out Symbols
	for i := 0; i < len(file.File.COFFSymbols); i++ {
		symbol := file.File.COFFSymbols[i]
		name, err := symbol.FullName(file.File.StringTable)
		if err != nil || name == "" {
			continue
		}
		if symbol.SectionNumber <= 0 || int(symbol.SectionNumber) > len(file.File.Sections) {
			continue
		}
		section := file.File.Sections[symbol.SectionNumber-1]
		out = append(out, Symbol{
			Name:    name,
			Address: uint64(section.VirtualAddress) + uint64(symbol.Value),
			Code:    symbol.Type>>8 == imageSymbolFunction,
		})
		i += int(symbol.NumberOfAuxSymbols)
	}
	return out
}
