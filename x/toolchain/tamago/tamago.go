// Package tamago is the bare-metal amd64 target.
//
// LinkFlags are the linker arguments for the firmware image. Pass them
// as x/toolchain/go Image.Ldflags. The text address is RamStart plus
// 64KiB. Package goos in this directory is the runtime/goos overlay:
// the Go driver compiles it as runtime/goos. The stock linux runtime
// archives cannot link this image.
package tamago

// LinkFlags is the amd64 firmware link. cpuinit is the entry. The text
// sits at 0x10010000, which is RamStart plus 64KiB, aligned to 4KiB.
var LinkFlags = []string{"-E", "cpuinit", "-T", "0x10010000", "-R", "0x1000"}

const (
	// RamStart is the identity-mapped window shared by the MBR loader
	// and the UEFI image base.
	RamStart = 0x10000000
	// RamSize is the mapped RAM.
	RamSize = 0x20000000
	// RamStackOffset is the gap between the end of RAM and the initial stack.
	RamStackOffset = 0x100000
)
