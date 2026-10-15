package iso

// Platform is the El Torito boot system.
//
//lewkit:enum
type Platform uint8

const (
	PlatformBIOS Platform = iota + 1 // bios
	PlatformPPC                      // ppc
	PlatformMac                      // mac
	PlatformEFI                      // efi
)

// Emulation is the El Torito boot medium.
//
//lewkit:enum
type Emulation uint8

const (
	EmulationNone      Emulation = iota + 1 // none
	EmulationFloppy12                       // floppy12
	EmulationFloppy144                      // floppy144
	EmulationFloppy288                      // floppy288
	EmulationHardDisk                       // harddisk
)
