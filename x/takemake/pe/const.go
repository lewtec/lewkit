package pe

// Machine is the COFF machine.
type Machine uint16

const (
	MachineNone Machine = 0
	I386        Machine = 0x14c
	Arm         Machine = 0x1c0
	Amd64       Machine = 0x8664
	Arm64       Machine = 0xaa64
)

// Subsystem is the optional-header subsystem.
type Subsystem uint16

const (
	SubsystemUnknown Subsystem = 0
	SubsystemNative  Subsystem = 1
	SubsystemGUI     Subsystem = 2
	SubsystemCUI     Subsystem = 3
	SubsystemEFI     Subsystem = 10
)

const (
	magic32 uint16 = 0x10b
	magic64 uint16 = 0x20b

	fileExecutable uint16 = 0x0002
	fileLargeAddr  uint16 = 0x0020
	file32Bit      uint16 = 0x0100
	fileDLL        uint16 = 0x2000

	scnCode    uint32 = 0x00000020
	scnInit    uint32 = 0x00000040
	scnUninit  uint32 = 0x00000080
	scnDiscard uint32 = 0x02000000
	scnExec    uint32 = 0x20000000
	scnRead    uint32 = 0x40000000
	scnWrite   uint32 = 0x80000000

	dirExport = 0
	dirImport = 1
	dirIAT    = 12
	dirCount  = 16
)
