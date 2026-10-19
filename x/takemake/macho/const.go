package macho

// CPU is the Mach-O cpu type.
type CPU uint32

const (
	CPUAny CPU = 0
	I386   CPU = 7
	Amd64  CPU = I386 | CPU(cpuArch64)
	Arm    CPU = 12
	Arm64  CPU = Arm | CPU(cpuArch64)
	PPC    CPU = 18
	PPC64  CPU = PPC | CPU(cpuArch64)
)

// Kind is the Mach-O file type.
type Kind uint32

const (
	TypeObj    Kind = 1
	TypeExec   Kind = 2
	TypeDylib  Kind = 6
	TypeBundle Kind = 8
)

// Endian is the file byte order.
type Endian uint8

const (
	Little Endian = 1
	Big    Endian = 2
)

const (
	cpuArch64 uint32 = 0x01000000

	magic32 uint32 = 0xfeedface
	magic64 uint32 = 0xfeedfacf

	lcSegment       uint32 = 0x1
	lcSymtab        uint32 = 0x2
	lcDysymtab      uint32 = 0xb
	lcLoadDylib     uint32 = 0xc
	lcIDDylib       uint32 = 0xd
	lcSegment64     uint32 = 0x19
	lcLazyDylib     uint32 = 0x20
	lcWeakDylib     uint32 = 0x80000018
	lcReexportDylib uint32 = 0x8000001f
	lcUpwardDylib   uint32 = 0x80000023
	lcMain          uint32 = 0x80000028

	header32      = 28
	header64      = 32
	segment32Size = 56
	segment64Size = 72
	section32Size = 68
	section64Size = 80
	symtabCmdSize = 24
	mainCmdSize   = 24
	dylibHead     = 24
	nlist32Size   = 12
	nlist64Size   = 16
	dysymtabSize  = 80

	sectionTypeMask    uint32 = 0xff
	sectionZerofill    uint32 = 0x1
	sectionGBZerofill  uint32 = 0xc
	sectionTLSZerofill uint32 = 0x12

	symExternal uint8 = 0x01
	symSectExt  uint8 = 0x0f
)
