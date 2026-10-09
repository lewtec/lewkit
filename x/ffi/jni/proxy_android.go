//go:build android

package jni

import (
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
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

// InvokeProxy runs a Go proxy method for Java_lewkit_GoProxy_nativeInvoke.
// env is the calling thread's JNIEnv. name is a jstring and args is a jobjectArray.
func InvokeProxy(env uintptr, id int64, name, args uintptr) (out uintptr) {
	var owned []any
	handed := false
	defer func() {
		if rec := recover(); rec != nil {
			if !handed {
				releaseValues(owned)
			}
			slog.Error("jni proxy", "panic", rec)
			out = 0
		}
	}()
	e := openEnv(env)
	if e.tab == 0 {
		return 0
	}
	vm := bound.Load()
	if vm == nil {
		return 0
	}
	method, err := e.readString(name)
	if err != nil {
		return 0
	}
	owned, err = e.values(vm, args)
	if err != nil {
		slog.Error("jni proxy", "err", err)
		return 0
	}
	fn := proxyFunc(id)
	if fn == nil {
		releaseValues(owned)
		return 0
	}
	handed = true
	v, err := fn(method, owned)
	if err != nil {
		slog.Error("jni proxy", "method", method, "err", err)
		return 0
	}
	if v == nil {
		return 0
	}
	val, err := classify(v)
	if err != nil {
		slog.Error("jni proxy", "method", method, "err", err)
		return 0
	}
	p, err := e.box(vm, val)
	if err != nil {
		slog.Error("jni proxy", "method", method, "err", err)
		return 0
	}
	return p
}
