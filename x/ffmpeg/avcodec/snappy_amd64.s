#include "textflag.h"

// func snappyCopySSE2(dst, src *byte, n int)
TEXT ·snappyCopySSE2(SB), NOSPLIT, $0-24
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ n+16(FP), CX
	TESTQ CX, CX
	JZ   done
loop:
	MOVUPS (SI), X0
	MOVUPS X0, (DI)
	ADDQ   $16, SI
	ADDQ   $16, DI
	SUBQ   $16, CX
	JNZ    loop
done:
	RET
