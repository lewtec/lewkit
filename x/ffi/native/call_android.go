//go:build android && !cgo && arm64

package native

import (
	"fmt"
	"reflect"
	"unsafe"
)

// Register binds fnptr to a C function at addr.
// arm64 passes the first eight integer arguments in R0–R7 and returns
// an integer in R0 or a float in V0, which is the Go ABI as well.
// Float arguments and calls with more than eight words are rejected.
func Register(fnptr any, addr uintptr) {
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
	if typ.NumIn() > 8 {
		panic("native: android call supports at most 8 integer arguments")
	}
	for i := 0; i < typ.NumIn(); i++ {
		if _, ok := asWord(reflect.Zero(typ.In(i))); !ok {
			panic(fmt.Sprintf("native: argument %d is not an integer word", i))
		}
	}
	fn.Set(reflect.MakeFunc(typ, func(args []reflect.Value) []reflect.Value {
		var a [8]uintptr
		for i, arg := range args {
			word, ok := asWord(arg)
			if !ok {
				panic(fmt.Sprintf("native: argument %d is not an integer word", i))
			}
			a[i] = word
		}
		r, f32, f64 := callC(addr, a[0], a[1], a[2], a[3], a[4], a[5], a[6], a[7])
		if typ.NumOut() == 0 {
			return nil
		}
		return []reflect.Value{fromWord(typ.Out(0), r, f32, f64)}
	}))
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

// entersyscall records this frame, then the C function runs on m.g0.
// exitsyscall pairs with a reentrant cgocallback. Neither call can grow
// the stack: the uintptr arguments may be pointers.
//
//go:nosplit
func callC(fn, a0, a1, a2, a3, a4, a5, a6, a7 uintptr) (r uintptr, f32 float32, f64 float64) {
	runtimeEntersyscall()
	r, f32, f64 = callCOnG0(fn, a0, a1, a2, a3, a4, a5, a6, a7)
	runtimeExitsyscall()
	return
}

//go:nosplit
func callCOnG0(fn, a0, a1, a2, a3, a4, a5, a6, a7 uintptr) (r uintptr, f32 float32, f64 float64)

//go:linkname runtimeEntersyscall runtime.entersyscall
func runtimeEntersyscall()

//go:linkname runtimeExitsyscall runtime.exitsyscall
func runtimeExitsyscall()
