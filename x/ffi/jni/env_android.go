//go:build android && cgo

package jni

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/native"
)

// JNINativeInterface indexes from the NDK jni.h.
const (
	idxIsAssignableFrom      = 11
	idxExceptionOccurred     = 15
	idxExceptionClear        = 17
	idxPushLocalFrame        = 19
	idxPopLocalFrame         = 20
	idxNewGlobalRef          = 21
	idxDeleteGlobalRef       = 22
	idxFindClass             = 6
	idxGetObjectClass        = 31
	idxGetMethodID           = 33
	idxCallObjectA           = 36
	idxCallBooleanA          = 39
	idxCallIntA              = 51
	idxCallLongA             = 54
	idxCallFloatA            = 57
	idxCallDoubleA           = 60
	idxGetStaticMethodID     = 113
	idxCallStaticObjectA     = 116
	idxNewStringUTF          = 167
	idxGetStringUTFLength    = 168
	idxGetStringUTFChars     = 169
	idxReleaseStringUTFChars = 170
	idxGetArrayLength        = 171
	idxNewObjectArray        = 172
	idxGetObjectArrayElement = 173
	idxSetObjectArrayElement = 174
)

var (
	errNoEnv         = errors.New("jni env is not set")
	errJava          = errors.New("java exception")
	errNilSlot       = errors.New("jni slot is nil")
	errLocalFrame    = errors.New("jni local frame")
	errClass         = errors.New("class")
	errMissingMethod = errors.New("missing method")
	errNullRef       = errors.New("jni null reference")
	errGlobalRef     = errors.New("jni global ref")
	errNullClass     = errors.New("java class is null")
	errObjectArray   = errors.New("jni object array")
)

// env is one JNIEnv. self is the pointer passed to JNI functions.
// tab is the function table. On Android those are not always the same address.
type env struct {
	self uintptr
	tab  uintptr
}

type errIDs struct {
	toString uintptr
	getCause uintptr
}

type jvalue struct {
	bits uint64
}

func jobj(p uintptr) jvalue { return jvalue{bits: uint64(p)} }
func jbool(v bool) jvalue {
	if v {
		return jvalue{bits: 1}
	}
	return jvalue{}
}
func ji32(v int32) jvalue   { return jvalue{bits: uint64(uint32(v))} }
func ji64(v int64) jvalue   { return jvalue{bits: uint64(v)} }
func jf32(v float32) jvalue { return jvalue{bits: uint64(math.Float32bits(v))} }
func jf64(v float64) jvalue { return jvalue{bits: math.Float64bits(v)} }

func argv(vals []jvalue) uintptr {
	if len(vals) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(&vals[0]))
}

func cstr(s string) []byte {
	b := make([]byte, len(s)+1)
	copy(b, s)
	return b
}

func cptr(b []byte) uintptr {
	if len(b) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(&b[0]))
}

// openEnv separates the JNIEnv from its function table.
// A non-null word at FindClass is not enough: JNIEnvExt fills that slot
// after the first calls, and indexing the env object jumps into the heap.
func openEnv(raw uintptr) env {
	self, tab, ok := functionTable(raw, func(p uintptr) uintptr {
		return *(*uintptr)(unsafe.Pointer(p))
	})
	if !ok {
		return env{}
	}
	return env{self: self, tab: tab}
}

func (e env) slot(i int) uintptr {
	return *(*uintptr)(unsafe.Pointer(e.tab + uintptr(i)*unsafe.Sizeof(uintptr(0))))
}

func (e env) fn(slot int) (uintptr, error) {
	if e.tab == 0 {
		return 0, errNoEnv
	}
	p := e.slot(slot)
	if p == 0 {
		return 0, fmt.Errorf("%w: %d", errNilSlot, slot)
	}
	return p, nil
}

func (e env) push(n int32) error {
	fp, err := e.fn(idxPushLocalFrame)
	if err != nil {
		return err
	}
	var fn func(uintptr, int32) int32
	native.Register(&fn, fp)
	if rc := fn(e.self, n); rc != 0 {
		e.clear()
		return fmt.Errorf("%w: %d", errLocalFrame, rc)
	}
	return nil
}

func (e env) pop() {
	fp, err := e.fn(idxPopLocalFrame)
	if err != nil {
		return
	}
	var fn func(uintptr, uintptr) uintptr
	native.Register(&fn, fp)
	fn(e.self, 0)
}

func (e env) occurred() uintptr {
	fp, err := e.fn(idxExceptionOccurred)
	if err != nil {
		return 0
	}
	var fn func(uintptr) uintptr
	native.Register(&fn, fp)
	return fn(e.self)
}

func (e env) clear() {
	fp, err := e.fn(idxExceptionClear)
	if err != nil {
		return
	}
	var fn func(uintptr)
	native.Register(&fn, fp)
	fn(e.self)
}

