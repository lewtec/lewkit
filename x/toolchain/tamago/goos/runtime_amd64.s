//go:build tamago && amd64

#include "go_asm.h"
#include "textflag.h"

// cpuinit is the link entry (-E cpuinit). The firmware has already entered
// long mode. Rebuild an identity map, turn on SSE, then start the runtime.
TEXT cpuinit(SB),NOSPLIT|NOFRAME,$0
	CLI

	// New tables, not the ones CR3 already uses (0x9000 on the MBR path).
	MOVQ	$0x20000, DI
	XORQ	AX, AX
	MOVQ	$0x3000, CX
	REP
	STOSB

	MOVQ	$0x20000, DI
	MOVQ	$(0x21000|3), (DI)
	MOVQ	$0x21000, DI
	MOVQ	$(0x22000|3), (DI)

	MOVQ	$0x22000, DI
	XORQ	AX, AX
map:
	MOVQ	AX, BX
	ORQ	$0x83, BX
	MOVQ	BX, (DI)
	ADDQ	$0x200000, AX
	ADDQ	$8, DI
	CMPQ	AX, $0x40000000
	JB	map

	MOVQ	CR4, AX
	ORQ	$(1<<5|1<<7|1<<9|1<<10), AX
	MOVQ	AX, CR4

	MOVQ	$0x20000, AX
	MOVQ	AX, CR3

	MOVQ	CR0, AX
	ANDQ	$~(1<<2), AX
	ORQ	$(1<<0|1<<1|1<<31), AX
	MOVQ	AX, CR0

	MOVQ	·RamStart(SB), SP
	ADDQ	·RamSize(SB), SP
	SUBQ	·RamStackOffset(SB), SP

	JMP	runtime·rt0_amd64_tamago(SB)

TEXT ·CPUInit(SB),NOSPLIT|NOFRAME,$0
	JMP	cpuinit(SB)

TEXT ·Hwinit0(SB),NOSPLIT|NOFRAME,$0
	RET

TEXT ·Printk(SB),NOSPLIT,$0-1
	MOVB	c+0(FP), BX
	MOVQ	$0x3fd, DX
wait:
	INB
	TESTB	$0x20, AX
	JZ	wait
	MOVQ	$0x3f8, DX
	MOVB	BX, AX
	OUTB
	RET

TEXT ·Nanotime(SB),NOSPLIT,$0-8
	RDTSC
	SHLQ	$32, DX
	ADDQ	DX, AX
	MOVQ	AX, ret+0(FP)
	RET
