//go:build (linux || android) && clibexport

#include "textflag.h"

// call is the AAPCS hard-float entry. The trampoline saved R4 at 0(SP)
// and left the handler id in R4, with SP eight bytes below the C entry.
// R0-R3 are the integer arguments. F0-F7 are D0-D7. R4, R5, and R7 are
// callee-saved and this function uses them for mmap and the syscall number.
//
// The C stack is 8-byte aligned on entry to the trampoline. The frame
// here is 192 bytes and keeps that alignment.
TEXT ·call(SB),NOSPLIT|NOFRAME,$0
	MOVW R4, R12
	MOVW 0(R13), R4
	ADD $8, R13

	SUB $192, R13
	MOVW R14, 168(R13)
	MOVW R7, 172(R13)
	MOVW R4, 176(R13)
	MOVW R5, 180(R13)

	MOVW R12, 0(R13)
	MOVW $0, R12
	MOVW R12, 4(R13)
	MOVW R0, 8(R13)
	MOVW R12, 12(R13)
	MOVW R1, 16(R13)
	MOVW R12, 20(R13)
	MOVW R2, 24(R13)
	MOVW R12, 28(R13)
	MOVW R3, 32(R13)
	MOVW R12, 36(R13)
	MOVW R12, 40(R13)
	MOVW R12, 44(R13)
	MOVW R12, 48(R13)
	MOVW R12, 52(R13)
	MOVW R12, 56(R13)
	MOVW R12, 60(R13)
	MOVW R12, 64(R13)
	MOVW R12, 68(R13)
	MOVD F0, 72(R13)
	MOVD F1, 80(R13)
	MOVD F2, 88(R13)
	MOVD F3, 96(R13)
	MOVD F4, 104(R13)
	MOVD F5, 112(R13)
	MOVD F6, 120(R13)
	MOVD F7, 128(R13)
	MOVW R12, 136(R13)
	MOVW R12, 140(R13)
	MOVW R12, 144(R13)
	MOVW R12, 148(R13)
	MOVW R12, 152(R13)
	MOVW R12, 156(R13)

	CALL ·ensure(SB)

	MOVW ·rtReady(SB), R0
	CMP $1, R0
	BNE callFail

	MOVW R13, 160(R13)
	MOVW $0, R12
	MOVW R12, 164(R13)
	MOVW R13, R4
	ADD $160, R4
	MOVW $8, R5
callWrite:
	MOVW ·pipeW(SB), R0
	MOVW R4, R1
	MOVW R5, R2
	MOVW $4, R7
	SWI $0
	CMP $0, R0
	BLE callWriteErr
	ADD R0, R4, R4
	SUB R0, R5, R5
	CMP $0, R5
	BNE callWrite
	JMP callWait
callWriteErr:
	CMP $-4, R0
	BEQ callWrite
	JMP callFail

callWait:
	MOVW 152(R13), R0
	CMP $0, R0
	BNE callDone
	MOVW R13, R0
	ADD $152, R0
	MOVW $128, R1
	MOVW $0, R2
	MOVW $0, R3
	MOVW $0, R4
	MOVW $0, R5
	MOVW $240, R7
	SWI $0
	JMP callWait

callDone:
	MOVW 136(R13), R0
	MOVW 140(R13), R1
	MOVD 144(R13), F0
	JMP callRestore

callFail:
	MOVW $0, R0
	MOVW $0, R1
	MOVD 144(R13), F0

callRestore:
	MOVW 168(R13), R14
	MOVW 172(R13), R7
	MOVW 176(R13), R4
	MOVW 180(R13), R5
	ADD $192, R13
	RET

// ensure boots the runtime once. Losers, and the winner, wait until
// rtReady leaves 0.
TEXT ·ensure(SB),NOSPLIT|NOFRAME,$0
	MOVW ·rtReady(SB), R0
	CMP $0, R0
	BNE ensureDone

	SUB $24, R13
	MOVW R14, 0(R13)
	MOVW R7, 4(R13)
	MOVW R4, 8(R13)
	MOVW R5, 12(R13)

	MOVW $·booted(SB), R1
