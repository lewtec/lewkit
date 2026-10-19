package pe

import (
	"io"

	"github.com/lewtec/lewkit/x/errs"
)

var (
	ErrNilReader = errs.Wrap(errs.ErrInvalid, "pe: nil reader")
	ErrNilWriter = errs.Wrap(errs.ErrInvalid, "pe: nil writer")
	ErrNilFile   = errs.Wrap(errs.ErrInvalid, "pe: nil file")
	ErrName      = errs.Wrap(errs.ErrInvalid, "pe: invalid name")
	ErrSection   = errs.Wrap(errs.ErrInvalid, "pe: no section")
	ErrImport    = errs.Wrap(errs.ErrInvalid, "pe: no import")
	ErrFormat    = errs.Wrap(errs.ErrSyntax, "pe: invalid file")
)

// Header is the COFF and optional header fields the file round-trips.
type Header struct {
	Machine             Machine
	TimeDateStamp       uint32
	Characteristics     uint16
	Magic               uint16
	LinkerMajor         uint8
	LinkerMinor         uint8
	SizeOfCode          uint32
	SizeOfInitialized   uint32
	SizeOfUninitialized uint32
	Entry               uint32
	BaseOfCode          uint32
	BaseOfData          uint32
	ImageBase           uint64
	SectionAlign        uint32
	FileAlign           uint32
	OSMajor             uint16
	OSMinor             uint16
	ImageMajor          uint16
	ImageMinor          uint16
	SubsystemMajor      uint16
	SubsystemMinor      uint16
	Win32Version        uint32
	SizeOfImage         uint32
	SizeOfHeaders       uint32
	CheckSum            uint32
	Subsystem           Subsystem
	DllCharacteristics  uint16
	StackReserve        uint64
	StackCommit         uint64
	HeapReserve         uint64
	HeapCommit          uint64
	LoaderFlags         uint32
	Lfanew              uint32
	Optional            uint16
	SymbolCount         uint32
	SymbolOff           uint32
}

// Directory is one data directory entry.
type Directory struct {
	VirtualAddress uint32
	Size           uint32
}

// Section is one COFF section.
type Section struct {
	Name            string
	RawName         [8]byte
	VirtualSize     uint32
	VirtualAddress  uint32
	Size            uint32
	Offset          uint32
	RelocOff        uint32
	LineOff         uint32
	Relocs          uint16
	Lines           uint16
	Characteristics uint32
	Bytes           []byte

	auto bool
}

// Imported is one symbol imported from a DLL.
type Imported struct {
	Name      string
	Hint      uint16
	Ordinal   uint16
	ByOrdinal bool
}

// Import is one imported DLL and the symbols taken from it.
type Import struct {
	Library string
	Symbols []Imported
}

// Export is one exported symbol.
type Export struct {
	Name      string
	Ordinal   uint32
	RVA       uint32
	Forwarder string
}

// File is one PE image.
type File struct {
	Header
	Directories []Directory
	Sections    []*Section
	Imports     []*Import
	Exports     []Export
	DllName     string
	OrdinalBase uint32
	Symbols     []byte
	Span        uint64

	loaded bool
	cur    int
	imp    int
	exp    int
	bad    error
}

// New returns an empty PE32+ image for AMD64.
func New() *File {
	return &File{
		Header: Header{
			Machine:            Amd64,
			Magic:              magic64,
			Characteristics:    fileExecutable | fileLargeAddr,
			Subsystem:          SubsystemCUI,
			ImageBase:          0x140000000,
			SectionAlign:       0x1000,
			FileAlign:          0x200,
			OSMajor:            6,
			SubsystemMajor:     6,
			StackReserve:       0x100000,
			StackCommit:        0x1000,
			HeapReserve:        0x100000,
			HeapCommit:         0x1000,
			DllCharacteristics: 0x8160,
		},
		cur: -1,
		imp: -1,
		exp: -1,
	}
}

// Machine sets the COFF machine and returns the file.
func (f *File) Machine(machine Machine) *File {
	if f == nil {
		return nil
	}
	f.Header.Machine = machine
	if machine == I386 {
		f.Magic = magic32
		if f.Header.ImageBase == 0 || f.Header.ImageBase == 0x140000000 {
			f.Header.ImageBase = 0x400000
		}
		f.Characteristics |= file32Bit
	}
	return f
}

// Subsystem sets the optional-header subsystem and returns the file.
func (f *File) Subsystem(subsystem Subsystem) *File {
	if f == nil {
		return nil
	}
	f.Header.Subsystem = subsystem
	return f
}

// ImageBase sets the image base and returns the file.
func (f *File) ImageBase(base uint64) *File {
	if f == nil {
		return nil
	}
	f.Header.ImageBase = base
	return f
}

