package macho

import (
	"io"

	"github.com/lewtec/lewkit/x/errs"
)

var (
	ErrNilReader = errs.Wrap(errs.ErrInvalid, "macho: nil reader")
	ErrNilWriter = errs.Wrap(errs.ErrInvalid, "macho: nil writer")
	ErrNilFile   = errs.Wrap(errs.ErrInvalid, "macho: nil file")
	ErrName      = errs.Wrap(errs.ErrInvalid, "macho: invalid name")
	ErrSection   = errs.Wrap(errs.ErrInvalid, "macho: no section")
	ErrFormat    = errs.Wrap(errs.ErrSyntax, "macho: invalid file")
)

// Header is the Mach-O file header.
type Header struct {
	Magic    uint32
	CPU      CPU
	SubCPU   uint32
	Kind     Kind
	NCmd     uint32
	CmdSz    uint32
	Flags    uint32
	Reserved uint32
	Endian   Endian
}

// Command is one load command, in file order.
type Command struct {
	Cmd  uint32
	Data []byte
}

// Section is one section inside a segment.
type Section struct {
	Name    string
	Seg     string
	Addr    uint64
	Size    uint64
	Offset  uint32
	Align   uint32
	Reloff  uint32
	Nreloc  uint32
	Flags   uint32
	Bytes   []byte
	Symbols []*Symbol
}

// Symbol is one symbol table entry.
type Symbol struct {
	Name  string
	Type  uint8
	Sect  uint8
	Desc  uint16
	Value uint64
}

// Segment is one segment load command and the sections it contains.
type Segment struct {
	Name     string
	Addr     uint64
	Memsz    uint64
	Offset   uint64
	Filesz   uint64
	MaxProt  uint32
	Prot     uint32
	Flag     uint32
	Sections []*Section
	Bytes    []byte
}

// Dylib is one dynamic-library load command.
type Dylib struct {
	Cmd     uint32
	Name    string
	Time    uint32
	Current uint32
	Compat  uint32
}

// File is one Mach-O image.
type File struct {
	Header
	Commands   []Command
	Segments   []*Segment
	Symbols    []*Symbol
	Dylibs     []*Dylib
	EntryOff   uint64
	Stack      uint64
	SymOff     uint32
	StrOff     uint32
	SymBytes   []byte
	StrBytes   []byte
	Indirect   []byte
	IndirectAt uint32
	Span       uint64

	loaded bool
	cur    int
	sec    int
	entry  bool
	bad    error
}

// New returns an empty 64-bit relocatable file for AMD64.
func New() *File {
	return &File{
		Header: Header{
			Magic:  magic64,
			CPU:    Amd64,
			Kind:   TypeObj,
			Endian: Little,
		},
		cur: -1,
		sec: -1,
	}
}

// CPU sets the cpu type and returns the file.
func (f *File) CPU(cpu CPU) *File {
	if f == nil {
		return nil
	}
	f.Header.CPU = cpu
	if cpu&CPU(cpuArch64) != 0 {
		f.Magic = magic64
		return f
	}
	if cpu != 0 {
		f.Magic = magic32
	}
	return f
}

// Type sets the file kind and returns the file.
func (f *File) Type(kind Kind) *File {
	if f == nil {
		return nil
	}
	f.Kind = kind
	return f
}

// Dylib marks the file as a dynamic library and returns the file.
func (f *File) Dylib() *File {
	if f == nil {
		return nil
	}
	f.Kind = TypeDylib
	if f.idDylib() != nil {
		return f
	}
	id := &Dylib{Cmd: lcIDDylib, Name: "lib.dylib", Current: 0x10000, Compat: 0x10000}
	f.Dylibs = append([]*Dylib{id}, f.Dylibs...)
	return f
}

// Entry sets the LC_MAIN entry file offset and returns the file.
func (f *File) Entry(off uint64) *File {
	if f == nil {
		return nil
	}
	f.EntryOff = off
	f.entry = true
	return f
}

// Segment appends a segment, or selects one with the same name, and returns the file.
func (f *File) Segment(name string) *File {
	if f == nil {
		return nil
	}
	if name == "" || len(name) > 16 {
		f.bad = ErrName
		return f
	}
	for i, segment := range f.Segments {
		if segment.Name == name {
			f.cur = i
			f.sec = -1
			return f
		}
	}
	f.Segments = append(f.Segments, &Segment{Name: name})
	f.cur = len(f.Segments) - 1
	f.sec = -1
	return f
}

// Section appends a section on the current segment, or selects one with the same name.
func (f *File) Section(name string) *File {
	segment := f.currentSegment()
	if segment == nil {
		return f
	}
	if name == "" || len(name) > 16 {
		f.bad = ErrName
		return f
	}
	for i, section := range segment.Sections {
		if section.Name == name {
			f.sec = i
			return f
		}
	}
	segment.Sections = append(segment.Sections, &Section{Name: name, Seg: segment.Name})
	f.sec = len(segment.Sections) - 1
	return f
}

// Bytes sets the current section's contents and returns the file.
func (f *File) Bytes(data []byte) *File {
	section := f.currentSection()
	if section == nil {
		return f
	}
	section.Bytes = append([]byte(nil), data...)
	section.Size = uint64(len(section.Bytes))
	return f
}

// Symbol appends a symbol on the current section and returns the file.
func (f *File) Symbol(name string) *File {
	section := f.currentSection()
	if section == nil {
		return f
	}
	if name == "" {
		f.bad = ErrName
		return f
	}
	section.Symbols = append(section.Symbols, &Symbol{Name: name})
	return f
}

func (f *File) currentSegment() *Segment {
	if f == nil {
		return nil
	}
	if f.bad != nil {
		return nil
	}
	if f.cur < 0 || f.cur >= len(f.Segments) {
		f.bad = ErrSection
		return nil
	}
	return f.Segments[f.cur]
}

func (f *File) currentSection() *Section {
	segment := f.currentSegment()
	if segment == nil {
		return nil
	}
	if f.sec < 0 || f.sec >= len(segment.Sections) {
		f.bad = ErrSection
		return nil
	}
	return segment.Sections[f.sec]
}

func (f *File) idDylib() *Dylib {
	for _, item := range f.Dylibs {
		if item.Cmd == lcIDDylib {
			return item
		}
	}
	return nil
}

func (f *File) wide() bool {
	return f.Magic != magic32
}

// Take reads a Mach-O image into a [File].
func Take(r io.Reader) (*File, error) {
	if r == nil {
		return nil, ErrNilReader
	}
	return take(r)
}

// Make writes the file as a Mach-O image.
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
