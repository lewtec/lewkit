// Unikernel prints one line and waits until Enter.
//
// The ELF is compiled by the embedded toolchain. TAMAGO lists the
// GOOS=tamago archives with GOOSPKG pointed at the runtime/goos overlay,
// so the runtime is built against that overlay. Returning from main
// powers the machine off, which firmware without a power switch reboots.
//
//	TAMAGO=/path/to/tamago/bin/go go run ./cmd/lewkit release build --config ./examples/unikernel/eletrocromo.json --goos mbr --goarch amd64
//	QEMU_SYSTEM=qemu-system-x86_64 go run ./cmd/lewkit release run --config ./examples/unikernel/eletrocromo.json --goos mbr --goarch amd64
//
// uefi is the same flags with --goos uefi. Run also needs OVMF_CODE and OVMF_VARS.
// linux_iso is the Linux guest CD: --goos linux_iso --goarch amd64.
package main

import "fmt"

func main() {
	fmt.Println("hello from the unikernel")
	waitEnter()
}