// Entry sets the entry-point RVA and returns the file.
func (f *File) Entry(rva uint32) *File {
	if f == nil {
		return nil
	}
	f.Header.Entry = rva
	return f
}

// DLL marks the image as a DLL and returns the file.
func (f *File) DLL() *File {
	if f == nil {
		return nil
	}
	f.Characteristics |= fileDLL
	return f
}

// Section appends a section, or selects one with the same name, and returns the file.
func (f *File) Section(name string) *File {
	if f == nil {
		return nil
	}
	if name == "" || len(name) > 8 {
		f.bad = ErrName
		return f
	}
	for i, section := range f.Sections {
		if section.Name == name {
			f.cur = i
			return f
		}
	}
	section := &Section{Name: name}
	copy(section.RawName[:], name)
	f.Sections = append(f.Sections, section)
	f.cur = len(f.Sections) - 1
	return f
}

// Exec marks the current section executable and returns the file.
func (f *File) Exec() *File {
	return f.addFlag(scnExec | scnCode)
}

// Read marks the current section readable and returns the file.
func (f *File) Read() *File {
	return f.addFlag(scnRead)
}

// Write marks the current section writable and returns the file.
func (f *File) Write() *File {
	return f.addFlag(scnWrite)
}

// Data marks the current section as initialized data and returns the file.
func (f *File) Data() *File {
	return f.addFlag(scnInit)
}

func (f *File) addFlag(flag uint32) *File {
	section := f.current()
	if section == nil {
		return f
	}
	section.Characteristics |= flag
	return f
}

// Addr sets the current section's RVA and returns the file.
func (f *File) Addr(rva uint32) *File {
	section := f.current()
	if section == nil {
		return f
	}
	section.VirtualAddress = rva
	return f
}

// Bytes sets the current section's contents and returns the file.
func (f *File) Bytes(data []byte) *File {
	section := f.current()
	if section == nil {
		return f
	}
	section.Bytes = append([]byte(nil), data...)
	section.VirtualSize = uint32(len(section.Bytes))
	section.Size = section.VirtualSize
	return f
}

// Import appends a DLL import and returns the file.
func (f *File) Import(library string) *File {
	if f == nil {
		return nil
	}
	if library == "" {
		f.bad = ErrName
		return f
	}
	for i, item := range f.Imports {
		if item.Library == library {
			f.imp = i
			return f
		}
	}
	f.Imports = append(f.Imports, &Import{Library: library})
	f.imp = len(f.Imports) - 1
	return f
}

// Symbol appends an imported symbol to the current DLL and returns the file.
func (f *File) Symbol(name string) *File {
	if f == nil {
		return nil
	}
	if f.bad != nil {
		return f
	}
	if f.imp < 0 || f.imp >= len(f.Imports) {
		f.bad = ErrImport
		return f
	}
	if name == "" {
		f.bad = ErrName
		return f
	}
	f.Imports[f.imp].Symbols = append(f.Imports[f.imp].Symbols, Imported{Name: name})
	return f
}

// ByOrdinal appends an ordinal import to the current DLL and returns the file.
func (f *File) ByOrdinal(ordinal uint16) *File {
	if f == nil {
		return nil
	}
	if f.bad != nil {
		return f
	}
	if f.imp < 0 || f.imp >= len(f.Imports) {
		f.bad = ErrImport
		return f
	}
	f.Imports[f.imp].Symbols = append(f.Imports[f.imp].Symbols, Imported{Ordinal: ordinal, ByOrdinal: true})
	return f
}

// Export appends an exported symbol and returns the file.
func (f *File) Export(name string) *File {
	if f == nil {
		return nil
	}
	if name == "" {
		f.bad = ErrName
		return f
	}
	f.Exports = append(f.Exports, Export{Name: name, Ordinal: uint32(len(f.Exports) + 1)})
	f.exp = len(f.Exports) - 1
	if f.OrdinalBase == 0 {
		f.OrdinalBase = 1
	}
	return f
}

// RVA sets the current export's address and returns the file.
func (f *File) RVA(rva uint32) *File {
	if f == nil {
		return nil
	}
	if f.bad != nil {
		return f
	}
	if f.exp < 0 || f.exp >= len(f.Exports) {
		f.bad = ErrSection
		return f
	}
	f.Exports[f.exp].RVA = rva
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

func (f *File) is64() bool {
	if f.Magic == magic32 {
		return false
	}
	if f.Magic == magic64 {
		return true
	}
	return f.Header.Machine != I386
}

// Take reads a PE image into a [File].
func Take(r io.Reader) (*File, error) {
	if r == nil {
		return nil, ErrNilReader
	}
	return take(r)
}

// Make writes the file as a PE image.
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