func (e env) raw4(slot int, a, b, c, d uintptr) (uintptr, error) {
	fp, err := e.fn(slot)
	if err != nil {
		return 0, err
	}
	var fn func(uintptr, uintptr, uintptr, uintptr) uintptr
	native.Register(&fn, fp)
	return fn(a, b, c, d), nil
}

func (e env) callObject(slot int, a, b, c uintptr, args []jvalue, ids errIDs) (uintptr, error) {
	out, err := e.raw4(slot, a, b, c, argv(args))
	runtime.KeepAlive(args)
	if err != nil {
		return 0, err
	}
	if err := e.ex(ids); err != nil {
		return 0, err
	}
	return out, nil
}

func (e env) ex(ids errIDs) error {
	thr := e.occurred()
	if thr == 0 {
		return nil
	}
	e.clear()
	for range 8 {
		if ids.getCause == 0 {
			break
		}
		cause, err := e.raw4(idxCallObjectA, e.self, thr, ids.getCause, 0)
		if err != nil || e.occurred() != 0 || cause == 0 {
			e.clear()
			break
		}
		thr = cause
	}
	if ids.toString == 0 {
		return errJava
	}
	msg, err := e.raw4(idxCallObjectA, e.self, thr, ids.toString, 0)
	if err != nil || e.occurred() != 0 || msg == 0 {
		e.clear()
		return errJava
	}
	s, err := e.readString(msg)
	if err != nil || s == "" {
		e.clear()
		return errJava
	}
	return fmt.Errorf("%w: %s", errJava, s)
}

func (e env) find(name string, ids errIDs) (uintptr, error) {
	b := cstr(slashed(name))
	fp, err := e.fn(idxFindClass)
	if err != nil {
		return 0, err
	}
	var fn func(uintptr, uintptr) uintptr
	native.Register(&fn, fp)
	cls := fn(e.self, cptr(b))
	runtime.KeepAlive(b)
	if err := e.ex(ids); err != nil {
		return 0, err
	}
	if cls == 0 {
		return 0, fmt.Errorf("%w: %s", errClass, dotted(name))
	}
	return cls, nil
}

func (e env) methodID(slot int, cls uintptr, name, sig string, ids errIDs) (uintptr, error) {
	nb, sb := cstr(name), cstr(sig)
	id, err := e.raw4(slot, e.self, cls, cptr(nb), cptr(sb))
	runtime.KeepAlive(nb)
	runtime.KeepAlive(sb)
	if err != nil {
		return 0, err
	}
	if err := e.ex(ids); err != nil {
		return 0, err
	}
	if id == 0 {
		return 0, fmt.Errorf("%w: %s%s", errMissingMethod, name, sig)
	}
	return id, nil
}

func (e env) global(obj uintptr, ids errIDs) (uintptr, error) {
	if obj == 0 {
		return 0, errNullRef
	}
	fp, err := e.fn(idxNewGlobalRef)
	if err != nil {
		return 0, err
	}
	var fn func(uintptr, uintptr) uintptr
	native.Register(&fn, fp)
	g := fn(e.self, obj)
	if err := e.ex(ids); err != nil {
		return 0, err
	}
	if g == 0 {
		return 0, errGlobalRef
	}
	return g, nil
}

func (e env) deleteGlobal(obj uintptr) {
	if obj == 0 {
		return
	}
	fp, err := e.fn(idxDeleteGlobalRef)
	if err != nil {
		return
	}
	var fn func(uintptr, uintptr)
	native.Register(&fn, fp)
	fn(e.self, obj)
}

func (e env) objectClass(obj uintptr, ids errIDs) (uintptr, error) {
	fp, err := e.fn(idxGetObjectClass)
	if err != nil {
		return 0, err
	}
	var fn func(uintptr, uintptr) uintptr
	native.Register(&fn, fp)
	cls := fn(e.self, obj)
	if err := e.ex(ids); err != nil {
		return 0, err
	}
	if cls == 0 {
		return 0, errNullClass
	}
	return cls, nil
}

func (e env) isAssign(from, to uintptr) bool {
	if from == 0 || to == 0 {
		return false
	}
	fp, err := e.fn(idxIsAssignableFrom)
	if err != nil {
		return false
	}
	var fn func(uintptr, uintptr, uintptr) uint8
	native.Register(&fn, fp)
	return fn(e.self, from, to) != 0
}

func (e env) arrayLen(arr uintptr, ids errIDs) (int32, error) {
	fp, err := e.fn(idxGetArrayLength)
	if err != nil {
		return 0, err
	}
	var fn func(uintptr, uintptr) int32
	native.Register(&fn, fp)
	n := fn(e.self, arr)
	if err := e.ex(ids); err != nil {
		return 0, err
	}
	return n, nil
}

func (e env) element(arr uintptr, i int32, ids errIDs) (uintptr, error) {
	fp, err := e.fn(idxGetObjectArrayElement)
	if err != nil {
		return 0, err
	}
	var fn func(uintptr, uintptr, int32) uintptr
	native.Register(&fn, fp)
	obj := fn(e.self, arr, i)
	if err := e.ex(ids); err != nil {
		return 0, err
	}
	return obj, nil
}

