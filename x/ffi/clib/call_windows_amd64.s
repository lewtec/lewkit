//go:build windows && clibexport

#include "textflag.h"

// call is the Win64 entry every generated trampoline jumps to.
// AX is the handler id. The first four arguments share one slot each:
// RCX, RDX, R8, R9 or X0 through X3. Later arguments are 8-byte stack
// slots starting at 40(SP) on entry, and those bits are copied into both
// the integer word and the xmm word. DI, SI, BX, BP, and R12-R15 belong
// to the caller.
//
// The frame is 216 bytes. SP is 8 mod 16 on entry and 0 mod 16 after the
// subtract, which is what a Windows call requires. The slot is the first
// 160 bytes. The pipe image is at 160, the write cursor at 168, and the
// remaining count at 176.
TEXT ·call(SB),NOSPLIT|NOFRAME,$0
	SUBQ $216, SP

	MOVL AX, 0(SP)
	MOVL $0, 4(SP)
	MOVQ CX, 8(SP)
	MOVQ DX, 16(SP)
	MOVQ R8, 24(SP)
	MOVQ R9, 32(SP)
	MOVQ 256(SP), R10
	MOVQ R10, 40(SP)
	MOVQ R10, 104(SP)
	MOVQ 264(SP), R10
	MOVQ R10, 48(SP)
	MOVQ R10, 112(SP)
	MOVQ 272(SP), R10
	MOVQ R10, 56(SP)
	MOVQ R10, 120(SP)
	MOVQ 280(SP), R10
	MOVQ R10, 64(SP)
	MOVQ R10, 128(SP)
	MOVQ X0, 72(SP)
	MOVQ X1, 80(SP)
	MOVQ X2, 88(SP)
	MOVQ X3, 96(SP)
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
	MOVQ ·pipeW(SB), CX
	MOVQ 168(SP), DX
	MOVQ 176(SP), R8
	LEAQ 184(SP), R9
	SUBQ $48, SP
	MOVQ $0, 32(SP)
	MOVQ ·_WriteFile(SB), AX
	CALL AX
	ADDQ $48, SP
	TESTQ AX, AX
	JE callFail
	MOVL 184(SP), AX
	CMPL AX, $0
	JEQ callFail
	MOVL AX, R10
	ADDQ R10, 168(SP)
	SUBQ R10, 176(SP)
	JNE callWrite

callWait:
	MOVL 152(SP), AX
	CMPL AX, $0
	JNE callDone
	SUBQ $32, SP
	MOVQ $-1, CX
	MOVQ $1, DX
	MOVQ ·_WaitForSingleObject(SB), AX
	CALL AX
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
// rtReady leaves 0. Waiting is a 1ms sleep so a one-CPU host can schedule
// the new thread.
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
	MOVQ $-1, CX
	MOVQ $1, DX
	MOVQ ·_WaitForSingleObject(SB), AX
	CALL AX
	ADDQ $40, SP
	JMP ensureWait

ensureDone:
	RET

// boot opens the pipe, sets islibrary, and calls _rt0_amd64_lib.
// That entry installs a dummy TLS and CreateThread's the runtime.
// DI and SI are callee-saved on Win64, and _rt0 restores the values it
// was entered with, so the originals are put back before returning.
TEXT ·boot(SB),NOSPLIT|NOFRAME,$0
	SUBQ $56, SP
	MOVQ DI, 32(SP)
	MOVQ SI, 40(SP)

	MOVQ $·pipeR(SB), CX
	MOVQ $·pipeW(SB), DX
	XORQ R8, R8
	XORQ R9, R9
	MOVQ ·_CreatePipe(SB), AX
	CALL AX
	TESTQ AX, AX
	JE bootFail

	MOVB $1, runtime·islibrary(SB)
	XORQ DI, DI
	XORQ SI, SI
	CALL _rt0_amd64_lib(SB)

	MOVQ 32(SP), DI
	MOVQ 40(SP), SI
	ADDQ $56, SP
	RET

bootFail:
	MOVL $2, ·rtReady(SB)
	MOVQ 32(SP), DI
	MOVQ 40(SP), SI
	ADDQ $56, SP
	RET
