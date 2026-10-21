#include "textflag.h"

// func hevcCopy16SSE2(dst, src *byte)
TEXT ·hevcCopy16SSE2(SB), NOSPLIT, $0-16
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVUPS (SI), X0
	MOVUPS X0, (DI)
	RET
