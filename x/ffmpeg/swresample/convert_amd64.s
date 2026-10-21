//go:build amd64

#include "textflag.h"

// float32(1/32768) replicated
DATA scaleS16<>(SB)/4, $0x38000000
DATA scaleS16<>+4(SB)/4, $0x38000000
DATA scaleS16<>+8(SB)/4, $0x38000000
DATA scaleS16<>+12(SB)/4, $0x38000000
GLOBL scaleS16<>(SB), RODATA, $16

// float32(32768)
DATA scaleF32<>(SB)/4, $0x47000000
DATA scaleF32<>+4(SB)/4, $0x47000000
DATA scaleF32<>+8(SB)/4, $0x47000000
DATA scaleF32<>+12(SB)/4, $0x47000000
GLOBL scaleF32<>(SB), RODATA, $16

// float32(-1)
DATA clampLo<>(SB)/4, $0xbf800000
DATA clampLo<>+4(SB)/4, $0xbf800000
DATA clampLo<>+8(SB)/4, $0xbf800000
DATA clampLo<>+12(SB)/4, $0xbf800000
GLOBL clampLo<>(SB), RODATA, $16

// float32(32767)/32768
DATA clampHi<>(SB)/4, $0x3f7ffe00
DATA clampHi<>+4(SB)/4, $0x3f7ffe00
DATA clampHi<>+8(SB)/4, $0x3f7ffe00
DATA clampHi<>+12(SB)/4, $0x3f7ffe00
GLOBL clampHi<>(SB), RODATA, $16

// func s16ToF32AsmRaw(dst *float32, src *int16, n int)
TEXT ·s16ToF32AsmRaw(SB), NOSPLIT, $0-24
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ n+16(FP), CX
	SHRQ $3, CX
	TESTQ CX, CX
	JZ   s16done
	MOVUPS scaleS16<>(SB), X5
s16loop:
	MOVUPS (SI), X0
	MOVO   X0, X1
	PXOR   X2, X2
	PCMPGTW X0, X2
	PUNPCKLWL X2, X0
	PUNPCKHWL X2, X1
	CVTPL2PS X0, X0
	CVTPL2PS X1, X1
	MULPS X5, X0
	MULPS X5, X1
	MOVUPS X0, (DI)
	MOVUPS X1, 16(DI)
	ADDQ $16, SI
	ADDQ $32, DI
	DECQ CX
	JNZ  s16loop
s16done:
	RET

// func f32ToS16AsmRaw(dst *int16, src *float32, n int)
TEXT ·f32ToS16AsmRaw(SB), NOSPLIT, $0-24
	MOVQ dst+0(FP), DI
	MOVQ src+8(FP), SI
	MOVQ n+16(FP), CX
	SHRQ $3, CX
	TESTQ CX, CX
	JZ   f32done
	MOVUPS scaleF32<>(SB), X5
	MOVUPS clampLo<>(SB), X6
	MOVUPS clampHi<>(SB), X7
f32loop:
	MOVUPS (SI), X0
	MOVUPS 16(SI), X1
	MAXPS X6, X0
	MAXPS X6, X1
	MINPS X7, X0
	MINPS X7, X1
	MULPS X5, X0
	MULPS X5, X1
	CVTPS2PL X0, X0
	CVTPS2PL X1, X1
	PACKSSLW X1, X0
	MOVUPS X0, (DI)
	ADDQ $32, SI
	ADDQ $16, DI
	DECQ CX
	JNZ  f32loop
f32done:
	RET
