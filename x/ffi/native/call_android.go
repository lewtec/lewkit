//go:build android && !cgo && arm64

package native

import (
	"fmt"
	"reflect"
	"runtime"
	"unsafe"
)

// callWords is the most integer words one android call passes.
// Eight travel in R0–R7. The rest are stacked, up to eight more.
const callWords = 16

// Register binds fnptr to a C function at addr.
// arm64 passes the first eight integer arguments in R0–R7, the rest
// on the stack, and returns an integer in R0 or a float in V0.
// A string argument is a NUL-terminated C string, the same as purego.
// Float arguments are rejected.
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
		var keep [][]byte
		for i, arg := range args {
			word, buf, ok := asWord(arg)
			if !ok {
				panic(fmt.Sprintf("native: argument %d is not an integer word", i))
			}
			a[i] = word
			if buf != nil {
				keep = append(keep, buf)
			}
		}
		r, f32, f64 := callC(addr, &a[0], uintptr(len(args)))
		runtime.KeepAlive(keep)
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
		reflect.Bool, reflect.String:
		return true
	default:
		return false
	}
}

func asWord(v reflect.Value) (uintptr, []byte, bool) {
	switch v.Kind() {
	case reflect.String:
		// C sees a pointer to the bytes, plus the trailing NUL.
		buf := append([]byte(v.String()), 0)
		return uintptr(unsafe.Pointer(unsafe.SliceData(buf))), buf, true
	case reflect.Uintptr:
		return uintptr(v.Uint()), nil, true
	case reflect.UnsafePointer:
		return uintptr(v.Pointer()), nil, true
	case reflect.Pointer:
		return uintptr(v.Pointer()), nil, true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return uintptr(v.Uint()), nil, true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return uintptr(v.Int()), nil, true
	case reflect.Bool:
		if v.Bool() {
			return 1, nil, true
		}
		return 0, nil, true
	default:
		return 0, nil, false
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
// the stack: argv's words may be pointers. The caller keeps any string
// buffers live across this call.
//
//go:nosplit
func callC(fn uintptr, argv *uintptr, argc uintptr) (r uintptr, f32 float32, f64 float64) {
	runtimeEntersyscall()
	r, f32, f64 = callCOnG0(fn, uintptr(unsafe.Pointer(argv)), argc)
	runtimeExitsyscall()
	return
}

//go:nosplit
func callCOnG0(fn, argv, argc uintptr) (r uintptr, f32 float32, f64 float64)

//go:linkname runtimeEntersyscall runtime.entersyscall
func runtimeEntersyscall()

//go:linkname runtimeExitsyscall runtime.exitsyscall
func runtimeExitsyscall()
