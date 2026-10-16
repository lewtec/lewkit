package elf

import (
	"encoding/binary"
	"io"

	"github.com/lewtec/lewkit/x/errs"
)

var (
	ErrNilReader = errs.Wrap(errs.ErrInvalid, "elf: nil reader")
	ErrNilWriter = errs.Wrap(errs.ErrInvalid, "elf: nil writer")
	ErrNilFile   = errs.Wrap(errs.ErrInvalid, "elf: nil file")
	ErrName      = errs.Wrap(errs.ErrInvalid, "elf: invalid name")
	ErrSection   = errs.Wrap(errs.ErrInvalid, "elf: no section")
	ErrSymbol    = errs.Wrap(errs.ErrInvalid, "elf: no symbol")
	ErrFormat    = errs.Wrap(errs.ErrSyntax, "elf: invalid file")
)

// Header is the ELF file header, without the section or program tables.
type Header struct {
	Class   Class
	Endian  Endian
	OSABI   uint8
	ABI     uint8
	Kind    Kind
	Machine Machine
	Entry   uint64
	Flags   uint32
}

// Symbol is one symbol table entry other than the null symbol.
// Section is the defining section index, or [SHNAbs], [SHNCommon], or [SHNUndef].
type Symbol struct {
	Name    string
	Info    uint8
	Other   uint8
	Section uint16
	Value   uint64
	Size    uint64
	Dynamic bool
}

// Reloc is one relocation.
// Sym is the symbol's index in its table. Zero means the reloc has no symbol.
type Reloc struct {
	Offset    uint64
	Sym       uint32
	SymName   string
	Dynamic   bool
	Kind      uint32
	Addend    int64
	HasAddend bool
}

// Section is one section header and its payload.
// Symbols defined by this section hang off it, in table order.
// Relocs are the relocations whose target is this section.
type Section struct {
	Name    string
	NameOff uint32
	Kind    uint32
	Flags   uint64
	Addr    uint64
	Offset  uint64
	Size    uint64
	Link    uint32
	Info    uint32
	Align   uint64
	Entsize uint64
	Bytes   []byte
	Symbols []*Symbol
	Relocs  []*Reloc
}

// Segment is one program header.
// Covers lists the section indices whose file or address range meets this segment.
type Segment struct {
	Type   uint32
	Flags  uint32
	Off    uint64
	Vaddr  uint64
	Paddr  uint64
	Filesz uint64
	Memsz  uint64
	Align  uint64
	Covers []int
}

// File is one ELF image.
type File struct {
	Header
	PhOff    uint64
	ShOff    uint64
	ShStr    uint16
	Span     uint64
	Sections []*Section
	Segments []*Segment
	Symbols  []*Symbol
	Relocs   []*Reloc

	loaded bool
	cur    int
	seg    int
	sym    *Symbol
	bad    error
}

// New returns an empty relocatable ELF64 image.
// The null section is already present, so the first real section is index 1.
func New() *File {
	return &File{
		Header: Header{
			Class:   Class64,
			Endian:  Little,
			Kind:    Rel,
			Machine: X86_64,
		},
		Sections: []*Section{{}},
		cur:      -1,
		seg:      -1,
	}
}

// Class32 selects ELF32 and returns the file.
func (f *File) Class32() *File {
	if f == nil {
		return nil
	}
	f.Class = Class32
	return f
}

// Class64 selects ELF64 and returns the file.
func (f *File) Class64() *File {
	if f == nil {
		return nil
	}
	f.Class = Class64
	return f
}

// Little selects little-endian and returns the file.
func (f *File) Little() *File {
	if f == nil {
		return nil
	}
	f.Endian = Little
	return f
}

// Big selects big-endian and returns the file.
func (f *File) Big() *File {
	if f == nil {
		return nil
	}
	f.Endian = Big
	return f
}

// Machine sets e_machine and returns the file.
func (f *File) Machine(machine Machine) *File {
	if f == nil {
		return nil
	}
	f.Header.Machine = machine
	return f
}

// Type sets e_type and returns the file.
func (f *File) Type(kind Kind) *File {
	if f == nil {
		return nil
	}
	f.Kind = kind
	return f
}

// Entry sets e_entry and returns the file.
func (f *File) Entry(addr uint64) *File {
	if f == nil {
		return nil
	}
	f.Header.Entry = addr
	return f
}

// Section appends a section, or selects one with the same name, and returns the file.
func (f *File) Section(name string) *File {
	if f == nil {
		return nil
	}
	if name == "" {
		f.bad = ErrName
		return f
	}
	for i, section := range f.Sections {
		if section.Name == name {
			f.cur = i
			return f
		}
	}
	f.Sections = append(f.Sections, &Section{
		Name:  name,
		Kind:  SectProgBits,
		Align: 1,
	})
	f.cur = len(f.Sections) - 1
	return f
}

// SectKind sets the current section's sh_type and returns the file.
func (f *File) SectKind(kind uint32) *File {
	section := f.current()
	if section == nil {
		return f
	}
	section.Kind = kind
	return f
}

// Alloc marks the current section allocated and returns the file.
func (f *File) Alloc() *File {
	return f.addFlag(FlagAlloc)
}

// Exec marks the current section executable and returns the file.
func (f *File) Exec() *File {
	return f.addFlag(FlagExec)
}

// Write marks the current section writable and returns the file.
func (f *File) Write() *File {
	return f.addFlag(FlagWrite)
}

func (f *File) addFlag(flag uint64) *File {
	section := f.current()
	if section == nil {
		return f
	}
	section.Flags |= flag
	if section.Align < 16 {
		section.Align = 16
	}
	return f
}

