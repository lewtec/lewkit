//go:build tamago && amd64

#include "textflag.h"

TEXT ·outb(SB),NOSPLIT,$0-3
	MOVW	port+0(FP), DX
	MOVB	val+2(FP), AX
	OUTB
	RET

TEXT ·outw(SB),NOSPLIT,$0-4
	MOVW	port+0(FP), DX
	MOVW	val+2(FP), AX
	OUTW
	RET

TEXT ·freeze(SB),NOSPLIT|NOFRAME,$0
	HLT
	RET
