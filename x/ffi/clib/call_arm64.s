//go:build (linux || android) && clibexport

#include "textflag.h"

// call is the AAPCS64 entry every generated trampoline jumps to.
// R9 is the handler id. The integer arguments are R0 through R7 and the
// floats are F0 through F7. R9 is not an argument and not a veneer
// scratch, so the branch into this function leaves it intact.
//
// The frame is 176 bytes so it stays 16-byte aligned: the 160-byte slot,
// the 8-byte pipe pointer, and the saved LR. SP is 16-byte aligned on
// entry because the C caller aligned it and the trampoline branches.
TEXT ·call(SB),NOSPLIT|NOFRAME,$0
	SUB $176, RSP
	MOVD R30, 168(RSP)

	MOVW R9, 0(RSP)
	MOVW $0, 4(RSP)
	MOVD R0, 8(RSP)
	MOVD R1, 16(RSP)
	MOVD R2, 24(RSP)
	MOVD R3, 32(RSP)
	MOVD R4, 40(RSP)
	MOVD R5, 48(RSP)
	MOVD R6, 56(RSP)
	MOVD R7, 64(RSP)
	FMOVD F0, 72(RSP)
	FMOVD F1, 80(RSP)
	FMOVD F2, 88(RSP)
	FMOVD F3, 96(RSP)
	FMOVD F4, 104(RSP)
	FMOVD F5, 112(RSP)
	FMOVD F6, 120(RSP)
	FMOVD F7, 128(RSP)
	MOVD $0, 136(RSP)
	MOVD $0, 144(RSP)
	MOVW $0, 152(RSP)
	MOVW $0, 156(RSP)

	CALL ·ensure(SB)

	MOVW ·rtReady(SB), R0
	CMP $1, R0
	BNE callFail

	MOVD RSP, R3
	MOVD R3, 160(RSP)
	ADD $160, RSP, R1
	MOVD $8, R2
callWrite:
	MOVW ·pipeW(SB), R0
	MOVD $64, R8
	SVC
	CMP $0, R0
	BLE callWriteErr
	ADD R0, R1, R1
	SUB R0, R2, R2
	CBNZ R2, callWrite
	JMP callWait
callWriteErr:
	CMP $-4, R0
	BEQ callWrite
	JMP callFail

callWait:
	MOVW 152(RSP), R0
	CBNZW R0, callDone
	ADD $152, RSP, R0
	MOVW $128, R1
	MOVW $0, R2
	MOVD $0, R3
	MOVD $0, R4
	MOVD $0, R5
	MOVD $98, R8
	SVC
	JMP callWait

callDone:
	MOVD 136(RSP), R0
	FMOVD 144(RSP), F0
	MOVD 168(RSP), R30
	ADD $176, RSP
	RET

callFail:
	MOVD $0, R0
	FMOVD ZR, F0
	MOVD 168(RSP), R30
	ADD $176, RSP
	RET

// ensure boots the runtime once. Losers, and the winner, wait until
// rtReady leaves 0.
TEXT ·ensure(SB),NOSPLIT|NOFRAME,$0
	MOVW ·rtReady(SB), R0
	CBNZW R0, ensureDone

	SUB $16, RSP
	MOVD R30, 0(RSP)

	MOVD $·booted(SB), R8
ensureCas:
	LDAXRW (R8), R0
	CBNZW R0, ensureWait
	MOVW $1, R1
	STLXRW R1, (R8), R2
	CBNZ R2, ensureCas
	CALL ·boot(SB)

ensureWait:
	MOVW ·rtReady(SB), R0
	CBNZW R0, ensureLeave
	MOVD $·rtReady(SB), R0
	MOVW $128, R1
	MOVW $0, R2
	MOVD $0, R3
	MOVD $0, R4
	MOVD $0, R5
	MOVD $98, R8
	SVC
	JMP ensureWait

ensureLeave:
	MOVD 0(RSP), R30
	ADD $16, RSP
ensureDone:
	RET

// boot starts one thread at rt0_go. islibrary is set first so runtime.main
// returns after init instead of calling main and exiting the process.
// mp and gp stay nil so clone does not switch this thread's TLS.
// runtime.clone is NOFRAME, and on arm64 0(FP) is 8(RSP).
TEXT ·boot(SB),NOSPLIT|NOFRAME,$0
	SUB $64, RSP
	MOVD R30, 56(RSP)

	MOVW $1, R0
	MOVB R0, runtime·islibrary(SB)

	MOVD $0, R0
	MOVD $0x800000, R1
	MOVD $3, R2
	MOVD $0x22, R3
	MOVD $-1, R4
	MOVD $0, R5
	MOVD $222, R8
	SVC
	CMN $4095, R0
	BCS bootFail

	ADD $0x800000, R0, R1
	MOVD $0x50F00, R3
	MOVW R3, 8(RSP)
	MOVD R1, 16(RSP)
	MOVD $0, 24(RSP)
	MOVD $0, 32(RSP)
	MOVD $·rt0entry(SB), R2
	MOVD R2, 40(RSP)
	CALL runtime·clone(SB)
	MOVW 48(RSP), R0
	CMP $0, R0
	BLE bootFail

	MOVD 56(RSP), R30
	ADD $64, RSP
	RET

bootFail:
	MOVW $2, R0
	MOVW R0, ·rtReady(SB)
	MOVD $·rtReady(SB), R0
	MOVW $129, R1
	MOVW $1, R2
	MOVD $0, R3
	MOVD $0, R4
	MOVD $0, R5
	MOVD $98, R8
	SVC
	MOVD 56(RSP), R30
	ADD $64, RSP
	RET

// rt0entry is the cloned thread. argc is 1 and argv ends at AT_NULL so
// sysargs falls back to /proc/self/auxv.
TEXT ·rt0entry(SB),NOSPLIT|NOFRAME,$0
	MOVD $1, R0
	MOVD $·bootArgv(SB), R1
	JMP runtime·rt0_go(SB)

DATA ·bootArg0+0(SB)/8, $"clib\x00\x00\x00\x00"
GLOBL ·bootArg0(SB),RODATA,$8

DATA ·bootArgv+0(SB)/8, $·bootArg0(SB)
DATA ·bootArgv+8(SB)/8, $0
DATA ·bootArgv+16(SB)/8, $0
DATA ·bootArgv+24(SB)/8, $0
GLOBL ·bootArgv(SB),NOPTR,$32