ensureCas:
	LDREX (R1), R0
	CMP $0, R0
	BNE ensureWait
	MOVW $1, R2
	STREX R2, (R1), R0
	CMP $0, R0
	BNE ensureCas
	CALL ·boot(SB)

ensureWait:
	MOVW ·rtReady(SB), R0
	CMP $0, R0
	BNE ensureLeave
	MOVW $·rtReady(SB), R0
	MOVW $128, R1
	MOVW $0, R2
	MOVW $0, R3
	MOVW $0, R4
	MOVW $0, R5
	MOVW $240, R7
	SWI $0
	JMP ensureWait

ensureLeave:
	MOVW 0(R13), R14
	MOVW 4(R13), R7
	MOVW 8(R13), R4
	MOVW 12(R13), R5
	ADD $24, R13
ensureDone:
	RET

// boot starts one thread at rt0_go. runtime.clone is not a leaf: its
// prologue pushes LR, and 0(FP) lands at 8(SP) after that push, so the
// first argument is placed at 4(SP) before the call.
TEXT ·boot(SB),NOSPLIT|NOFRAME,$0
	SUB $48, R13
	MOVW R14, 32(R13)
	MOVW R4, 36(R13)
	MOVW R5, 40(R13)
	MOVW R7, 44(R13)

	MOVW $1, R0
	MOVB R0, runtime·islibrary(SB)

	MOVW $0, R0
	MOVW $0x800000, R1
	MOVW $3, R2
	MOVW $0x22, R3
	MOVW $-1, R4
	MOVW $0, R5
	MOVW $192, R7
	SWI $0
	MOVW $0xfffff001, R12
	CMP R12, R0
	BHI bootFail

	// rt0_go writes argv at the incoming SP. That word has to sit
	// inside the mapping, so the stack top is eight bytes below it.
	ADD $0x800000, R0, R1
	SUB $8, R1, R1
	MOVW $0x50F00, R12
	MOVW R12, 4(R13)
	MOVW R1, 8(R13)
	MOVW $0, R12
	MOVW R12, 12(R13)
	MOVW R12, 16(R13)
	MOVW $·rt0entry(SB), R2
	MOVW R2, 20(R13)
	CALL runtime·clone(SB)
	MOVW 24(R13), R0
	CMP $0, R0
	BLE bootFail

	MOVW 32(R13), R14
	MOVW 36(R13), R4
	MOVW 40(R13), R5
	MOVW 44(R13), R7
	ADD $48, R13
	RET

bootFail:
	MOVW $2, R0
	MOVW R0, ·rtReady(SB)
	MOVW $·rtReady(SB), R0
	MOVW $129, R1
	MOVW $1, R2
	MOVW $0, R3
	MOVW $0, R4
	MOVW $0, R5
	MOVW $240, R7
	SWI $0
	MOVW 32(R13), R14
	MOVW 36(R13), R4
	MOVW 40(R13), R5
	MOVW 44(R13), R7
	ADD $48, R13
	RET

// rt0entry is the cloned thread. argc is in R0 and argv is in R1.
TEXT ·rt0entry(SB),NOSPLIT|NOFRAME,$0
	MOVW $1, R0
	MOVW $·bootArgv(SB), R1
	JMP runtime·rt0_go(SB)

DATA ·bootArg0+0(SB)/8, $"clib\x00\x00\x00\x00"
GLOBL ·bootArg0(SB),RODATA,$8

DATA ·bootArgv+0(SB)/4, $·bootArg0(SB)
DATA ·bootArgv+4(SB)/4, $0
DATA ·bootArgv+8(SB)/4, $0
DATA ·bootArgv+12(SB)/4, $0
GLOBL ·bootArgv(SB),NOPTR,$16
