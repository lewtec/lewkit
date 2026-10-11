//go:build windows && clibexport

#include "textflag.h"

// call is the Windows ARM64 entry every generated trampoline jumps to.
// R9 is the handler id. The integer arguments are R0 through R7 and the
// floats are F0 through F7, in independent pools. R9 is not an argument
// and not a veneer scratch. R18 is the TEB and stays untouched.
//
// The frame is 208 bytes and SP stays 16-byte aligned. The slot is the
// first 160 bytes. The pipe image is at 160, the write cursor at 168,
// the remaining count at 176, and the saved LR at 200.
TEXT ·call(SB),NOSPLIT|NOFRAME,$0
	SUB $208, RSP
	MOVD R30, 200(RSP)

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
	ADD $160, RSP, R3
	MOVD R3, 168(RSP)
	MOVD $8, R3
	MOVD R3, 176(RSP)
callWrite:
	MOVD ·pipeW(SB), R0
	MOVD 168(RSP), R1
	MOVD 176(RSP), R2
	ADD $184, RSP, R3
	SUB $16, RSP
	MOVD ZR, 0(RSP)
	MOVD ·_WriteFile(SB), R16
	CALL (R16)
	ADD $16, RSP
	CBZ R0, callFail
	MOVW 184(RSP), R4
	CBZ R4, callFail
	MOVD 168(RSP), R5
	ADD R4, R5, R5
	MOVD R5, 168(RSP)
	MOVD 176(RSP), R6
	SUB R4, R6, R6
	MOVD R6, 176(RSP)
	CBNZ R6, callWrite

callWait:
	MOVW 152(RSP), R0
	CBNZW R0, callDone
	MOVD $-1, R0
	MOVD $1, R1
	MOVD ·_WaitForSingleObject(SB), R16
	CALL (R16)
	JMP callWait

callDone:
	MOVD 136(RSP), R0
	FMOVD 144(RSP), F0
	MOVD 200(RSP), R30
	ADD $208, RSP
	RET

callFail:
	MOVD $0, R0
	FMOVD ZR, F0
	MOVD 200(RSP), R30
	ADD $208, RSP
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
	MOVD $-1, R0
	MOVD $1, R1
	MOVD ·_WaitForSingleObject(SB), R16
	CALL (R16)
	JMP ensureWait

ensureLeave:
	MOVD 0(RSP), R30
	ADD $16, RSP
ensureDone:
	RET

// boot opens the pipe, sets islibrary, and calls _rt0_arm64_lib.
// That entry clears g and CreateThread's the runtime.
TEXT ·boot(SB),NOSPLIT|NOFRAME,$0
	SUB $32, RSP
	MOVD R30, 24(RSP)

	MOVD $·pipeR(SB), R0
	MOVD $·pipeW(SB), R1
	MOVD ZR, R2
	MOVD ZR, R3
	MOVD ·_CreatePipe(SB), R16
	CALL (R16)
	CBZ R0, bootFail

	MOVW $1, R0
	MOVB R0, runtime·islibrary(SB)
	MOVD ZR, R0
	MOVD ZR, R1
	CALL _rt0_arm64_lib(SB)

	MOVD 24(RSP), R30
	ADD $32, RSP
	RET

bootFail:
	MOVW $2, R0
	MOVW R0, ·rtReady(SB)
	MOVD 24(RSP), R30
	ADD $32, RSP
	RET
