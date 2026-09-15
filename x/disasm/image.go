package disasm

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// ErrUnknownFormat is [ReadText] on bytes that are not ELF, PE, or Mach-O.
var ErrUnknownFormat = errors.New("unknown object format")

// ErrNoText is [ReadText] when the named or default text section is missing.
var ErrNoText = errors.New("no text section")

// Text is a code slice taken from an object file.
type Text struct {
	Architecture Architecture
	Mode         Mode
	Address      uint64
	Bytes        []byte
	Section      string
	Format       string
	Symbols      Symbols
}

// ReadText reads the text section of an ELF, PE, or Mach-O image.
// An empty section name selects .text, __text, or the first executable
// section, in that order.
func ReadText(reader io.ReaderAt, size int64, section string) (Text, error) {
	if size < 4 {
		return Text{}, ErrUnknownFormat
	}
	var magic [4]byte
	if _, err := reader.ReadAt(magic[:], 0); err != nil {
		return Text{}, err
	}
	switch {
	case string(magic[:]) == elf.ELFMAG:
		return readELF(reader, section)
	case magic[0] == 'M' && magic[1] == 'Z':
		return readPE(reader, section)
	case machoMagic(magic):
		return readMacho(reader, section)
	default:
		return Text{}, ErrUnknownFormat
	}
}

func machoMagic(magic [4]byte) bool {
	value := binary.LittleEndian.Uint32(magic[:])
	switch value {
	case 0xfeedface, 0xfeedfacf, 0xcefaedfe, 0xcffaedfe:
		return true
	default:
		return false
	}
}

func readELF(reader io.ReaderAt, section string) (Text, error) {
	file, err := elf.NewFile(reader)
	if err != nil {
		return Text{}, fmt.Errorf("elf: %w", err)
	}
	defer file.Close()
	architecture, mode, err := elfArchitecture(file)
	if err != nil {
		return Text{}, err
	}
	elfSection, err := pickELF(file, section)
	if err != nil {
		return Text{}, err
	}
	data, err := elfSection.Data()
	if err != nil {
		return Text{}, fmt.Errorf("elf section %s: %w", elfSection.Name, err)
	}
	return Text{
		Architecture: architecture,
		Mode:         mode,
		Address:      elfSection.Addr,
		Bytes:        data,
		Section:      elfSection.Name,
		Format:       "elf",
		Symbols:      (&ELFFile{File: file}).Symbols(),
	}, nil
}

func elfArchitecture(file *elf.File) (Architecture, Mode, error) {
	var mode Mode
	switch file.Class {
	case elf.ELFCLASS32:
		mode = Mode32
	case elf.ELFCLASS64:
		mode = Mode64
	}
	if file.Data == elf.ELFDATA2MSB {
		mode |= ModeBigEndian
	}
	switch file.Machine {
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
		return 0, 0, fmt.Errorf("elf machine %s", file.Machine)
	}
}

func riscvMode(mode Mode) Mode {
	if mode&Mode64 != 0 {
		return ModeRISCV64
	}
	return ModeRISCV32
}

func pickELF(file *elf.File, name string) (*elf.Section, error) {
	if name != "" {
		section := file.Section(name)
		if section == nil {
			return nil, fmt.Errorf("%w: %s", ErrNoText, name)
		}
		return section, nil
	}
	if section := file.Section(".text"); section != nil {
		return section, nil
	}
	for _, section := range file.Sections {
		if section.Flags&elf.SHF_EXECINSTR != 0 && section.Type == elf.SHT_PROGBITS {
			return section, nil
		}
	}
	return nil, ErrNoText
}

func readPE(reader io.ReaderAt, section string) (Text, error) {
	file, err := pe.NewFile(reader)
	if err != nil {
		return Text{}, fmt.Errorf("pe: %w", err)
	}
	defer file.Close()
	architecture, mode, err := peArchitecture(file)
	if err != nil {
		return Text{}, err
	}
	peSection, err := pickPE(file, section)
	if err != nil {
		return Text{}, err
	}
	data, err := peSection.Data()
	if err != nil {
		return Text{}, fmt.Errorf("pe section %s: %w", peSection.Name, err)
	}
	return Text{
		Architecture: architecture,
		Mode:         mode,
		Address:      uint64(peSection.VirtualAddress),
		Bytes:        data,
		Section:      peSection.Name,
		Format:       "pe",
		Symbols:      (&PEFile{File: file}).Symbols(),
	}, nil
}

func peArchitecture(file *pe.File) (Architecture, Mode, error) {
	mode := Mode32
	switch file.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		mode = Mode32
	case *pe.OptionalHeader64:
		mode = Mode64
	}
	switch file.Machine {
	case pe.IMAGE_FILE_MACHINE_I386:
		return ArchitectureX86, Mode32, nil
	case pe.IMAGE_FILE_MACHINE_AMD64:
		return ArchitectureX86, Mode64, nil
	case pe.IMAGE_FILE_MACHINE_ARM:
		return ArchitectureARM, mode, nil
	case pe.IMAGE_FILE_MACHINE_ARM64:
		return ArchitectureAArch64, mode, nil
	default:
		return 0, 0, fmt.Errorf("pe machine %#x", file.Machine)
	}
}

func pickPE(file *pe.File, name string) (*pe.Section, error) {
	if name == "" {
		name = ".text"
	}
	for _, section := range file.Sections {
		if section.Name == name {
			return section, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrNoText, name)
}

func readMacho(reader io.ReaderAt, section string) (Text, error) {
	file, err := macho.NewFile(reader)
	if err != nil {
		return Text{}, fmt.Errorf("macho: %w", err)
	}
	defer file.Close()
	architecture, mode, err := machoArchitecture(file)
	if err != nil {
		return Text{}, err
	}
	machoSection, err := pickMacho(file, section)
	if err != nil {
		return Text{}, err
	}
	data, err := machoSection.Data()
	if err != nil {
		return Text{}, fmt.Errorf("macho section %s: %w", machoSection.Name, err)
	}
	return Text{
		Architecture: architecture,
		Mode:         mode,
		Address:      machoSection.Addr,
		Bytes:        data,
		Section:      machoSection.Name,
		Format:       "macho",
		Symbols:      (&MachOFile{File: file}).Symbols(),
	}, nil
}

func machoArchitecture(file *macho.File) (Architecture, Mode, error) {
	switch file.Cpu {
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
		return 0, 0, fmt.Errorf("macho cpu %s", file.Cpu)
	}
}

func pickMacho(file *macho.File, name string) (*macho.Section, error) {
	if name == "" {
		name = "__text"
	}
	if section := file.Section(name); section != nil {
		return section, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrNoText, name)
}

// ELFFile wraps [elf.File].
type ELFFile struct{ File *elf.File }

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

// PEFile wraps [pe.File].
type PEFile struct{ File *pe.File }

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
		})
		i += int(symbol.NumberOfAuxSymbols)
	}
	return out
}

// MachOFile wraps [macho.File].
type MachOFile struct{ File *macho.File }

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
