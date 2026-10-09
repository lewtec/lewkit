//go:build android && !cgo && arm64 && androidnocgo

package entry

import (
	"log/slog"
	"sync/atomic"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/jni"
	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
)

// runtimeNewextram makes an M for a Java thread that enters Go.
// mstartm0 only does this when cgo is linked. The Android link passes
// -checklinkname=0 because this symbol is not on the allowlist.
// The build schedule owns that flag and -tags androidnocgo.
//
//go:linkname runtimeNewextram runtime.newextram
func runtimeNewextram()

func addrHook() uintptr
func addrProxy() uintptr

// rtReady is 1 after that M exists. JNI_OnLoad waits for it before
// the first callback. booted makes the runtime start once.
// The pc values are the ABIInternal entry points. User assembly cannot
// name that ABI, so the callbacks are ordinary function values.
var (
	rtReady uint32
	booted  uint32

	pcLoad  uintptr
	pcHook  uintptr
	pcProxy uintptr
)

func init() {
	pcLoad = funcPC(onLoad)
	pcHook = funcPC(onHook)
	pcProxy = funcPC(onProxy)
	runtimeNewextram()
	atomic.StoreUint32(&rtReady, 1)
}

func funcPC(fn func(unsafe.Pointer)) uintptr {
	type fv struct{ fn uintptr }
	raw := *(*unsafe.Pointer)(unsafe.Pointer(&fn))
	return (*fv)(raw).fn
}

func words(frame unsafe.Pointer, n int) []uintptr {
	return unsafe.Slice((*uintptr)(frame), n)
}

func onLoad(frame unsafe.Pointer) {
	vm := words(frame, 1)[0]
	androidffi.NoteVM(vm)
	androidffi.NoteLoader()
	jni.SetVM(vm)
	env := jni.Attach()
	if env == 0 {
		slog.Error("jni hook", "err", "no env")
		return
	}
	// RegisterNatives binds the Java methods to these trampolines.
	// JNI_OnLoad is the symbol the loader has to publish. The method
	// names do not need their own dynamic exports, and they do not use cgo.
	if err := jni.Register(env, "lewkit/Hook", []jni.Native{{
		Name: "call",
		Sig:  "(Ljava/lang/String;[Ljava/lang/Object;)J",
		Fn:   addrHook(),
	}}); err != nil {
		slog.Error("jni hook", "err", err)
	}
	if err := jni.Register(env, "lewkit/GoProxy", []jni.Native{{
		Name: "nativeInvoke",
		Sig:  "(JLjava/lang/String;[Ljava/lang/Object;)Ljava/lang/Object;",
		Fn:   addrProxy(),
	}}); err != nil {
		slog.Error("jni proxy", "err", err)
	}
}

func onHook(frame unsafe.Pointer) {
	w := words(frame, 4)
	w[3] = hookCall(w[0], w[1], w[2])
}

func onProxy(frame unsafe.Pointer) {
	w := words(frame, 5)
	w[4] = jni.InvokeProxy(w[0], int64(w[1]), w[2], w[3])
}
