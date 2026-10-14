//go:build tamago && amd64

package goos

import "unsafe"

func CPUInit()
func Hwinit0()
func Nanotime() int64
func Printk(c byte)

func inb(port uint16) byte
func outb(port uint16, val byte)
func outw(port uint16, val uint16)
func freeze()

var (
	Bloc    uintptr
	BlocMax uintptr
	Exit    func(code int32)
	Idle    func(until int64)
	ProcID  func() uint64
	Task    func(sp, mp, gp, fn unsafe.Pointer)
	Wake    func(procid uint64)
)

var rng uint64 = 0x123456789abcdef0

func InitRNG() {}

func GetRandomData(b []byte) {
	for i := range b {
		rng = rng*6364136223846793005 + 1
		b[i] = byte(rng >> 33)
	}
}

func Hwinit1() {
	Exit = halt
}

const (
	comData  = 0x3f8
	comLSR   = 0x3fd
	psData   = 0x60
	psStatus = 0x64
)

// Getc returns the next console byte.
// A PS/2 Enter key is '\n'. A UART byte is returned as it was received.
//
//go:linkname Getc
func Getc() byte {
	for {
		if inb(comLSR)&1 != 0 {
			return inb(comData)
		}
		st := inb(psStatus)
		if st&1 == 0 {
			continue
		}
		b := inb(psData)
		// Bit 5 marks a mouse byte. It still has to leave the buffer.
		if st&0x20 != 0 || b != 0x1C {
			continue
		}
		return '\n'
	}
}

func halt(code int32) {
	// ACPI power-off on the default QEMU machine. OVMF ignores that port,
	// so the runner's isa-debug-exit device is 0xf4. A clean exit writes
	// 0x21; qemu then returns (0x21<<1)|1.
	outw(0x604, 0x2000)
	if code == 0 {
		outb(0xf4, 0x21)
	} else {
		outb(0xf4, byte(code))
	}
	for {
		freeze()
	}
}
