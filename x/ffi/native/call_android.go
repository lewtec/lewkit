//go:build android && !cgo && arm64

package native

import (
	"fmt"
	"reflect"
	"unsafe"
)

// pthread_create, so a C call is not the 16KB g0 of a non-cgo M.
// That stack is also the creator's TLS: clone does not pass CLONE_SETTLS.
// Register runs the call on the pthread from startCThread. A thread
// that already is that pthread stays on its own stack. RegisterLocal
// stays on the caller: JNI local refs belong to that thread, and
// moving them onto the worker is a CheckJNI abort.
//
//go:cgo_import_dynamic libc_pthread_create pthread_create "libc.so"
//go:cgo_import_dynamic _ _ "libc.so"

// callWords is the most integer words one android call passes.
// Eight travel in R0–R7. The rest are stacked, up to eight more.
const callWords = 16

// Register binds fnptr to a C function at addr.
// arm64 passes the first eight integer arguments in R0–R7, the rest
// on the stack, and returns an integer in R0 or a float in V0.
// A C string is a *byte, built by the caller with CString.
// Float arguments are rejected. The call runs on the worker pthread.
func Register(fnptr any, addr uintptr) {
	register(fnptr, addr, false)
}

// RegisterLocal binds fnptr like Register, but the call stays on this
// thread. JNI and ANativeWindow_fromSurface need the caller's JNIEnv.
// The call runs on that thread's g0 stack. A Java callback's g0 is the
// pthread stack; the goroutine stack under the callback is a few kilobytes.
func RegisterLocal(fnptr any, addr uintptr) {
	register(fnptr, addr, true)
}

func register(fnptr any, addr uintptr, local bool) {
	if fnptr == nil {
		panic("native: nil function pointer")
	}
	if addr == 0 {
		panic("native: nil symbol")
	}
	dst := reflect.ValueOf(fnptr)
	if dst.Kind() != reflect.Pointer || dst.Elem().Kind() != reflect.Func {
		panic("native: Register expects a pointer to a function")
	}
	fn := dst.Elem()
	typ := fn.Type()
	if typ.NumOut() > 1 {
		panic("native: function can only return zero or one value")
	}
	if typ.NumIn() > callWords {
		panic("native: android call supports at most 16 integer arguments")
	}
	for i := 0; i < typ.NumIn(); i++ {
		if !wordKind(typ.In(i).Kind()) {
			panic(fmt.Sprintf("native: argument %d is not an integer word", i))
		}
	}
	fn.Set(reflect.MakeFunc(typ, func(args []reflect.Value) []reflect.Value {
		var a [callWords]uintptr
		for i, arg := range args {
			word, ok := asWord(arg)
			if !ok {
				panic(fmt.Sprintf("native: argument %d is not an integer word", i))
			}
			a[i] = word
		}
		var r uintptr
		var f32 float32
		var f64 float64
		if local {
			r, f32, f64 = callHere(addr, &a[0], uintptr(len(args)))
		} else {
			r, f32, f64 = callC(addr, &a[0], uintptr(len(args)))
		}
		if typ.NumOut() == 0 {
			return nil
		}
		return []reflect.Value{fromWord(typ.Out(0), r, f32, f64)}
	}))
}

func wordKind(k reflect.Kind) bool {
	switch k {
	case reflect.Uintptr, reflect.UnsafePointer, reflect.Pointer,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Bool:
		return true
	default:
		return false
	}
}

func asWord(v reflect.Value) (uintptr, bool) {
	switch v.Kind() {
	case reflect.Uintptr:
		return uintptr(v.Uint()), true
	case reflect.UnsafePointer:
		return uintptr(v.Pointer()), true
	case reflect.Pointer:
		return uintptr(v.Pointer()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return uintptr(v.Uint()), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return uintptr(v.Int()), true
	case reflect.Bool:
		if v.Bool() {
			return 1, true
		}
		return 0, true
	default:
		return 0, false
	}
}

func fromWord(typ reflect.Type, r uintptr, f32 float32, f64 float64) reflect.Value {
	switch typ.Kind() {
	case reflect.Float32:
		return reflect.ValueOf(f32).Convert(typ)
	case reflect.Float64:
		return reflect.ValueOf(f64).Convert(typ)
	case reflect.Uintptr:
		return reflect.ValueOf(r).Convert(typ)
	case reflect.UnsafePointer:
		return reflect.ValueOf(unsafe.Pointer(r)).Convert(typ)
	case reflect.Pointer:
		return reflect.NewAt(typ.Elem(), unsafe.Pointer(r))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return reflect.ValueOf(uint64(r)).Convert(typ)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflect.ValueOf(int64(r)).Convert(typ)
	case reflect.Bool:
		return reflect.ValueOf(r != 0).Convert(typ)
	default:
		panic("native: unsupported result type " + typ.String())
	}
}

// entersyscall records this frame, then the C function runs.
// The worker pthread runs it, unless this thread already is that
// pthread. exitsyscall pairs with a reentrant cgocallback. Neither
// call can grow the stack: argv's words may be pointers. The caller
// keeps those pointers live across this call.
//
//go:nosplit
func callC(fn uintptr, argv *uintptr, argc uintptr) (r uintptr, f32 float32, f64 float64) {
	runtimeEntersyscall()
	r, f32, f64 = callCOnG0(fn, uintptr(unsafe.Pointer(argv)), argc)
	runtimeExitsyscall()
	return
}

// callHere is RegisterLocal. The C function runs on this stack.
//
//go:nosplit
func callHere(fn uintptr, argv *uintptr, argc uintptr) (r uintptr, f32 float32, f64 float64) {
	runtimeEntersyscall()
	r, f32, f64 = callHereOnG0(fn, uintptr(unsafe.Pointer(argv)), argc)
	runtimeExitsyscall()
	return
}

// startCThread starts the pthread Register uses when the caller does
// not own its TLS. Java_lewkit_Hook_call starts it on the ART thread
// after the library load. JNI_OnLoad cannot: pthread_create returns
// EAGAIN while the linker is inside that load. Zero means the thread
// is running. A non-cgo M must not call this: clone shares the
// creator's TLS, and pthread_create does not return.
func startCThread() int32

//go:nosplit
func callCOnG0(fn, argv, argc uintptr) (r uintptr, f32 float32, f64 float64)

//go:nosplit
func callHereOnG0(fn, argv, argc uintptr) (r uintptr, f32 float32, f64 float64)

//go:linkname runtimeEntersyscall runtime.entersyscall
func runtimeEntersyscall()

//go:linkname runtimeExitsyscall runtime.exitsyscall
func runtimeExitsyscall()
