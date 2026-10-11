//go:build darwin && !ios && clibexport

#include "textflag.h"

// call is the SysV entry every generated trampoline jumps to.
// AX is the handler id. The integer arguments are DI, SI, DX, CX, R8, R9
// and the floats are X0 through X7. The slot stays on this stack. A Go
// thread reads the pointer from the pipe. This thread sleeps until Done
// is stored.
//
// The frame is 216 bytes. SP is 8 mod 16 on entry and 0 mod 16 after the
// subtract. The slot is the first 160 bytes. The pipe image is at 160,
// the write cursor at 168, and the remaining count at 176.
TEXT ·call(SB),NOSPLIT|NOFRAME,$0
	SUBQ $216, SP

	MOVL AX, 0(SP)
	MOVL $0, 4(SP)
	MOVQ DI, 8(SP)
	MOVQ SI, 16(SP)
	MOVQ DX, 24(SP)
	MOVQ CX, 32(SP)
	MOVQ R8, 40(SP)
	MOVQ R9, 48(SP)
	MOVQ $0, 56(SP)
	MOVQ $0, 64(SP)
	MOVQ X0, 72(SP)
	MOVQ X1, 80(SP)
	MOVQ X2, 88(SP)
	MOVQ X3, 96(SP)
	MOVQ X4, 104(SP)
	MOVQ X5, 112(SP)
	MOVQ X6, 120(SP)
	MOVQ X7, 128(SP)
	MOVQ $0, 136(SP)
	MOVQ $0, 144(SP)
	MOVL $0, 152(SP)
	MOVL $0, 156(SP)

	CALL ·ensure(SB)

	MOVL ·rtReady(SB), AX
	CMPL AX, $1
	JNE callFail

	MOVQ SP, 160(SP)
	LEAQ 160(SP), R10
	MOVQ R10, 168(SP)
	MOVQ $8, 176(SP)
callWrite:
	MOVL ·pipeW(SB), DI
	MOVQ 168(SP), SI
	MOVQ 176(SP), DX
	SUBQ $32, SP
	CALL libc_write(SB)
	ADDQ $32, SP
	CMPQ AX, $0
	JLE callWriteErr
	ADDQ AX, 168(SP)
	SUBQ AX, 176(SP)
	JNE callWrite
	JMP callWait
callWriteErr:
	SUBQ $32, SP
	CALL libc_error(SB)
	ADDQ $32, SP
	MOVL (AX), CX
	CMPL CX, $4
	JE callWrite
	JMP callFail

callWait:
	MOVL 152(SP), AX
	CMPL AX, $0
	JNE callDone
	SUBQ $32, SP
	MOVQ $1000, DI
	CALL libc_usleep(SB)
	ADDQ $32, SP
	JMP callWait

callDone:
	MOVQ 136(SP), AX
	MOVQ 144(SP), X0
	ADDQ $216, SP
	RET

callFail:
	XORQ AX, AX
	XORPS X0, X0
	ADDQ $216, SP
	RET

// ensure boots the runtime once. Losers, and the winner, wait until
// rtReady leaves 0.
TEXT ·ensure(SB),NOSPLIT|NOFRAME,$0
	MOVL ·rtReady(SB), AX
	CMPL AX, $0
	JNE ensureDone

	MOVL $0, AX
	MOVL $1, CX
	LEAQ ·booted(SB), R11
	LOCK
	CMPXCHGL CX, 0(R11)
	JNE ensureWait
	CALL ·boot(SB)

ensureWait:
	MOVL ·rtReady(SB), AX
	CMPL AX, $0
	JNE ensureDone
	SUBQ $40, SP
	MOVQ $1000, DI
	CALL libc_usleep(SB)
	ADDQ $40, SP
	JMP ensureWait

ensureDone:
	RET

// boot opens the pipe, sets islibrary, and calls _rt0_amd64_lib.
// That entry starts the runtime on a pthread. The two fds are ints.
TEXT ·boot(SB),NOSPLIT|NOFRAME,$0
	SUBQ $40, SP
	LEAQ 8(SP), DI
	CALL libc_pipe(SB)
	TESTL AX, AX
	JNE bootFail
	MOVL 8(SP), AX
	MOVQ AX, ·pipeR(SB)
	MOVL 12(SP), AX
	MOVQ AX, ·pipeW(SB)

	MOVB $1, runtime·islibrary(SB)
	XORQ DI, DI
	XORQ SI, SI
	CALL _rt0_amd64_lib(SB)
	ADDQ $40, SP
	RET

bootFail:
	MOVL $2, ·rtReady(SB)
	ADDQ $40, SP
	RET
