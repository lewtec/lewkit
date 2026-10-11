//go:build linux && amd64 && clibexport

#include "textflag.h"

// call is the SysV entry every generated trampoline jumps to.
// AX is the handler id. The integer arguments are DI, SI, DX, CX, R8, R9
// and the floats are X0 through X7. This thread never joins the Go
// scheduler: its FS register belongs to libc. The slot stays on this
// stack. A cloned Go thread reads the pointer from the pipe and wakes
// the futex at Done.
//
// The frame is 152 bytes so the 144-byte slot is followed by the 8-byte
// pointer written to the pipe, and SP is 0 mod 16 before CALL.
TEXT ·call(SB),NOSPLIT|NOFRAME,$0
	SUBQ $152, SP

	MOVL AX, 0(SP)
	MOVL $0, 4(SP)
	MOVQ DI, 8(SP)
	MOVQ SI, 16(SP)
	MOVQ DX, 24(SP)
	MOVQ CX, 32(SP)
	MOVQ R8, 40(SP)
	MOVQ R9, 48(SP)
	MOVQ X0, 56(SP)
	MOVQ X1, 64(SP)
	MOVQ X2, 72(SP)
	MOVQ X3, 80(SP)
	MOVQ X4, 88(SP)
	MOVQ X5, 96(SP)
	MOVQ X6, 104(SP)
	MOVQ X7, 112(SP)
	MOVQ $0, 120(SP)
	MOVQ $0, 128(SP)
	MOVL $0, 136(SP)
	MOVL $0, 140(SP)

	CALL ·ensure(SB)

	MOVL ·rtReady(SB), AX
	CMPL AX, $1
	JNE callFail

	MOVQ SP, 144(SP)
	LEAQ 144(SP), SI
	MOVQ $8, DX
callWrite:
	MOVQ $1, AX
	MOVL ·pipeW(SB), DI
	SYSCALL
	CMPQ AX, $0
	JLE callWriteErr
	ADDQ AX, SI
	SUBQ AX, DX
	JNE callWrite
	JMP callWait
callWriteErr:
	// EINTR is -4. Any other failure returns zero instead of parking.
	CMPQ AX, $-4
	JE callWrite
	JMP callFail

callWait:
	MOVL 136(SP), AX
	CMPL AX, $0
	JNE callDone
	LEAQ 136(SP), DI
	MOVL $128, SI
	MOVL $0, DX
	MOVQ $0, R10
	MOVQ $0, R8
	MOVQ $0, R9
	MOVQ $202, AX
	SYSCALL
	JMP callWait

callDone:
	MOVQ 120(SP), AX
	MOVQ 128(SP), X0
	ADDQ $152, SP
	RET

callFail:
	XORQ AX, AX
	XORPS X0, X0
	ADDQ $152, SP
	RET

// ensure boots the runtime once. Losers, and the winner, wait until
// rtReady leaves 0. A spin would deadlock a one-CPU host: the new thread
// could not be scheduled while this thread burned the only core.
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
	LEAQ ·rtReady(SB), DI
	MOVL $128, SI
	MOVL $0, DX
	MOVQ $0, R10
	MOVQ $0, R8
	MOVQ $0, R9
	MOVQ $202, AX
	SYSCALL
	JMP ensureWait

ensureDone:
	RET

// boot starts one thread at rt0_go. islibrary is set first so runtime.main
// returns after init instead of calling main and exiting the process.
// The stack is anonymous memory. mp and gp stay nil so clone does not
// switch this thread's TLS. A failure stores rtReady 2 and wakes waiters.
TEXT ·boot(SB),NOSPLIT|NOFRAME,$0
	SUBQ $72, SP
	MOVQ R12, 56(SP)
	MOVQ R13, 64(SP)

	MOVB $1, runtime·islibrary(SB)

	MOVQ $0, DI
	MOVQ $0x800000, SI
	MOVQ $3, DX
	MOVQ $0x22, R10
	MOVQ $-1, R8
	MOVQ $0, R9
	MOVQ $9, AX
	SYSCALL
	CMPQ AX, $0xfffffffffffff001
	JAE bootFail

	ADDQ $0x800000, AX
	MOVL $0x50F00, 0(SP)
	MOVQ AX, 8(SP)
	MOVQ $0, 16(SP)
	MOVQ $0, 24(SP)
	LEAQ ·rt0entry(SB), AX
	MOVQ AX, 32(SP)
	CALL runtime·clone(SB)
	CMPQ AX, $0
	JLE bootFail

	MOVQ 56(SP), R12
	MOVQ 64(SP), R13
	ADDQ $72, SP
	RET

bootFail:
	MOVQ 56(SP), R12
	MOVQ 64(SP), R13
	ADDQ $72, SP
	MOVL $2, ·rtReady(SB)
	LEAQ ·rtReady(SB), DI
	MOVL $129, SI
	MOVL $1, DX
	MOVQ $0, R10
	MOVQ $0, R8
	MOVQ $0, R9
	MOVQ $202, AX
	SYSCALL
	RET

// rt0entry is the cloned thread. argc is 1 and argv ends at AT_NULL so
// sysargs falls back to /proc/self/auxv.
TEXT ·rt0entry(SB),NOSPLIT|NOFRAME,$0
	MOVQ $1, DI
	LEAQ ·bootArgv(SB), SI
	JMP runtime·rt0_go(SB)

DATA ·bootArg0+0(SB)/8, $"clib\x00\x00\x00\x00"
GLOBL ·bootArg0(SB),RODATA,$8

DATA ·bootArgv+0(SB)/8, $·bootArg0(SB)
DATA ·bootArgv+8(SB)/8, $0
DATA ·bootArgv+16(SB)/8, $0
DATA ·bootArgv+24(SB)/8, $0
GLOBL ·bootArgv(SB),NOPTR,$32
