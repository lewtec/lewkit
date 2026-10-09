//go:build android && !cgo && arm64

#include "textflag.h"

// func callCOnG0(fn, a0, a1, a2, a3, a4, a5, a6, a7 uintptr) (r uintptr, f32 float32, f64 float64)
//
// The C callee runs on m.g0. A Java callback's goroutine stack is a few
// kilobytes, and cgocallback assumes the C frame is already on g0: it
// records that SP as g0.sched.sp. callC enters the syscall first, so a
// reentrant call can exitsyscall. g.sched offsets are Go 1.27's gobuf.
TEXT ·callCOnG0(SB),NOSPLIT,$0-96
	CBZ	g, callNosave

	// g.m is 48, m.g0 is 0. Already on the system stack: stay there.
	MOVD	48(g), R8
	MOVD	(R8), R3
	CMP	R3, g
	BEQ	callNosave

	// gosave smashes R0. Hold the C arguments in registers that survive it
	// and save_g. R19–R26 belong to the Go caller.
	MOVD	fn+0(FP), R16
	MOVD	a0+8(FP), R9
	MOVD	a1+16(FP), R1
	MOVD	a2+24(FP), R2
	MOVD	a3+32(FP), R4
	MOVD	a4+40(FP), R5
	MOVD	a5+48(FP), R6
	MOVD	a6+56(FP), R7
	MOVD	a7+64(FP), R10
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
	MOVD	8(R11), R13 // old g.stack.hi
	SUB	R12, R13, R13
	MOVD	RSP, R14
	SUB	$16, R14
	MOVD	R14, RSP
	MOVD	R11, (RSP)
	MOVD	R13, 8(RSP)

	MOVD	R9, R0
	MOVD	R4, R3
	MOVD	R5, R4
	MOVD	R6, R5
	MOVD	R7, R6
	MOVD	R10, R7
	BL	(R16)
	MOVD	R0, R9

	MOVD	(RSP), g
	BL	runtime·save_g(SB)
	MOVD	8(RSP), R6 // depth
	MOVD	8(g), R5 // stack.hi
	SUB	R6, R5, R5
	MOVD	R9, R0
	MOVD	R5, RSP

	MOVD	R0, r+72(FP)
	FMOVS	F0, f32+80(FP)
	FMOVD	F0, f64+88(FP)
	RET

callNosave:
	MOVD	fn+0(FP), R16
	MOVD	a0+8(FP), R0
	MOVD	a1+16(FP), R1
	MOVD	a2+24(FP), R2
	MOVD	a3+32(FP), R3
	MOVD	a4+40(FP), R4
	MOVD	a5+48(FP), R5
	MOVD	a6+56(FP), R6
	MOVD	a7+64(FP), R7
	BL	(R16)
	MOVD	R0, r+72(FP)
	FMOVS	F0, f32+80(FP)
	FMOVD	F0, f64+88(FP)
	RET
