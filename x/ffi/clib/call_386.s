//go:build (linux || android) && clibexport

#include "textflag.h"

// call is the cdecl entry every generated trampoline jumps to.
// AX is the handler id. Integer and float arguments are on the stack,
// four-byte aligned, starting at 4(SP) on entry. EBX, ESI, EDI, and EBP
// belong to the C caller. A float result is loaded into ST(0) from Kind.
//
// The frame is 192 bytes. The slot is the first 160. The pipe pointer is
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
	MOVL $8, DX
callWrite:
	MOVL $4, AX
	MOVL ·pipeW(SB), BX
	MOVL SI, CX
	INT $0x80
	CMPL AX, $0
	JLE callWriteErr
	ADDL AX, SI
	SUBL AX, DX
	JNE callWrite
	JMP callWait
callWriteErr:
	CMPL AX, $-4
	JE callWrite
	JMP callFail

callWait:
	MOVL 152(SP), AX
	CMPL AX, $0
	JNE callDone
	LEAL 152(SP), BX
	MOVL $128, CX
	MOVL $0, DX
	MOVL $0, SI
	MOVL $0, DI
	MOVL $0, BP
	MOVL $240, AX
	INT $0x80
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
	SUBL $24, SP
	MOVL BP, 20(SP)
	LEAL ·rtReady(SB), BX
	MOVL $128, CX
	MOVL $0, DX
	MOVL $0, SI
	MOVL $0, DI
	MOVL $0, BP
	MOVL $240, AX
	INT $0x80
	MOVL 20(SP), BP
	ADDL $24, SP
	JMP ensureWait

ensureDone:
	RET

// boot starts one thread at rt0_go. islibrary is set first so runtime.main
// returns after init instead of calling main and exiting the process.
// mmap2 takes the offset in pages. mp and gp stay nil.
TEXT ·boot(SB),NOSPLIT|NOFRAME,$0
	SUBL $32, SP
	MOVL BP, 28(SP)

	MOVL $1, AX
	MOVB AX, runtime·islibrary(SB)

	MOVL $192, AX
	MOVL $0, BX
	MOVL $0x800000, CX
	MOVL $3, DX
	MOVL $0x22, SI
	MOVL $-1, DI
	MOVL $0, BP
	INT $0x80
	CMPL AX, $0xfffff001
	JA bootFail

	ADDL $0x800000, AX
	MOVL $0x50F00, 0(SP)
	MOVL AX, 4(SP)
	MOVL $0, 8(SP)
	MOVL $0, 12(SP)
	LEAL ·rt0entry(SB), AX
	MOVL AX, 16(SP)
	CALL runtime·clone(SB)
	MOVL 20(SP), AX
	CMPL AX, $0
	JLE bootFail

	MOVL 28(SP), BP
	ADDL $32, SP
	RET

bootFail:
	MOVL 28(SP), BP
	ADDL $32, SP
	MOVL $2, ·rtReady(SB)
	SUBL $24, SP
	MOVL BP, 20(SP)
	LEAL ·rtReady(SB), BX
	MOVL $129, CX
	MOVL $1, DX
	MOVL $0, SI
	MOVL $0, DI
	MOVL $0, BP
	MOVL $240, AX
	INT $0x80
	MOVL 20(SP), BP
	ADDL $24, SP
	RET

// rt0entry is the cloned thread. clone called it, so 0(SP) is that return
// address. rt0_go reads argc and argv from the stack and does not return.
TEXT ·rt0entry(SB),NOSPLIT|NOFRAME,$0
	MOVL $1, 0(SP)
	LEAL ·bootArgv(SB), AX
	MOVL AX, 4(SP)
	JMP runtime·rt0_go(SB)

DATA ·bootArg0+0(SB)/8, $"clib\x00\x00\x00\x00"
GLOBL ·bootArg0(SB),RODATA,$8

DATA ·bootArgv+0(SB)/4, $·bootArg0(SB)
DATA ·bootArgv+4(SB)/4, $0
DATA ·bootArgv+8(SB)/4, $0
DATA ·bootArgv+12(SB)/4, $0
GLOBL ·bootArgv(SB),NOPTR,$16
