package disasm

import (
	"debug/macho"
	"fmt"
	"io"
)

// MachOFile wraps [macho.File].
type MachOFile struct{ File *macho.File }

var _ Object = (*MachOFile)(nil)

func openMachO(reader io.ReaderAt) (*MachOFile, error) {
	file, err := macho.NewFile(reader)
	if err != nil {
		return nil, fmt.Errorf("macho: %w", err)
	}
	return &MachOFile{File: file}, nil
}

// Close closes the underlying file.
func (file *MachOFile) Close() error {
	return file.File.Close()
}

// Architecture maps the Mach-O CPU type to Capstone arch and mode.
func (file *MachOFile) Architecture() (Architecture, Mode, error) {
	switch file.File.Cpu {
	case macho.Cpu386:
		return ArchitectureX86, Mode32, nil
	case macho.CpuAmd64:
		return ArchitectureX86, Mode64, nil
	case macho.CpuArm:
		return ArchitectureARM, Mode32, nil
	case macho.CpuArm64:
		return ArchitectureAArch64, Mode64, nil
	case macho.CpuPpc:
		return ArchitecturePowerPC, Mode32, nil
	case macho.CpuPpc64:
		return ArchitecturePowerPC, Mode64, nil
	default:
		return 0, 0, fmt.Errorf("macho cpu %s", file.File.Cpu)
	}
}

// TextSection returns the named section, or __text.
func (file *MachOFile) TextSection(name string) (uint64, []byte, string, error) {
	if name == "" {
		name = "__text"
	}
	section := file.File.Section(name)
	if section == nil {
		return 0, nil, "", fmt.Errorf("%w: %s", ErrNoText, name)
	}
	data, err := section.Data()
	if err != nil {
		return 0, nil, "", fmt.Errorf("macho section %s: %w", section.Name, err)
	}
	return section.Addr, data, section.Name, nil
}

const (
	machoSectionPureInstructions = 0x80000000
	machoSectionSomeInstructions = 0x00000400
	machoLoadCommandMain         = 0x80000028
)

// Entry is the address of LC_MAIN relative to the __TEXT segment.
func (file *MachOFile) Entry() uint64 {
	var text uint64
	var offset uint64
	var found bool
	for _, load := range file.File.Loads {
		if segment, ok := load.(*macho.Segment); ok && segment.Name == "__TEXT" {
			text = segment.Addr
		}
		raw := load.Raw()
		if len(raw) < 24 || file.File.ByteOrder.Uint32(raw[:4]) != machoLoadCommandMain {
			continue
		}
		offset = file.File.ByteOrder.Uint64(raw[8:16])
		found = true
	}
	if !found {
		return 0
	}
	return text + offset
}

// Sections returns Mach-O sections.
func (file *MachOFile) Sections() ([]Section, error) {
	var sections []Section
	for _, section := range file.File.Sections {
		if section.Size == 0 {
			continue
		}
		var data []byte
		if section.Offset != 0 {
			read, err := section.Data()
			if err != nil {
				return nil, fmt.Errorf("macho section %s: %w", section.Name, err)
			}
			data = read
		}
		sections = append(sections, Section{
			Name:    section.Name,
			Address: section.Addr,
			Size:    section.Size,
			Bytes:   data,
			Exec:    machoExecutable(section),
			Read:    true,
		})
	}
	return sections, nil
}

// Symbols returns named addresses from the symbol table.
func (file *MachOFile) Symbols() Symbols {
	if file.File.Symtab == nil {
		return nil
	}
	executable := map[uint8]bool{}
	for index, section := range file.File.Sections {
		if machoExecutable(section) {
			executable[uint8(index+1)] = true
		}
	}
	var out Symbols
	for _, symbol := range file.File.Symtab.Syms {
		if symbol.Name == "" || symbol.Value == 0 {
			continue
		}
		out = append(out, Symbol{
			Name:    symbol.Name,
			Address: symbol.Value,
			Code:    executable[symbol.Sect],
		})
	}
	return out
}

func machoExecutable(section *macho.Section) bool {
	return section.Flags&machoSectionPureInstructions != 0 ||
		section.Flags&machoSectionSomeInstructions != 0 ||
		section.Name == "__text"
}
