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
//
// _rt0_arm64_lib clears g and libInit then calls newosproc0. The 8MB
// stack allocation reaches setVMAName, which is not nosplit, so the
// prologue loads g.stackguard0. With g nil that is a fault at 0x10 on
// this thread. libpreinit is nosplit. The stack comes from mmap, and
// the new thread enters rt0_go. g stays clear through the callback so
// load_g, a no-op without cgo, leaves needm to attach an M. x28 is
// restored for ART.
TEXT JNI_OnLoad(SB),NOSPLIT|NOFRAME,$0
	SUB	$32, RSP
	STP	(R29, R30), 16(RSP)
	MOVD	R0, 0(RSP)
	MOVD	g, 8(RSP)
	MOVD	ZR, g

	MOVD	$·booted(SB), R2
	LDARW	(R2), R3
	CBNZ	R3, jniLoaded

	MOVD	$1, R3
	STLRW	R3, (R2)

	MOVD	$runtime·islibrary(SB), R4
	MOVD	$1, R5
	MOVB	R5, (R4)

	// Indirect so this nosplit frame does not absorb libpreinit.
	// bootPreinit is the ABI0 wrapper around runtime.libpreinit.
	MOVD	$·bootPreinit(SB), R4
	BL	(R4)
	MOVD	$·startRuntime(SB), R4
	BL	(R4)
	CMP	$0, R0
	BLE	jniFail

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

	MOVD	8(RSP), g
	LDP	16(RSP), (R29, R30)
	ADD	$32, RSP
	MOVW	$0x00010006, R0
	RET

jniFail:
	MOVD	8(RSP), g
	LDP	16(RSP), (R29, R30)
	ADD	$32, RSP
	MOVD	ZR, R0
	RET

// startRuntime maps an 8MB stack and clones a thread at rt0entry.
// R0 is the clone result, or -1 when mmap fails. A positive tid is success.
// arm64 0(FP) is 8(RSP): the assembler reserves a return slot at 0(RSP).
TEXT ·startRuntime(SB),NOSPLIT|NOFRAME,$0
	SUB	$64, RSP
	MOVD	R30, 56(RSP)

	// mmap(nil, 8MB, PROT_READ|PROT_WRITE, MAP_ANON|MAP_PRIVATE, -1, 0)
	MOVD	ZR, R0
	MOVD	$0x800000, R1
	MOVD	$3, R2
	MOVD	$0x22, R3
	MOVW	$-1, R4
	MOVD	ZR, R5
	MOVD	$222, R8
	SVC
	CMN	$4095, R0
	BCS	stackFail

	ADD	$0x800000, R0, R1
	// CLONE_VM|CLONE_FS|CLONE_FILES|CLONE_SIGHAND|CLONE_THREAD|CLONE_SYSVSEM
	MOVD	$0x50F00, R0
	MOVD	R0, 8(RSP)
	MOVD	R1, 16(RSP)
	MOVD	ZR, 24(RSP)
	MOVD	ZR, 32(RSP)
	MOVD	$·rt0entry(SB), R0
	MOVD	R0, 40(RSP)
	BL	runtime·clone(SB)

	MOVD	56(RSP), R30
	ADD	$64, RSP
	RET

stackFail:
	MOVD	56(RSP), R30
	ADD	$64, RSP
	MOVD	$-1, R0
	RET

// rt0entry is the new thread. argc and argv match the Android lib
// startup: one argument, then empty env and auxv, so sysargs reads
// /proc/self/auxv.
TEXT ·rt0entry(SB),NOSPLIT|NOFRAME,$0
	MOVD	$runtime·rt0_go(SB), R4
	MOVW	$1, R0
	MOVD	$·bootArgv(SB), R1
	B	(R4)

DATA ·bootArgv+0(SB)/8, $·bootArgv0(SB)
DATA ·bootArgv+8(SB)/8, $0
DATA ·bootArgv+16(SB)/8, $0
DATA ·bootArgv+24(SB)/8, $0
GLOBL ·bootArgv(SB), NOPTR, $32

DATA ·bootArgv0+0(SB)/8, $"gojni"
GLOBL ·bootArgv0(SB), RODATA, $8

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
