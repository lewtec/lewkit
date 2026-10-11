//go:build android && !cgo && arm64 && appmode

#include "textflag.h"

// These run on the calling goroutine. libcDlopen and the others call
// them only while the worker pthread is down. callCOnG0 reads eight
// argument words, so a hopped call passes that many.

TEXT ·libcDlopenDirect(SB),NOSPLIT,$0-24
	MOVD	path+0(FP), R0
	MOVD	mode+8(FP), R1
	BL	libc_dlopen(SB)
	MOVD	R0, ret+16(FP)
	RET

TEXT ·libcDlsymDirect(SB),NOSPLIT,$0-24
	MOVD	handle+0(FP), R0
	MOVD	name+8(FP), R1
	BL	libc_dlsym(SB)
	MOVD	R0, ret+16(FP)
	RET

TEXT ·libcDlerrorDirect(SB),NOSPLIT,$0-8
	BL	libc_dlerror(SB)
	MOVD	R0, ret+0(FP)
	RET

TEXT ·libcDlcloseDirect(SB),NOSPLIT,$0-16
	MOVD	handle+0(FP), R0
	BL	libc_dlclose(SB)
	MOVD	R0, ret+8(FP)
	RET

// Tail calls. callC already put the C arguments in R0–R7. A dynimport
// symbol has no address relocation, so the hopped call lands here.
TEXT ·dlopenTramp(SB),NOSPLIT|NOFRAME,$0
	B	libc_dlopen(SB)

TEXT ·dlsymTramp(SB),NOSPLIT|NOFRAME,$0
	B	libc_dlsym(SB)

TEXT ·dlerrorTramp(SB),NOSPLIT|NOFRAME,$0
	B	libc_dlerror(SB)

TEXT ·dlcloseTramp(SB),NOSPLIT|NOFRAME,$0
	B	libc_dlclose(SB)

TEXT ·libcDlopenPC(SB),NOSPLIT,$0-8
	MOVD	$·dlopenTramp(SB), R0
	MOVD	R0, ret+0(FP)
	RET

TEXT ·libcDlsymPC(SB),NOSPLIT,$0-8
	MOVD	$·dlsymTramp(SB), R0
	MOVD	R0, ret+0(FP)
	RET

TEXT ·libcDlerrorPC(SB),NOSPLIT,$0-8
	MOVD	$·dlerrorTramp(SB), R0
	MOVD	R0, ret+0(FP)
	RET

TEXT ·libcDlclosePC(SB),NOSPLIT,$0-8
	MOVD	$·dlcloseTramp(SB), R0
	MOVD	R0, ret+0(FP)
	RET

TEXT ·cThreadAlive(SB),NOSPLIT,$0-1
	MOVD	$·cComm+48(SB), R0
	LDARW	(R0), R0
	CBZ	R0, cThreadDown
	MOVW	$1, R0
cThreadDown:
	MOVB	R0, ret+0(FP)
	RET
