//go:build android && !cgo && arm64 && appmode

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

	// ART calls with its own x28. load_g does not replace g unless cgo
	// is linked, and cgocallback then loads g.m. On the lewkit-go thread
	// that word was nil, so the load faulted at 0. Keep g when it is
	// already m.g0: a JNI call runs there, and the recorded bounds of
	// that stack are only an estimate. Otherwise keep it when this stack
	// is the goroutine stack and m.g0 points back at that m. Anything
	// else is cleared so needm can attach an M. The saved x28 is restored.
	CBZ	g, crossHaveG
	MOVD	$0x10000, R4
	CMP	R4, g
	BLS	crossDropG
	TST	$7, g
	BNE	crossDropG
	MOVD	g, R4
	LSR	$56, R4, R4
	CBNZ	R4, crossDropG
	MOVD	48(g), R4
	CBZ	R4, crossDropG
	TST	$7, R4
	BNE	crossDropG
	MOVD	R4, R6
	LSR	$56, R6, R6
	CBNZ	R6, crossDropG
	MOVD	$0x10000, R7
	CMP	R7, R4
	BLS	crossDropG
	MOVD	(R4), R6
	CBZ	R6, crossDropG
	MOVD	48(R6), R7
	CMP	R4, R7
	BNE	crossDropG
	CMP	R6, g
	BEQ	crossHaveG
	MOVD	(g), R4
	MOVD	8(g), R5
	ADD	$(8*24), RSP, R16
	CMP	R4, R16
	BLS	crossDropG
	CMP	R5, R16
	BHS	crossDropG
	SUB	R4, R5, R6
	MOVD	$1024, R7
	CMP	R7, R6
	BLS	crossDropG
	MOVD	$0x40000000, R7
	CMP	R7, R6
	BHI	crossDropG
	B	crossHaveG
crossDropG:
	MOVD	ZR, g
crossHaveG:
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
// the new thread enters rt0_go. g stays clear through this callback so
// load_g, a no-op without cgo, leaves needm to attach an M. A later
// Java thread arrives with ART's x28; crosscall2 drops that unless it
// is already the g for this stack. x28 is restored for ART.
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

	// Point runtime sigaction at libc before libpreinit. A raw
	// rt_sigaction replaces libsigchain, so ART never sees the fault.
	// _cgo_sigaction is the linkname, not runtime._cgo_sigaction.
	MOVD	$·xCgoSigaction(SB), R5
	MOVD	$_cgo_sigaction(SB), R4
	MOVD	R5, (R4)

	// Indirect so this nosplit frame does not absorb libpreinit.
	// bootPreinit is the ABI0 wrapper around runtime.libpreinit.
	MOVD	$·bootPreinit(SB), R4
	BL	(R4)
	// libpreinit installed Go's SIGSEGV handler. Without cgo, that
	// handler trusts x28, and ART's value is not a g. A fault on a
	// Java or WebView thread is then Go's, and the process dies
	// with no tombstone. installSigGate runs before any Go thread.
	MOVD	$·installSigGate(SB), R4
	BL	(R4)
	// pthread_create here returns EAGAIN. The linker is still inside
	// this load, and bionic will not start a thread. Hook.call starts
	// the worker after the load, on an ART pthread. A cloned Go M
	// shares this TLS, and pthread_create from there does not return.
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

// installSigGate wraps the SIGBUS and SIGSEGV handlers libpreinit just
// installed. The saved address is runtime.sigtramp. Flags are copied
// whole, including SA_ONSTACK, so a Go fault is delivered on gsignal.
// The swap goes through libc sigaction, so libsigchain stays the kernel
// handler. g is nil on this thread; this frame is NOFRAME so the
// assembler does not load it.
TEXT ·installSigGate(SB),NOSPLIT|NOFRAME,$0
	SUB	$128, RSP
	MOVD	R30, 120(RSP)
	MOVD	R19, 112(RSP)

	// SIGBUS is 7, SIGSEGV is 11. Both use the same Go trampoline.
	MOVD	$7, R19

