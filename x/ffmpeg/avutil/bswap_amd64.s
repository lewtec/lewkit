//go:build amd64

#include "textflag.h"

// func bswap16Pairs(p *byte, n int)
TEXT ·bswap16Pairs(SB), NOSPLIT, $0-16
	MOVQ p+0(FP), SI
	MOVQ n+8(FP), CX
	TESTQ CX, CX
	JZ done
loop:
	MOVW (SI), AX
	ROLW $8, AX
	MOVW AX, (SI)
	ADDQ $2, SI
	DECQ CX
	JNZ loop
done:
	RET

// func bswap32Groups(p *byte, n int)
TEXT ·bswap32Groups(SB), NOSPLIT, $0-16
	MOVQ p+0(FP), SI
	MOVQ n+8(FP), CX
	TESTQ CX, CX
	JZ done
loop:
	MOVL (SI), AX
	BSWAPL AX
	MOVL AX, (SI)
	ADDQ $4, SI
	DECQ CX
	JNZ loop
done:
	RET
