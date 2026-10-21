//go:build amd64

#include "textflag.h"

// 16384 replicated
DATA gsmRound<>(SB)/4, $16384
DATA gsmRound<>+4(SB)/4, $16384
DATA gsmRound<>+8(SB)/4, $16384
DATA gsmRound<>+12(SB)/4, $16384
GLOBL gsmRound<>(SB), RODATA, $16

// func gsmMult40SSE2(dst *int16, src *int16, gain int16)
TEXT ·gsmMult40SSE2(SB), NOSPLIT, $0-18
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVWQZX gain+16(FP), AX
	MOVL AX, X3
	PSHUFD $0, X3, X3
	MOVUPS gsmRound<>(SB), X4
	MOVQ $5, CX
loop:
	MOVUPS (SI), X0
	MOVO   X0, X1
	PXOR   X2, X2
	PCMPGTW X0, X2
	PUNPCKLWL X2, X0
	PUNPCKHWL X2, X1
	PMULLD X3, X0
	PMULLD X3, X1
	PADDD  X4, X0
	PADDD  X4, X1
	PSRAL  $15, X0
	PSRAL  $15, X1
	PACKSSLW X1, X0
	MOVUPS X0, (DI)
	ADDQ   $16, SI
	ADDQ   $16, DI
	DECQ   CX
	JNZ    loop
	RET
