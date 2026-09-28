//go:build android && cgo

package android

import (
	"fmt"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/native"
)

// JNI function table indexes from jni.h JNINativeInterface.
const (
	idxGetVersion            = 4
	idxFindClass             = 6
	idxExceptionClear        = 17
	idxNewGlobalRef          = 21
	idxDeleteLocalRef        = 23
	idxGetStaticMethodID     = 113
	idxCallStaticObjectA     = 116
	idxCallStaticVoidA       = 143
	idxNewStringUTF          = 167
	idxGetStringUTFChars     = 169
	idxReleaseStringUTFChars = 170
)

// JavaVM invoke table indexes.
const (
	idxAttachCurrentThread = 4
	idxGetEnv              = 6
	jniVersion16           = 0x00010006
)

// Env is a JNIEnv. self is the pointer passed to JNI functions. tab is the
// function table. On Android those are not the same address: the env object
// points at the table.
type Env struct {
	self uintptr
	tab  uintptr
}

var (
	hostClass  uintptr
	currentEnv func() uintptr
)

// SetCurrentEnv supplies a JNIEnv for the calling thread. The function is
// called on each Java call so a Go thread can attach.
func SetCurrentEnv(fn func() uintptr) { currentEnv = fn }

// BindHost records lewkit.Host from a Java thread. Later Go threads attach
// and call that global ref. FindClass on an attached Go thread does not see
// the app class loader.
// SetHostClass records the global lewkit.Host reference taken on the Java thread.
func SetHostClass(cls uintptr) { hostClass = cls }

func BindHost(env uintptr) error {
	e := openEnv(env)
	cls, err := e.findClass("lewkit/Host")
	if err != nil {
		return err
	}
	g := e.newGlobalRef(cls)
	if g == 0 {
		return fmt.Errorf("lewkit/Host global ref")
	}
	hostClass = g
	return nil
}

// StaticString calls a no-argument static method that returns a String.
func StaticString(name string) (string, error) {
	return StaticStrings(name, "()Ljava/lang/String;")
}

// StaticStrings calls a static method that returns a String.
// strArgs are the Ljava/lang/String; parameters, in order.
func StaticStrings(name, sig string, strArgs ...string) (string, error) {
	env, err := attach()
	if err != nil {
		return "", err
	}
	cls, err := host()
	if err != nil {
		return "", err
	}
	mid, err := env.method(cls, name, sig)
	if err != nil {
		return "", err
	}
	vals := make([]jvalue, len(strArgs))
	for i, s := range strArgs {
		vals[i].obj = env.newString(s)
		if vals[i].obj == 0 {
			env.clear()
			for _, v := range vals[:i] {
				env.deleteLocal(v.obj)
			}
			return "", fmt.Errorf("android host %s string", name)
		}
	}
	defer func() {
		for _, v := range vals {
			env.deleteLocal(v.obj)
		}
	}()
	var args uintptr
	if len(vals) > 0 {
		args = uintptr(unsafe.Pointer(&vals[0]))
	}
	var fn func(env, cls, mid, args uintptr) uintptr
	fp, err := env.fn(idxCallStaticObjectA)
	if err != nil {
		return "", err
	}
	native.Register(&fn, fp)
	obj := fn(env.ptr(), cls, mid, args)
	if obj == 0 {
		env.clear()
		return "", fmt.Errorf("android host %s", name)
	}
	out := env.goString(obj)
	env.deleteLocal(obj)
	return out, nil
}

// StaticVoid calls a static void method. strArgs fill Ljava/lang/String; parameters.
func StaticVoid(name, sig string, strArgs ...string) error {
	env, err := attach()
	if err != nil {
		return err
	}
	cls, err := host()
	if err != nil {
		return err
	}
	mid, err := env.method(cls, name, sig)
	if err != nil {
		return err
	}
	vals := make([]jvalue, len(strArgs))
	for i, s := range strArgs {
		vals[i].obj = env.newString(s)
		if vals[i].obj == 0 {
			return fmt.Errorf("android host %s string", name)
		}
	}
	var args uintptr
	if len(vals) > 0 {
		args = uintptr(unsafe.Pointer(&vals[0]))
	}
	var fn func(env, cls, mid, args uintptr)
	fp, err := env.fn(idxCallStaticVoidA)
	if err != nil {
		return err
	}
	native.Register(&fn, fp)
	fn(env.ptr(), cls, mid, args)
	env.clear()
	return nil
}

// JString reads a jstring from the Java thread's JNIEnv.
func JString(env, jstr uintptr) (string, error) {
	if jstr == 0 {
		return "", fmt.Errorf("null jstring")
	}
	return openEnv(env).goString(jstr), nil
}

// normalize accepts either the JNI function table or a pointer to it.
// cgo types JNIEnv as *JNINativeInterface, and the export parameter adds another pointer.
func openEnv(raw uintptr) Env {
	e := Env{self: raw, tab: raw}
	if raw == 0 || e.slot(idxFindClass) != 0 {
		return e
	}
	inner := untag(e.slot(0))
	if inner != 0 && (Env{self: raw, tab: inner}).slot(idxFindClass) != 0 {
		e.tab = inner
	}
	return e
}