installOne:
	MOVD	ZR, 0(RSP)
	MOVD	ZR, 8(RSP)
	MOVD	ZR, 16(RSP)
	MOVD	ZR, 24(RSP)

	// xCgoSigaction(sig, nil, old). The struct is the kernel layout.
	MOVD	R19, R0
	MOVD	ZR, R1
	MOVD	RSP, R2
	BL	·xCgoSigaction(SB)
	CBNZ	R0, installNext

	// SIG_DFL is 0 and SIG_IGN is 1. Anything else is Go's trampoline.
	MOVD	0(RSP), R5
	CMP	$1, R5
	BLS	installNext
	MOVD	$·goSigtramp(SB), R10
	MOVD	R5, (R10)

	MOVD	$·sigGate(SB), R6
	MOVD	R6, 32(RSP)
	// sa_flags, including SA_ONSTACK. Go rejects a signal that
	// arrives off gsignal when this flag is missing.
	MOVD	8(RSP), R6
	MOVD	R6, 40(RSP)
	MOVD	16(RSP), R6
	MOVD	R6, 48(RSP)
	MOVD	24(RSP), R6
	MOVD	R6, 56(RSP)

	MOVD	R19, R0
	ADD	$32, RSP, R1
	MOVD	ZR, R2
	BL	·xCgoSigaction(SB)

installNext:
	CMP	$11, R19
	BEQ	installDone
	MOVD	$11, R19
	B	installOne

installDone:
	MOVD	112(RSP), R19
	MOVD	120(RSP), R30
	ADD	$128, RSP
	RET

// xCgoSigaction is the C function stored in runtime._cgo_sigaction.
// R0 is the signal, R1 the new kernel sigactiont or nil, R2 the old one
// or nil. Return 0, or errno. Bionic's struct sigaction is not the kernel
// struct: flags, pad, handler, mask, restorer, 32 bytes. SA_RESTORER is
// stripped so the VDSO restorer stays in place.
TEXT ·xCgoSigaction(SB),NOSPLIT|NOFRAME,$0
	SUB	$112, RSP
	STP	(R29, R30), 96(RSP)
	MOVD	R19, 80(RSP)
	MOVD	R20, 88(RSP)
	MOVD	R1, R19
	MOVD	R2, R20

	MOVD	ZR, 0(RSP)
	MOVD	ZR, 8(RSP)
	MOVD	ZR, 16(RSP)
	MOVD	ZR, 24(RSP)
	MOVD	ZR, 32(RSP)
	MOVD	ZR, 40(RSP)
	MOVD	ZR, 48(RSP)
	MOVD	ZR, 56(RSP)

	CBZ	R19, xCgoNoNew
	MOVD	8(R19), R3
	MOVD	$0x4000000, R4
	BIC	R4, R3, R3
	MOVW	R3, 0(RSP)
	MOVD	(R19), R3
	MOVD	R3, 8(RSP)
	MOVD	24(R19), R3
	MOVD	R3, 16(RSP)
	MOVD	RSP, R1
	B	xCgoHaveNew

xCgoNoNew:
	MOVD	ZR, R1

xCgoHaveNew:
	MOVD	ZR, R2
	CBZ	R20, xCgoCall
	ADD	$32, RSP, R2

xCgoCall:
	BL	libc_sigaction(SB)
	CMP	$0, R0
	BGE	xCgoOK
	BL	libc_errno(SB)
	MOVW	(R0), R0
	B	xCgoDone

xCgoOK:
	CBZ	R20, xCgoZero
	MOVD	40(RSP), R3
	MOVD	R3, (R20)
	MOVWU	32(RSP), R3
	MOVD	R3, 8(R20)
	MOVD	56(RSP), R3
	MOVD	R3, 16(R20)
	MOVD	48(RSP), R3
	MOVD	R3, 24(R20)

xCgoZero:
	MOVD	ZR, R0

xCgoDone:
	MOVD	80(RSP), R19
	MOVD	88(RSP), R20
	LDP	96(RSP), (R29, R30)
	ADD	$112, RSP
	RET

GLOBL ·goSigtramp(SB), NOPTR, $8

