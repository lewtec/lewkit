package disasm

import (
	"debug/pe"
	"fmt"
	"io"
)

// PEFile wraps [pe.File].
type PEFile struct{ File *pe.File }

var _ Object = (*PEFile)(nil)

func openPE(reader io.ReaderAt) (*PEFile, error) {
	file, err := pe.NewFile(reader)
	if err != nil {
		return nil, fmt.Errorf("pe: %w", err)
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
		return 0, 0, fmt.Errorf("pe machine %#x", file.File.Machine)
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
		return 0, nil, "", fmt.Errorf("pe section %s: %w", section.Name, err)
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
func (file *PEFile) Sections() ([]Section, error) {
	base := file.imageBase()
	var sections []Section
	for _, section := range file.File.Sections {
		size := uint64(section.VirtualSize)
		var data []byte
		if section.Offset != 0 {
			read, err := section.Data()
			if err != nil {
				return nil, fmt.Errorf("pe section %s: %w", section.Name, err)
			}
			data = read
		}
		if size < uint64(len(data)) {
			size = uint64(len(data))
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
