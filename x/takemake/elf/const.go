package elf

// Class is the ELF class.
type Class uint8

const (
	ClassNone Class = 0
	Class32   Class = 1
	Class64   Class = 2
)

// Endian is the ELF data encoding.
type Endian uint8

const (
	EndianNone Endian = 0
	Little     Endian = 1
	Big        Endian = 2
)

// Kind is e_type.
type Kind uint16

const (
	KindNone Kind = 0
	Rel      Kind = 1
	Exec     Kind = 2
	Dyn      Kind = 3
	Core     Kind = 4
)

// Machine is e_machine.
type Machine uint16

const (
	MachineNone Machine = 0
	MIPS        Machine = 8
	PPC         Machine = 20
	PPC64       Machine = 21
	S390        Machine = 22
	ARM         Machine = 40
	X86_64      Machine = 62
	AARCH64     Machine = 183
	RISCV       Machine = 243
	BPF         Machine = 247
	I386        Machine = 3
)

// Section kinds are sh_type values. OS-specific types round-trip as their number.
const (
	SectNone     uint32 = 0
	SectProgBits uint32 = 1
	SectSymTab   uint32 = 2
	SectStrTab   uint32 = 3
	SectRela     uint32 = 4
	SectHash     uint32 = 5
	SectDynamic  uint32 = 6
	SectNote     uint32 = 7
	SectNoBits   uint32 = 8
	SectRel      uint32 = 9
	SectDynSym   uint32 = 11
)

// Section flags are sh_flags bits.
const (
	FlagWrite uint64 = 0x1
	FlagAlloc uint64 = 0x2
	FlagExec  uint64 = 0x4
)

// Segment kinds are p_type values.
const (
	ProgNull    uint32 = 0
	ProgLoad    uint32 = 1
	ProgDynamic uint32 = 2
	ProgInterp  uint32 = 3
	ProgNote    uint32 = 4
	ProgPhdr    uint32 = 6
	ProgTLS     uint32 = 7
)

// Segment permissions are p_flags bits.
const (
	PermExec  uint32 = 0x1
	PermWrite uint32 = 0x2
	PermRead  uint32 = 0x4
)

// Symbol binding and type live in the info byte.
const (
	BindLocal  uint8 = 0
	BindGlobal uint8 = 1
	BindWeak   uint8 = 2
	SymNoType  uint8 = 0
	SymObject  uint8 = 1
	SymFunc    uint8 = 2
	SymSection uint8 = 3
	SymFile    uint8 = 4
)

// Special section indices from the symbol table.
const (
	SHNUndef  uint16 = 0
	SHNAbs    uint16 = 0xfff1
	SHNCommon uint16 = 0xfff2
)
