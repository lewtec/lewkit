//go:build android && !cgo && arm64

#include "textflag.h"

// arm64 Go and C put these integer arguments in the same registers.
// The ABI0 body reloads the stack frame the compiler wrapper spilled.

TEXT ·libcDlopen(SB),NOSPLIT,$0-24
	MOVD	path+0(FP), R0
	MOVD	mode+8(FP), R1
	BL	libc_dlopen(SB)
	MOVD	R0, ret+16(FP)
	RET

TEXT ·libcDlsym(SB),NOSPLIT,$0-24
	MOVD	handle+0(FP), R0
	MOVD	name+8(FP), R1
	BL	libc_dlsym(SB)
	MOVD	R0, ret+16(FP)
	RET

TEXT ·libcDlerror(SB),NOSPLIT,$0-8
	BL	libc_dlerror(SB)
	MOVD	R0, ret+0(FP)
	RET

TEXT ·libcDlclose(SB),NOSPLIT,$0-16
	MOVD	handle+0(FP), R0
	BL	libc_dlclose(SB)
	MOVD	R0, ret+8(FP)
	RET
