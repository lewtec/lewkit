package disasm

import (
	"debug/elf"
	"fmt"
)

// ELFFile wraps [elf.File].
type ELFFile struct{ File *elf.File }

var _ Object = (*ELFFile)(nil)

// Close closes the underlying file.
func (file *ELFFile) Close() error {
	return file.File.Close()
}

// Architecture maps the ELF header to Capstone arch and mode.
func (file *ELFFile) Architecture() (Architecture, Mode, error) {
	var mode Mode
	switch file.File.Class {
	case elf.ELFCLASS32:
		mode = Mode32
	case elf.ELFCLASS64:
		mode = Mode64
	}
	if file.File.Data == elf.ELFDATA2MSB {
		mode |= ModeBigEndian
	}
	switch file.File.Machine {
	case elf.EM_386:
		return ArchitectureX86, mode&^Mode64 | Mode32, nil
	case elf.EM_X86_64:
		return ArchitectureX86, mode&^Mode32 | Mode64, nil
	case elf.EM_ARM:
		return ArchitectureARM, mode, nil
	case elf.EM_AARCH64:
		return ArchitectureAArch64, mode, nil
	case elf.EM_MIPS, elf.EM_MIPS_RS3_LE:
		return ArchitectureMIPS, mode, nil
	case elf.EM_PPC:
		return ArchitecturePowerPC, mode&^Mode64 | Mode32, nil
	case elf.EM_PPC64:
		return ArchitecturePowerPC, mode&^Mode32 | Mode64, nil
	case elf.EM_S390:
		return ArchitectureSystemZ, mode, nil
	case elf.EM_RISCV:
		return ArchitectureRISCV, riscvMode(mode), nil
	case elf.EM_BPF:
		return ArchitectureBPF, ModeBPFExtended, nil
	default:
		return 0, 0, fmt.Errorf("elf machine %s", file.File.Machine)
	}
}

func riscvMode(mode Mode) Mode {
	if mode&Mode64 != 0 {
		return ModeRISCV64
	}
	return ModeRISCV32
}

// TextSection returns the named section, or .text / first executable PROGBITS.
func (file *ELFFile) TextSection(name string) (uint64, []byte, string, error) {
	section, err := file.pick(name)
	if err != nil {
		return 0, nil, "", err
	}
	data, err := section.Data()
	if err != nil {
		return 0, nil, "", fmt.Errorf("elf section %s: %w", section.Name, err)
	}
	return section.Addr, data, section.Name, nil
}

func (file *ELFFile) pick(name string) (*elf.Section, error) {
	if name != "" {
		section := file.File.Section(name)
		if section == nil {
			return nil, fmt.Errorf("%w: %s", ErrNoText, name)
		}
		return section, nil
	}
	if section := file.File.Section(".text"); section != nil {
		return section, nil
	}
	for _, section := range file.File.Sections {
		if section.Flags&elf.SHF_EXECINSTR != 0 && section.Type == elf.SHT_PROGBITS {
			return section, nil
		}
	}
	return nil, ErrNoText
}

// Symbols returns function symbols first, then other named addresses.
func (file *ELFFile) Symbols() Symbols {
	var functions, others Symbols
	add := func(list []elf.Symbol, err error) {
		if err != nil {
			return
		}
		for _, symbol := range list {
			if symbol.Name == "" || symbol.Value == 0 {
				continue
			}
			switch elf.ST_TYPE(symbol.Info) {
			case elf.STT_FILE, elf.STT_SECTION:
				continue
			case elf.STT_FUNC:
				functions = append(functions, Symbol{Name: symbol.Name, Address: symbol.Value})
			default:
				others = append(others, Symbol{Name: symbol.Name, Address: symbol.Value})
			}
		}
	}
	list, err := file.File.Symbols()
	add(list, err)
	list, err = file.File.DynamicSymbols()
	add(list, err)
	return append(functions, others...)
}