// Addr sets the current section's address and returns the file.
func (f *File) Addr(addr uint64) *File {
	section := f.current()
	if section == nil {
		return f
	}
	section.Addr = addr
	return f
}

// Align sets the current section's alignment and returns the file.
func (f *File) Align(align uint64) *File {
	section := f.current()
	if section == nil {
		return f
	}
	if align == 0 {
		align = 1
	}
	section.Align = align
	return f
}

// Bytes sets the current section's file payload and returns the file.
func (f *File) Bytes(data []byte) *File {
	section := f.current()
	if section == nil {
		return f
	}
	section.Bytes = append([]byte(nil), data...)
	section.Size = uint64(len(section.Bytes))
	return f
}

// Symbol appends a symbol defined by the current section and returns the file.
func (f *File) Symbol(name string) *File {
	section := f.current()
	if section == nil {
		return f
	}
	if name == "" {
		f.bad = ErrName
		return f
	}
	symbol := &Symbol{
		Name:    name,
		Section: uint16(f.cur),
	}
	section.Symbols = append(section.Symbols, symbol)
	f.sym = symbol
	return f
}

// Func marks the current symbol as a function and returns the file.
func (f *File) Func() *File {
	return f.setSym(func(symbol *Symbol) {
		symbol.Info = symbol.Info&0xf0 | SymFunc
	})
}

// Object marks the current symbol as a data object and returns the file.
func (f *File) Object() *File {
	return f.setSym(func(symbol *Symbol) {
		symbol.Info = symbol.Info&0xf0 | SymObject
	})
}

// Global marks the current symbol global and returns the file.
func (f *File) Global() *File {
	return f.setSym(func(symbol *Symbol) {
		symbol.Info = symbol.Info&0x0f | BindGlobal<<4
	})
}

// Local marks the current symbol local and returns the file.
func (f *File) Local() *File {
	return f.setSym(func(symbol *Symbol) {
		symbol.Info = symbol.Info&0x0f | BindLocal<<4
	})
}

// Weak marks the current symbol weak and returns the file.
func (f *File) Weak() *File {
	return f.setSym(func(symbol *Symbol) {
		symbol.Info = symbol.Info&0x0f | BindWeak<<4
	})
}

// SymSize sets the current symbol's size and returns the file.
func (f *File) SymSize(size uint64) *File {
	return f.setSym(func(symbol *Symbol) {
		symbol.Size = size
	})
}

// Value sets the current symbol's value and returns the file.
func (f *File) Value(value uint64) *File {
	return f.setSym(func(symbol *Symbol) {
		symbol.Value = value
	})
}

func (f *File) setSym(apply func(*Symbol)) *File {
	if f == nil {
		return nil
	}
	if f.bad != nil {
		return f
	}
	if f.sym == nil {
		f.bad = ErrSymbol
		return f
	}
	apply(f.sym)
	return f
}

// Reloc appends a relocation against the current section and returns the file.
// sym is a symbol name in the static table. kind is the architecture-specific type.
func (f *File) Reloc(offset uint64, kind uint32, sym string, addend int64) *File {
	section := f.current()
	if section == nil {
		return f
	}
	section.Relocs = append(section.Relocs, &Reloc{
		Offset:    offset,
		SymName:   sym,
		Kind:      kind,
		Addend:    addend,
		HasAddend: true,
	})
	return f
}

// Segment appends a program header and returns the file.
func (f *File) Segment(kind uint32) *File {
	if f == nil {
		return nil
	}
	f.Segments = append(f.Segments, &Segment{Type: kind})
	f.seg = len(f.Segments) - 1
	return f
}

// Perm sets the current segment's permission bits and returns the file.
func (f *File) Perm(flags uint32) *File {
	if f == nil {
		return nil
	}
	if f.seg < 0 || f.seg >= len(f.Segments) {
		f.bad = ErrSection
		return f
	}
	f.Segments[f.seg].Flags = flags
	return f
}

// At sets the current segment's virtual address and returns the file.
func (f *File) At(addr uint64) *File {
	if f == nil {
		return nil
	}
	if f.seg < 0 || f.seg >= len(f.Segments) {
		f.bad = ErrSection
		return f
	}
	f.Segments[f.seg].Vaddr = addr
	f.Segments[f.seg].Paddr = addr
	return f
}

// Cover records that the current segment covers the named section.
func (f *File) Cover(name string) *File {
	if f == nil {
		return nil
	}
	if f.seg < 0 || f.seg >= len(f.Segments) {
		f.bad = ErrSection
		return f
	}
	for i, section := range f.Sections {
		if section.Name != name {
			continue
		}
		f.Segments[f.seg].Covers = append(f.Segments[f.seg].Covers, i)
		return f
	}
	f.bad = ErrName
	return f
}

func (f *File) current() *Section {
	if f == nil {
		return nil
	}
	if f.bad != nil {
		return nil
	}
	if f.cur < 0 || f.cur >= len(f.Sections) {
		f.bad = ErrSection
		return nil
	}
	return f.Sections[f.cur]
}

func (f *File) order() binary.ByteOrder {
	if f.Endian == Big {
		return binary.BigEndian
	}
	return binary.LittleEndian
}

func (f *File) dims() (eh, ph, sh int) {
	if f.Class == Class32 {
		return 52, 32, 40
	}
	return 64, 56, 64
}

// Take reads an ELF image into a [File].
func Take(r io.Reader) (*File, error) {
	if r == nil {
		return nil, ErrNilReader
	}
	return take(r)
}

// Make writes the file as an ELF image.
func (f *File) Make(w io.Writer) error {
	if f == nil {
		return ErrNilFile
	}
	if w == nil {
		return ErrNilWriter
	}
	if f.bad != nil {
		return f.bad
	}
	return f.clone().emit(w)
}
