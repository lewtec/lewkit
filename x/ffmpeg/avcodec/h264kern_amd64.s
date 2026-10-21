//go:build amd64

#include "textflag.h"

// func addClip8SSE2(dst *byte, residual *int16, n int)
// n is a multiple of 8. Each residual is added to one byte and saturated.
TEXT ·addClip8SSE2(SB), NOSPLIT, $0-24
	MOVQ dst+0(FP), DI
	MOVQ residual+8(FP), SI
	MOVQ n+16(FP), CX
	TESTQ CX, CX
	JZ   done
loop:
	MOVQ   (DI), X0
	PXOR   X1, X1
	PUNPCKLBW X1, X0
	MOVUPS (SI), X2
	PADDW  X2, X0
	PACKUSWB X0, X0
	MOVQ   X0, (DI)
	ADDQ   $16, SI
	ADDQ   $8, DI
	SUBQ   $8, CX
	JNZ    loop
done:
	RET
