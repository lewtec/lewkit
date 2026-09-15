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

// Symbols returns named addresses from the symbol table.
func (file *MachOFile) Symbols() Symbols {
	if file.File.Symtab == nil {
		return nil
	}
	var out Symbols
	for _, symbol := range file.File.Symtab.Syms {
		if symbol.Name == "" || symbol.Value == 0 {
			continue
		}
		out = append(out, Symbol{Name: symbol.Name, Address: symbol.Value})
	}
	return out
}
