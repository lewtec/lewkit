//go:build windows && clibexport

#include "textflag.h"

// call is the Win386 cdecl entry every generated trampoline jumps to.
// AX is the handler id. Arguments are on the stack, and an 8-byte argument
// is aligned to 8. EBX, ESI, EDI, and EBP belong to the caller. A float
// result is loaded into ST(0) from Kind. DllMain's stdcall pop is not
// this function's convention: the C caller pops.
//
// The frame is 192 bytes. The slot is the first 160. The pipe image is
// at 160. The saved registers start at 168. After the subtract, the C
// return address is at 192(SP) and the first argument is at 196(SP).
TEXT ·call(SB),NOSPLIT|NOFRAME,$0
	SUBL $192, SP
	MOVL BX, 168(SP)
	MOVL SI, 172(SP)
	MOVL DI, 176(SP)
	MOVL BP, 180(SP)

	MOVL AX, 0(SP)
	MOVL $0, 4(SP)
	MOVL $0, CX
callCopy:
	CMPL CX, $64
	JEQ callCopied
	MOVL 196(SP)(CX*1), AX
	MOVL AX, 8(SP)(CX*1)
	ADDL $4, CX
	JMP callCopy
callCopied:
	MOVL $0, 136(SP)
	MOVL $0, 140(SP)
	MOVL $0, 144(SP)
	MOVL $0, 148(SP)
	MOVL $0, 152(SP)
	MOVL $0, 156(SP)

	CALL ·ensure(SB)

	MOVL ·rtReady(SB), AX
	CMPL AX, $1
	JNE callFail

	MOVL SP, 160(SP)
	MOVL $0, 164(SP)
	LEAL 160(SP), SI
	MOVL $8, 188(SP)
callWrite:
	MOVL ·pipeW(SB), DI
	MOVL 188(SP), CX
	LEAL 184(SP), DX
	// The amount is in a register so the assembler does not treat it as a
	// frame adjustment. WriteFile is stdcall and pops these 20 bytes.
	MOVL $20, AX
	SUBL AX, SP
	MOVL DI, 0(SP)
	MOVL SI, 4(SP)
	MOVL CX, 8(SP)
	MOVL DX, 12(SP)
	MOVL $0, 16(SP)
	MOVL ·_WriteFile(SB), AX
	CALL AX
	TESTL AX, AX
	JE callFail
	MOVL 184(SP), AX
	CMPL AX, $0
	JE callFail
	ADDL AX, SI
	SUBL AX, 188(SP)
	JNE callWrite

callWait:
	MOVL 152(SP), AX
	CMPL AX, $0
	JNE callDone
	MOVL $8, AX
	SUBL AX, SP
	MOVL $-1, 0(SP)
	MOVL $1, 4(SP)
	MOVL ·_WaitForSingleObject(SB), AX
	CALL AX
	JMP callWait

callDone:
	MOVL 156(SP), CX
	CMPL CX, $1
	JE callF32
	CMPL CX, $2
	JE callF64
	MOVL 136(SP), AX
	MOVL 140(SP), DX
	JMP callRestore
callF32:
	FMOVF 144(SP), F0
	JMP callRestore
callF64:
	FMOVD 144(SP), F0
callRestore:
	MOVL 168(SP), BX
	MOVL 172(SP), SI
	MOVL 176(SP), DI
	MOVL 180(SP), BP
	ADDL $192, SP
	RET

callFail:
	XORL AX, AX
	XORL DX, DX
	MOVL $0, 144(SP)
	MOVL $0, 148(SP)
	FMOVD 144(SP), F0
	JMP callRestore

// ensure boots the runtime once. Losers, and the winner, wait until
// rtReady leaves 0.
TEXT ·ensure(SB),NOSPLIT|NOFRAME,$0
	MOVL ·rtReady(SB), AX
	CMPL AX, $0
	JNE ensureDone

	MOVL $0, AX
	MOVL $1, CX
	LEAL ·booted(SB), BX
	LOCK
	CMPXCHGL CX, 0(BX)
	JNE ensureWait
	CALL ·boot(SB)

ensureWait:
	MOVL ·rtReady(SB), AX
	CMPL AX, $0
	JNE ensureDone
	MOVL $8, AX
	SUBL AX, SP
	MOVL $-1, 0(SP)
	MOVL $1, 4(SP)
	MOVL ·_WaitForSingleObject(SB), AX
	CALL AX
	JMP ensureWait

ensureDone:
	RET

// boot opens the pipe, sets islibrary, and calls _rt0_386_lib.
// That entry CreateThread's the runtime. The two arguments are cdecl,
// so the caller pops them.
TEXT ·boot(SB),NOSPLIT|NOFRAME,$0
	// CreatePipe is stdcall and pops these 16 bytes.
	MOVL $16, AX
	SUBL AX, SP
	LEAL ·pipeR(SB), AX
	MOVL AX, 0(SP)
	LEAL ·pipeW(SB), AX
	MOVL AX, 4(SP)
	MOVL $0, 8(SP)
	MOVL $0, 12(SP)
	MOVL ·_CreatePipe(SB), AX
	CALL AX
	TESTL AX, AX
	JE bootFail

	MOVL $1, AX
	MOVB AX, runtime·islibrary(SB)
	// _rt0_386_lib is cdecl. A register adjustment keeps PUSH/POP balanced.
	MOVL $8, AX
	SUBL AX, SP
	MOVL $0, 0(SP)
	MOVL $0, 4(SP)
	CALL _rt0_386_lib(SB)
	MOVL $8, AX
	ADDL AX, SP
	RET

bootFail:
	MOVL $2, ·rtReady(SB)
	RET
