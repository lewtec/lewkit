//go:build tamago && amd64

package goos

const (
	ArenaBaseOffset     = 0
	HeapAddrBits        = 40
	LogHeapArenaBytes   = 22
	LogPallocChunkPages = 9
	MinPhysPageSize     = 4096
	StackSystem         = 512
)

var (
	// Identity-mapped RAM the bootloader and the UEFI image base share.
	// The linked text is 0x10010000, inside this window.
	RamStart       uint = 0x10000000
	RamSize        uint = 0x20000000
	RamStackOffset uint = 0x100000
)
