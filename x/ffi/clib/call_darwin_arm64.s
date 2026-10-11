//go:build darwin && !ios && clibexport

#include "textflag.h"

// call is the AAPCS64 entry every generated trampoline jumps to.
// R9 is the handler id. The integer arguments are R0 through R7 and the
// floats are F0 through F7. The slot stays on this stack. A Go thread
// reads the pointer from the pipe. This thread sleeps until Done is stored.
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
	MOVW ·pipeW(SB), R0
	MOVD 168(RSP), R1
	MOVD 176(RSP), R2
	CALL libc_write(SB)
	CMP $0, R0
	BLE callWriteErr
	MOVD 168(RSP), R1
	ADD R0, R1, R1
	MOVD R1, 168(RSP)
	MOVD 176(RSP), R2
	SUB R0, R2, R2
	MOVD R2, 176(RSP)
	CBNZ R2, callWrite
	JMP callWait
callWriteErr:
	CALL libc_error(SB)
	MOVW (R0), R0
	CMP $4, R0
	BEQ callWrite
	JMP callFail

callWait:
	MOVW 152(RSP), R0
	CBNZW R0, callDone
	MOVD $1000, R0
	CALL libc_usleep(SB)
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
	MOVD $1000, R0
	CALL libc_usleep(SB)
	JMP ensureWait

ensureLeave:
	MOVD 0(RSP), R30
	ADD $16, RSP
ensureDone:
	RET

// boot opens the pipe, sets islibrary, and calls _rt0_arm64_lib.
// That entry clears g and pthread_create's the runtime. The two fds are ints.
TEXT ·boot(SB),NOSPLIT|NOFRAME,$0
	SUB $32, RSP
	MOVD R30, 24(RSP)

	ADD $8, RSP, R0
	CALL libc_pipe(SB)
	CBNZ R0, bootFail
	MOVW 8(RSP), R0
	MOVD R0, ·pipeR(SB)
	MOVW 12(RSP), R0
	MOVD R0, ·pipeW(SB)

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
