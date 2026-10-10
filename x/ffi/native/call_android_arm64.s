//go:build android && !cgo && arm64

#include "textflag.h"

// func callCOnG0(fn, argv, argc uintptr) (r uintptr, f32 float32, f64 float64)
//
// argv points at callWords uintptrs. The first eight fill R0–R7. Words
// past that are stored at the C stack, eight bytes each, before BL.
// The C callee runs on m.g0. A Java callback's goroutine stack is a few
// kilobytes, and cgocallback assumes the C frame is already on g0: it
// records that SP as g0.sched.sp. callC enters the syscall first, so a
// reentrant call can exitsyscall. g.sched offsets are Go 1.27's gobuf.
//
// R19 and R20 are saved in the frame. callNosave parks the Go SP in R20
// across BL; AAPCS64 callees preserve it. The g0 path keeps the old g
// and the stack depth above the outgoing argument area.
TEXT ·callCOnG0(SB),NOSPLIT,$16-48
	// Negative SP offsets are the local frame. 0(RSP) is the saved LR.
	MOVD	R19, r19-8(SP)
	MOVD	R20, r20-16(SP)

	CBZ	g, callNosave

	// g.m is 48, m.g0 is 0. Already on the system stack: stay there.
	MOVD	48(g), R8
	MOVD	(R8), R3
	CMP	R3, g
	BEQ	callNosave

	// Load the call before switching SP. FP is the goroutine frame.
	MOVD	fn+0(FP), R16
	MOVD	argv+8(FP), R15
	MOVD	argc+16(FP), R14
	MOVD	g, R11
	MOVD	RSP, R12

	// Inlined gosave_systemstack_switch. The saved PC stops traceback
	// at the bottom of this goroutine stack while C runs on g0.
	MOVD	$runtime·systemstack_switch(SB), R0
	ADD	$8, R0
	MOVD	R0, 64(g) // sched.pc
	MOVD	R12, R0
	MOVD	R0, 56(g) // sched.sp
	MOVD	R29, 96(g) // sched.bp
	MOVD	ZR, 88(g) // sched.lr
	MOVD	80(g), R0 // sched.ctxt
	CBZ	R0, callSaved
	BL	runtime·abort(SB)

callSaved:
	MOVD	R3, g
	BL	runtime·save_g(SB)
	MOVD	56(g), R0 // g0.sched.sp
	MOVD	R0, RSP
	MOVD	96(g), R29 // g0.sched.bp

	// Depth, not a raw SP. A callback can move the goroutine stack.
	// 64 bytes of stacked arguments, then the saved g and depth.
	// 80 keeps SP 16-byte aligned for BL.
	MOVD	8(R11), R13 // old g.stack.hi
	SUB	R12, R13, R13
	SUB	$80, RSP
	MOVD	R11, 64(RSP)
	MOVD	R13, 72(RSP)

	CMP	$8, R14
	BLS	g0regs
	MOVD	64(R15), R8
	MOVD	R8, (RSP)
	CMP	$9, R14
	BLS	g0regs
	MOVD	72(R15), R8
	MOVD	R8, 8(RSP)
	CMP	$10, R14
	BLS	g0regs
	MOVD	80(R15), R8
	MOVD	R8, 16(RSP)
	CMP	$11, R14
	BLS	g0regs
	MOVD	88(R15), R8
	MOVD	R8, 24(RSP)
	CMP	$12, R14
	BLS	g0regs
	MOVD	96(R15), R8
	MOVD	R8, 32(RSP)
	CMP	$13, R14
	BLS	g0regs
	MOVD	104(R15), R8
	MOVD	R8, 40(RSP)
	CMP	$14, R14
	BLS	g0regs
	MOVD	112(R15), R8
	MOVD	R8, 48(RSP)
	CMP	$15, R14
	BLS	g0regs
	MOVD	120(R15), R8
	MOVD	R8, 56(RSP)

g0regs:
	MOVD	(R15), R0
	MOVD	8(R15), R1
	MOVD	16(R15), R2
	MOVD	24(R15), R3
	MOVD	32(R15), R4
	MOVD	40(R15), R5
	MOVD	48(R15), R6
	MOVD	56(R15), R7
	BL	(R16)
	MOVD	R0, R9

	MOVD	64(RSP), g
	BL	runtime·save_g(SB)
	MOVD	72(RSP), R6 // depth
	MOVD	8(g), R5 // stack.hi
	SUB	R6, R5, R5
	MOVD	R9, R0
	MOVD	R5, RSP

	MOVD	r19-8(SP), R19
	MOVD	r20-16(SP), R20
	MOVD	R0, r+24(FP)
	FMOVS	F0, f32+32(FP)
	FMOVD	F0, f64+40(FP)
	RET

callNosave:
	MOVD	fn+0(FP), R16
	MOVD	argv+8(FP), R15
	MOVD	argc+16(FP), R14
	MOVD	RSP, R20
	// 80, not 64: the saved frame pointer sits at SP-8, and a 16-word
	// argument list would otherwise land on it. 80 keeps that byte free
	// and keeps SP 16-byte aligned.
	SUB	$80, RSP
	CMP	$8, R14
	BLS	nsRegs
	MOVD	64(R15), R8
	MOVD	R8, (RSP)
	CMP	$9, R14
	BLS	nsRegs
	MOVD	72(R15), R8
	MOVD	R8, 8(RSP)
	CMP	$10, R14
	BLS	nsRegs
	MOVD	80(R15), R8
	MOVD	R8, 16(RSP)
	CMP	$11, R14
	BLS	nsRegs
	MOVD	88(R15), R8
	MOVD	R8, 24(RSP)
	CMP	$12, R14
	BLS	nsRegs
	MOVD	96(R15), R8
	MOVD	R8, 32(RSP)
	CMP	$13, R14
	BLS	nsRegs
	MOVD	104(R15), R8
	MOVD	R8, 40(RSP)
	CMP	$14, R14
	BLS	nsRegs
	MOVD	112(R15), R8
	MOVD	R8, 48(RSP)
	CMP	$15, R14
	BLS	nsRegs
	MOVD	120(R15), R8
	MOVD	R8, 56(RSP)

nsRegs:
	MOVD	(R15), R0
	MOVD	8(R15), R1
	MOVD	16(R15), R2
	MOVD	24(R15), R3
	MOVD	32(R15), R4
	MOVD	40(R15), R5
	MOVD	48(R15), R6
	MOVD	56(R15), R7
	BL	(R16)
	MOVD	R20, RSP
	MOVD	r19-8(SP), R19
	MOVD	r20-16(SP), R20
	MOVD	R0, r+24(FP)
	FMOVS	F0, f32+32(FP)
	FMOVD	F0, f64+40(FP)
	RET
