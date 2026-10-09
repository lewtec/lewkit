//go:build android && !cgo && arm64 && androidnocgo

#include "textflag.h"

// lewkit_crosscall2 matches runtime/cgo/asm_arm64.s crosscall2.
// C calls it with fn in R0, a frame pointer in R1, and ctxt in R3.
// g is R28. The assembler spells that register g, including in user code.
TEXT lewkit_crosscall2<>(SB),NOSPLIT|NOFRAME,$0
	SUB	$(8*24), RSP
	STP	(R0, R1), (8*1)(RSP)
	MOVD	R3, (8*3)(RSP)

	STP	(R19, R20), (8*4)(RSP)
	STP	(R21, R22), (8*6)(RSP)
	STP	(R23, R24), (8*8)(RSP)
	STP	(R25, R26), (8*10)(RSP)
	STP	(R27, g), (8*12)(RSP)
	FSTPD	(F8, F9), (8*14)(RSP)
	FSTPD	(F10, F11), (8*16)(RSP)
	FSTPD	(F12, F13), (8*18)(RSP)
	FSTPD	(F14, F15), (8*20)(RSP)
	STP	(R29, R30), (8*22)(RSP)

	BL	runtime·load_g(SB)
	BL	runtime·cgocallback(SB)

	LDP	(8*4)(RSP), (R19, R20)
	LDP	(8*6)(RSP), (R21, R22)
	LDP	(8*8)(RSP), (R23, R24)
	LDP	(8*10)(RSP), (R25, R26)
	LDP	(8*12)(RSP), (R27, g)
	FLDPD	(8*14)(RSP), (F8, F9)
	FLDPD	(8*16)(RSP), (F10, F11)
	FLDPD	(8*18)(RSP), (F12, F13)
	FLDPD	(8*20)(RSP), (F14, F15)
	LDP	(8*22)(RSP), (R29, R30)
	ADD	$(8*24), RSP
	RET

// JNI_OnLoad is the C entry ART calls. It starts the Go runtime once,
// then notes the JavaVM on this thread and registers Hook.call.
TEXT JNI_OnLoad(SB),NOSPLIT|NOFRAME,$0
	SUB	$32, RSP
	STP	(R29, R30), 16(RSP)
	MOVD	R0, 0(RSP)

	MOVD	$·booted(SB), R2
	LDARW	(R2), R3
	CBNZ	R3, jniLoaded

	MOVD	$1, R3
	STLRW	R3, (R2)

	MOVD	$runtime·islibrary(SB), R4
	MOVD	$1, R5
	MOVB	R5, (R4)
	BL	_rt0_arm64_android_lib(SB)

	MOVD	$·rtReady(SB), R2
jniSpin:
	LDARW	(R2), R3
	CBZ	R3, jniSpin

jniLoaded:
	MOVD	$·pcLoad(SB), R0
	MOVD	(R0), R0
	MOVD	RSP, R1
	MOVD	ZR, R2
	MOVD	ZR, R3
	BL	lewkit_crosscall2<>(SB)

	LDP	16(RSP), (R29, R30)
	ADD	$32, RSP
	MOVW	$0x00010006, R0
	RET

// Java_lewkit_Hook_call is the one host entry.
// R0 is JNIEnv*, R2 is the name jstring, R3 is the argument array.
// The jlong result is written at 24(RSP).
TEXT Java_lewkit_Hook_call(SB),NOSPLIT|NOFRAME,$0
	SUB	$48, RSP
	STP	(R29, R30), 32(RSP)
	MOVD	R0, 0(RSP)
	MOVD	R2, 8(RSP)
	MOVD	R3, 16(RSP)
	MOVD	$·pcHook(SB), R0
	MOVD	(R0), R0
	MOVD	RSP, R1
	MOVD	ZR, R2
	MOVD	ZR, R3
	BL	lewkit_crosscall2<>(SB)
	MOVD	24(RSP), R0
	LDP	32(RSP), (R29, R30)
	ADD	$48, RSP
	RET

TEXT Java_lewkit_GoProxy_nativeInvoke(SB),NOSPLIT|NOFRAME,$0
	SUB	$64, RSP
	STP	(R29, R30), 48(RSP)
	MOVD	R0, 0(RSP)
	MOVD	R2, 8(RSP)
	MOVD	R3, 16(RSP)
	MOVD	R4, 24(RSP)
	MOVD	$·pcProxy(SB), R0
	MOVD	(R0), R0
	MOVD	RSP, R1
	MOVD	ZR, R2
	MOVD	ZR, R3
	BL	lewkit_crosscall2<>(SB)
	MOVD	32(RSP), R0
	LDP	48(RSP), (R29, R30)
	ADD	$64, RSP
	RET

TEXT ·addrHook(SB),NOSPLIT,$0-8
	MOVD	$Java_lewkit_Hook_call(SB), R0
	MOVD	R0, ret+0(FP)
	RET

TEXT ·addrProxy(SB),NOSPLIT,$0-8
	MOVD	$Java_lewkit_GoProxy_nativeInvoke(SB), R0
	MOVD	R0, ret+0(FP)
	RET
