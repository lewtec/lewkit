//go:build android && cgo

package jni

/*
#include <jni.h>
*/
import "C"
import (
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"unsafe"
)

var errNullProxy = errors.New("java proxy is null")

var (
	proxyMu    sync.Mutex
	proxyFn    = map[int64]func(string, []any) (any, error){}
	proxyByPtr = map[uintptr]int64{}
	proxySeq   atomic.Int64
)

func makeProxy(iface string, fn func(string, []any) (any, error)) (*Ref, error) {
	cls, err := classObject(iface)
	if err != nil {
		return nil, err
	}
	defer cls.Release()
	id := proxySeq.Add(1)
	proxyMu.Lock()
	proxyFn[id] = fn
	proxyMu.Unlock()
	got, err := callStatic("lewkit.GoProxy", "create", cls, id)
	if err != nil {
		dropProxyID(id)
		return nil, err
	}
	ref, ok := got.(*Ref)
	if !ok || ref == nil {
		dropProxyID(id)
		return nil, errNullProxy
	}
	proxyMu.Lock()
	proxyByPtr[ref.ptr] = id
	proxyMu.Unlock()
	return ref, nil
}

func proxyFunc(id int64) func(string, []any) (any, error) {
	proxyMu.Lock()
	defer proxyMu.Unlock()
	return proxyFn[id]
}

func dropProxyID(id int64) {
	proxyMu.Lock()
	delete(proxyFn, id)
	for ptr, got := range proxyByPtr {
		if got == id {
			delete(proxyByPtr, ptr)
		}
	}
	proxyMu.Unlock()
}

func dropProxy(ptr uintptr) {
	if ptr == 0 {
		return
	}
	proxyMu.Lock()
	id, ok := proxyByPtr[ptr]
	if ok {
		delete(proxyByPtr, ptr)
		delete(proxyFn, id)
	}
	proxyMu.Unlock()
}

func (e env) values(vm *vm, arr uintptr) ([]any, error) {
	if arr == 0 {
		return nil, nil
	}
	n, err := e.arrayLen(arr, vm.errIDs)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, n)
	for i := int32(0); i < n; i++ {
		el, err := e.element(arr, i, vm.errIDs)
		if err != nil {
			releaseValues(out)
			return nil, err
		}
		v, err := e.unbox(vm, el, "", false)
		if err != nil {
			releaseValues(out)
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func releaseValues(args []any) {
	for _, arg := range args {
		if ref, ok := arg.(*Ref); ok {
			ref.Release()
		}
	}
}

func jobject(p uintptr) C.jobject {
	if p == 0 {
		var zero C.jobject
		return zero
	}
	return C.jobject(unsafe.Pointer(p))
}

//export Java_lewkit_GoProxy_nativeInvoke
func Java_lewkit_GoProxy_nativeInvoke(envPtr *C.JNIEnv, _ C.jclass, id C.jlong, name C.jstring, args C.jobjectArray) (out C.jobject) {
	var owned []any
	handed := false
	defer func() {
		if rec := recover(); rec != nil {
			if !handed {
				releaseValues(owned)
			}
			slog.Error("jni proxy", "panic", rec)
			out = jobject(0)
		}
	}()
	e := openEnv(uintptr(unsafe.Pointer(envPtr)))
	if e.tab == 0 {
		return jobject(0)
	}
	vm := bound.Load()
	if vm == nil {
		return jobject(0)
	}
	method, err := e.readString(uintptr(unsafe.Pointer(name)))
	if err != nil {
		return jobject(0)
	}
	owned, err = e.values(vm, uintptr(unsafe.Pointer(args)))
	if err != nil {
		slog.Error("jni proxy", "err", err)
		return jobject(0)
	}
	fn := proxyFunc(int64(id))
	if fn == nil {
		releaseValues(owned)
		return jobject(0)
	}
	handed = true
	v, err := fn(method, owned)
	if err != nil {
		slog.Error("jni proxy", "method", method, "err", err)
		return jobject(0)
	}
	if v == nil {
		return jobject(0)
	}
	val, err := classify(v)
	if err != nil {
		slog.Error("jni proxy", "method", method, "err", err)
		return jobject(0)
	}
	p, err := e.box(vm, val)
	if err != nil {
		slog.Error("jni proxy", "method", method, "err", err)
		return jobject(0)
	}
	return jobject(p)
}