// sigGate is the wrapped SIGSEGV handler. R0 is the signal, R1 the
// siginfo, and R2 the kernel ucontext. Go 1.27's arm64 ucontext keeps
// mcontext at 176 and sigcontext.pc at 264, so the faulting PC is at
// 440(R2) and the faulting SP is at 432(R2). libsigchain is the kernel
// handler and runs ART's special handlers before this one. A PC inside
// Go text goes to sigtramp on the stack the kernel chose, which
// SA_ONSTACK makes gsignal. A PC outside that range is forwarded on
// the faulting thread's stack. bionic's sigaltstack is too small for
// debuggerd, and calling it there dies with no tombstone. There is no
// fallthrough into sigtramp from a foreign PC.
TEXT ·sigGate(SB),NOSPLIT|NOFRAME,$0
	MOVD	440(R2), R4
	MOVD	$runtime·text(SB), R5
	MOVD	$runtime·etext(SB), R6
	CMP	R5, R4
	BLO	sigForeign
	CMP	R6, R4
	BHS	sigForeign

	SUB	$48, RSP
	STP	(R29, R30), 32(RSP)
	MOVD	R0, 0(RSP)
	MOVD	R1, 8(RSP)
	MOVD	R2, 16(RSP)
	MOVD	g, 24(RSP)
	MOVD	0(RSP), R0
	MOVD	8(RSP), R1
	MOVD	16(RSP), R2
	MOVD	$·goSigtramp(SB), R3
	MOVD	(R3), R3
	CBZ	R3, sigGoDone
	BL	(R3)
sigGoDone:
	MOVD	24(RSP), g
	LDP	32(RSP), (R29, R30)
	ADD	$48, RSP
	RET

sigForeign:
	// 432(R2) is sigcontext.sp. Move there before calling the previous
	// handler. 160 bytes holds this frame; the callee uses the rest of
	// the faulting stack. A zero SP stays on the signal stack.
	MOVD	432(R2), R4
	CBZ	R4, sigForeignHere
	SUB	$160, R4, R4
	MOVD	RSP, R5
	MOVD	R4, RSP
	B	sigForeignFrame

sigForeignHere:
	SUB	$160, RSP
	MOVD	RSP, R5
	ADD	$160, R5, R5

sigForeignFrame:
	STP	(R29, R30), 144(RSP)
	MOVD	R5, 0(RSP)
	MOVD	R0, 8(RSP)
	MOVD	R1, 16(RSP)
	MOVD	R2, 24(RSP)
	MOVD	g, 32(RSP)
	MOVD	ZR, g

	// runtime.fwdSig is [_NSIG]uintptr. SIG_DFL is 0 and SIG_IGN is 1.
	MOVW	8(RSP), R5
	MOVD	$runtime·fwdSig(SB), R4
	LSL	$3, R5, R5
	MOVD	(R4)(R5), R4
	CMP	$1, R4
	BLS	sigForeignDefault
	MOVD	8(RSP), R0
	MOVD	16(RSP), R1
	MOVD	24(RSP), R2
	BL	(R4)
	B	sigForeignDone

sigForeignDefault:
	// No previous handler. Tell the chain this signal is SIG_DFL.
	// The faulting instruction runs again, and libsigchain's default
	// path can reach debuggerd. This is libc sigaction, not the raw
	// syscall, so the chain stays the kernel handler.
	MOVD	ZR, 48(RSP)
	MOVD	ZR, 56(RSP)
	MOVD	ZR, 64(RSP)
	MOVD	ZR, 72(RSP)
	MOVW	8(RSP), R0
	ADD	$48, RSP, R1
	MOVD	ZR, R2
	BL	·xCgoSigaction(SB)

sigForeignDone:
	MOVD	32(RSP), g
	MOVD	0(RSP), R5
	LDP	144(RSP), (R29, R30)
	MOVD	R5, RSP
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
// The worker starts here. This thread is an ART pthread and the
// library load has returned, so pthread_create can run. JNI_OnLoad
// cannot start it. g is ART's x28; startCThread does not load it.
TEXT Java_lewkit_Hook_call(SB),NOSPLIT|NOFRAME,$0
	SUB	$48, RSP
	STP	(R29, R30), 32(RSP)
	MOVD	R0, 0(RSP)
	MOVD	R2, 8(RSP)
	MOVD	R3, 16(RSP)
	// 16 bytes keeps startCThread's result off the saved JNI args.
	SUB	$16, RSP
	MOVD	$github·com∕lewtec∕lewkit∕x∕ffi∕native·startCThread(SB), R4
	BL	(R4)
	ADD	$16, RSP
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