func (e env) setElement(arr uintptr, i int32, val uintptr, ids errIDs) error {
	fp, err := e.fn(idxSetObjectArrayElement)
	if err != nil {
		return err
	}
	var fn func(uintptr, uintptr, int32, uintptr)
	native.Register(&fn, fp)
	fn(e.self, arr, i, val)
	return e.ex(ids)
}

func (e env) newObjectArray(n int32, cls uintptr, ids errIDs) (uintptr, error) {
	fp, err := e.fn(idxNewObjectArray)
	if err != nil {
		return 0, err
	}
	var fn func(uintptr, int32, uintptr, uintptr) uintptr
	native.Register(&fn, fp)
	arr := fn(e.self, n, cls, 0)
	if err := e.ex(ids); err != nil {
		return 0, err
	}
	if arr == 0 {
		return 0, errObjectArray
	}
	return arr, nil
}

func (e env) newString(s string, ids errIDs) (uintptr, error) {
	b := append(encodeMUTF8(nil, s), 0)
	fp, err := e.fn(idxNewStringUTF)
	if err != nil {
		return 0, err
	}
	var fn func(uintptr, uintptr) uintptr
	native.Register(&fn, fp)
	out := fn(e.self, cptr(b))
	runtime.KeepAlive(b)
	if err := e.ex(ids); err != nil {
		return 0, err
	}
	if out == 0 {
		return 0, errJava
	}
	return out, nil
}

func (e env) readString(jstr uintptr) (string, error) {
	if jstr == 0 {
		return "", nil
	}
	fp, err := e.fn(idxGetStringUTFLength)
	if err != nil {
		return "", err
	}
	var length func(uintptr, uintptr) int32
	native.Register(&length, fp)
	n := length(e.self, jstr)
	if e.occurred() != 0 {
		e.clear()
		return "", errJava
	}
	if n == 0 {
		return "", nil
	}
	if n < 0 {
		return "", errJava
	}
	fp, err = e.fn(idxGetStringUTFChars)
	if err != nil {
		return "", err
	}
	var chars func(uintptr, uintptr, uintptr) uintptr
	native.Register(&chars, fp)
	p := chars(e.self, jstr, 0)
	if e.occurred() != 0 || p == 0 {
		e.clear()
		return "", errJava
	}
	buf := make([]byte, n)
	copy(buf, unsafe.Slice((*byte)(unsafe.Pointer(p)), n))
	fp, err = e.fn(idxReleaseStringUTFChars)
	if err == nil {
		var rel func(uintptr, uintptr, uintptr)
		native.Register(&rel, fp)
		rel(e.self, jstr, p)
	}
	return decodeMUTF8(buf), nil
}

func (e env) callInt(obj, mid uintptr, args []jvalue, ids errIDs) (int32, error) {
	fp, err := e.fn(idxCallIntA)
	if err != nil {
		return 0, err
	}
	var fn func(uintptr, uintptr, uintptr, uintptr) int32
	native.Register(&fn, fp)
	out := fn(e.self, obj, mid, argv(args))
	runtime.KeepAlive(args)
	if err := e.ex(ids); err != nil {
		return 0, err
	}
	return out, nil
}

func (e env) callLong(obj, mid uintptr, ids errIDs) (int64, error) {
	fp, err := e.fn(idxCallLongA)
	if err != nil {
		return 0, err
	}
	var fn func(uintptr, uintptr, uintptr, uintptr) int64
	native.Register(&fn, fp)
	out := fn(e.self, obj, mid, 0)
	if err := e.ex(ids); err != nil {
		return 0, err
	}
	return out, nil
}

func (e env) callBool(obj, mid uintptr, ids errIDs) (bool, error) {
	fp, err := e.fn(idxCallBooleanA)
	if err != nil {
		return false, err
	}
	var fn func(uintptr, uintptr, uintptr, uintptr) uint8
	native.Register(&fn, fp)
	out := fn(e.self, obj, mid, 0)
	if err := e.ex(ids); err != nil {
		return false, err
	}
	return out != 0, nil
}

func (e env) callFloat(obj, mid uintptr, ids errIDs) (float32, error) {
	fp, err := e.fn(idxCallFloatA)
	if err != nil {
		return 0, err
	}
	var fn func(uintptr, uintptr, uintptr, uintptr) float32
	native.Register(&fn, fp)
	out := fn(e.self, obj, mid, 0)
	if err := e.ex(ids); err != nil {
		return 0, err
	}
	return out, nil
}

func (e env) callDouble(obj, mid uintptr, ids errIDs) (float64, error) {
	fp, err := e.fn(idxCallDoubleA)
	if err != nil {
		return 0, err
	}
	var fn func(uintptr, uintptr, uintptr, uintptr) float64
	native.Register(&fn, fp)
	out := fn(e.self, obj, mid, 0)
	if err := e.ex(ids); err != nil {
		return 0, err
	}
	return out, nil
}