func untag(p uintptr) uintptr { return p & 0x00ffffffffffffff }

type jvalue struct {
	obj uintptr
}

func host() (uintptr, error) {
	if hostClass == 0 {
		return 0, fmt.Errorf("lewkit/Host is not bound")
	}
	return hostClass, nil
}

func (e Env) ptr() uintptr { return e.self }

func (e Env) slot(i int) uintptr {
	return *(*uintptr)(unsafe.Pointer(e.tab + uintptr(i)*unsafe.Sizeof(uintptr(0))))
}

func (e Env) clear() {
	fp, err := e.fn(idxExceptionClear)
	if err != nil {
		return
	}
	var fn func(env uintptr)
	native.Register(&fn, fp)
	fn(e.ptr())
}

func (e Env) findClass(name string) (uintptr, error) {
	b := cstr(name)
	fp, err := e.fn(idxFindClass)
	if err != nil {
		return 0, err
	}
	var fn func(env, name uintptr) uintptr
	native.Register(&fn, fp)
	cls := fn(e.ptr(), uintptr(unsafe.Pointer(&b[0])))
	if cls == 0 {
		e.clear()
		return 0, fmt.Errorf("FindClass %s", name)
	}
	return cls, nil
}

func (e Env) deleteLocal(obj uintptr) {
	if obj == 0 {
		return
	}
	fp, err := e.fn(idxDeleteLocalRef)
	if err != nil {
		return
	}
	var fn func(env, obj uintptr)
	native.Register(&fn, fp)
	fn(e.ptr(), obj)
}

func (e Env) newGlobalRef(obj uintptr) uintptr {
	fp, err := e.fn(idxNewGlobalRef)
	if err != nil {
		return 0
	}
	var fn func(env, obj uintptr) uintptr
	native.Register(&fn, fp)
	return fn(e.ptr(), obj)
}

func (e Env) method(cls uintptr, name, sig string) (uintptr, error) {
	nb, sb := cstr(name), cstr(sig)
	fp, err := e.fn(idxGetStaticMethodID)
	if err != nil {
		return 0, err
	}
	var fn func(env, cls, name, sig uintptr) uintptr
	native.Register(&fn, fp)
	mid := fn(e.ptr(), cls, uintptr(unsafe.Pointer(&nb[0])), uintptr(unsafe.Pointer(&sb[0])))
	if mid == 0 {
		e.clear()
		return 0, fmt.Errorf("GetStaticMethodID %s %s", name, sig)
	}
	return mid, nil
}

func (e Env) newString(s string) uintptr {
	b := cstr(s)
	fp, err := e.fn(idxNewStringUTF)
	if err != nil {
		return 0
	}
	var fn func(env, utf uintptr) uintptr
	native.Register(&fn, fp)
	return fn(e.ptr(), uintptr(unsafe.Pointer(&b[0])))
}

func (e Env) goString(jstr uintptr) string {
	getp, err := e.fn(idxGetStringUTFChars)
	if err != nil {
		return ""
	}
	var get func(env, str, isCopy uintptr) uintptr
	native.Register(&get, getp)
	p := get(e.ptr(), jstr, 0)
	if p == 0 {
		return ""
	}
	s := goCString(p)
	relp, err := e.fn(idxReleaseStringUTFChars)
	if err != nil {
		return s
	}
	var rel func(env, str, chars uintptr)
	native.Register(&rel, relp)
	rel(e.ptr(), jstr, p)
	return s
}

func attach() (Env, error) {
	if currentEnv == nil {
		return Env{}, fmt.Errorf("jni env is not set")
	}
	raw := currentEnv()
	if raw == 0 {
		return Env{}, fmt.Errorf("jni attach")
	}
	return openEnv(raw), nil
}

func (e Env) fn(i int) (uintptr, error) {
	p := e.slot(i)
	if p == 0 {
		return 0, fmt.Errorf("jni slot %d is nil", i)
	}
	return p, nil
}

func vmSlot(vm uintptr, i int) uintptr {
	return *(*uintptr)(unsafe.Pointer(vm + uintptr(i)*unsafe.Sizeof(uintptr(0))))
}

func javaVM() (uintptr, error) {
	lib, err := open("libnativehelper.so", "libart.so")
	if err != nil {
		return 0, err
	}
	sym, err := native.Symbol(lib, "JNI_GetCreatedJavaVMs")
	if err != nil {
		return 0, err
	}
	var fn func(vmBuf *uintptr, bufLen int32, nVMs *int32) int32
	native.Register(&fn, sym)
	var vm uintptr
	var n int32
	if rc := fn(&vm, 1, &n); rc != 0 || n < 1 || vm == 0 {
		return 0, fmt.Errorf("JNI_GetCreatedJavaVMs: %d", rc)
	}
	return vm, nil
}

func cstr(s string) []byte {
	b := make([]byte, len(s)+1)
	copy(b, s)
	return b
}

func goCString(p uintptr) string {
	var buf []byte
	for {
		c := *(*byte)(unsafe.Pointer(p + uintptr(len(buf))))
		if c == 0 {
			break
		}
		buf = append(buf, c)
	}
	return string(buf)
}
