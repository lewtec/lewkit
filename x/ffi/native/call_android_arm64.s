//go:build android && !cgo && arm64

#include "textflag.h"

// func callC(fn, a0, a1, a2, a3, a4, a5, a6, a7 uintptr) (r uintptr, f32 float32, f64 float64)
//
// The C callee runs on the Go stack. JNI calls are shallow. A callback
// from Java into Go goes through cgocallback, which switches stacks.
TEXT ·callC(SB),NOSPLIT,$0-96
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
