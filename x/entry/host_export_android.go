//go:build android && cgo

package entry

/*
#cgo LDFLAGS: -landroid
#include <android/native_window_jni.h>
#include <jni.h>
#include <stdlib.h>

void lewkit_bind_host(JNIEnv *env);
jclass lewkit_host_class(void);
JNIEnv *lewkit_attach(void);
char *lewkit_go_string(JNIEnv *env, jstring s);
*/
import "C"
import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"unsafe"

	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
)

type surfaceBox struct {
	ptr    uintptr
	width  int
	height int
}

var surfaceCh = make(chan surfaceBox, 1)

// RequestSurface asks the Android host for a native window and waits for it.
// The size is the surface the activity reported, which is the GUI window size.
func RequestSurface(ctx context.Context) (uintptr, int, int, error) {
	if err := androidffi.StaticVoid("openSurface", "()V"); err != nil {
		return 0, 0, 0, err
	}
	select {
	case box := <-surfaceCh:
		if box.ptr == 0 || box.width < 1 || box.height < 1 {
			return 0, 0, 0, errNoSurface
		}
		return box.ptr, box.width, box.height, nil
	case <-ctx.Done():
		return 0, 0, 0, ctx.Err()
	}
}

var errNoSurface = errors.New("android surface closed")

// NotifyReady tells the Android host that the web window can open this URL.
func NotifyReady(url string) {
	_ = androidffi.StaticVoid("ready", "(Ljava/lang/String;)V", url)
}

// NotifyFail tells the Android host why startup stopped.
func NotifyFail(message string) {
	_ = androidffi.StaticVoid("fail", "(Ljava/lang/String;)V", message)
}

//export Java_lewkit_Host_start
func Java_lewkit_Host_start(env *C.JNIEnv, _ C.jclass, file C.jstring) {
	androidffi.SetCurrentEnv(func() uintptr {
		env := C.lewkit_attach()
		if env == nil {
			return 0
		}
		return uintptr(unsafe.Pointer(env))
	})
	C.lewkit_bind_host(env)
	cls := C.lewkit_host_class()
	if cls == 0 {
		return
	}
	androidffi.SetHostClass(uintptr(cls))
	raw := C.lewkit_go_string(env, file)
	if raw == nil {
		return
	}
	path := C.GoString(raw)
	C.free(unsafe.Pointer(raw))
	_ = os.Setenv("ELETROCROMO_NO_UI", "1")
	_ = os.Setenv("ELETROCROMO_READY_FILE", path)
	defer func() {
		if recovered := recover(); recovered != nil {
			NotifyFail(fmt.Sprint(recovered))
		}
	}()
	if err := RunBound(); err != nil {
		NotifyFail(err.Error())
	}
}

//export Java_lewkit_Host_nativeWindow
func Java_lewkit_Host_nativeWindow(env *C.JNIEnv, _ C.jclass, surface C.jobject, width C.jint, height C.jint) C.jlong {
	var ptr uintptr
	if surface != 0 {
		ptr = uintptr(unsafe.Pointer(C.ANativeWindow_fromSurface(env, surface)))
		ptr &= 0x00ffffffffffffff
	}
	box := surfaceBox{ptr: ptr, width: int(width), height: int(height)}
	select {
	case surfaceCh <- box:
	default:
		select {
		case <-surfaceCh:
		default:
		}
		surfaceCh <- box
	}
	if fn, ok := surfaceFn.Load().(func(uintptr, int, int)); ok && fn != nil {
		fn(ptr, int(width), int(height))
	}
	return C.jlong(ptr)
}

var surfaceFn atomic.Value

// HandleSurface receives a replacement Android native window and its size.
func HandleSurface(fn func(ptr uintptr, width, height int)) {
	if fn == nil {
		return
	}
	surfaceFn.Store(fn)
}

var surfaceLostFn atomic.Value

// HandleSurfaceLost runs when Android destroys the current surface.
func HandleSurfaceLost(fn func()) {
	if fn == nil {
		return
	}
	surfaceLostFn.Store(fn)
}

//export Java_lewkit_Host_surfaceLost
func Java_lewkit_Host_surfaceLost(env *C.JNIEnv, class C.jclass) {
	fn, _ := surfaceLostFn.Load().(func())
	if fn == nil {
		return
	}
	fn()
}

var pointerFn atomic.Value

// HandlePointer receives Android touch samples. action is 0 down, 1 up, 2 move.
func HandlePointer(fn func(x, y, action int)) {
	if fn == nil {
		return
	}
	pointerFn.Store(fn)
}

//export Java_lewkit_Host_pointer
func Java_lewkit_Host_pointer(env *C.JNIEnv, class C.jclass, x, y, action C.jint) {
	fn, _ := pointerFn.Load().(func(int, int, int))
	if fn == nil {
		return
	}
	fn(int(x), int(y), int(action))
}

var resizeFn atomic.Value

// HandleResize receives a new Android surface size in pixels.
func HandleResize(fn func(width, height int)) {
	if fn == nil {
		return
	}
	resizeFn.Store(fn)
}

//export Java_lewkit_Host_resize
func Java_lewkit_Host_resize(env *C.JNIEnv, class C.jclass, width, height C.jint) {
	fn, _ := resizeFn.Load().(func(int, int))
	if fn == nil {
		return
	}
	fn(int(width), int(height))
}

type promptHandler func(text string, code int)

var promptFn atomic.Pointer[promptHandler]

// HandlePrompt receives one activity dialog result.
// code 0 is cancel, 1 is the submitted text, and 2 means no foreground activity.
// A nil fn clears the handler.
func HandlePrompt(fn func(text string, code int)) {
	if fn == nil {
		promptFn.Store(nil)
		return
	}
	h := promptHandler(fn)
	promptFn.Store(&h)
}

//export Java_lewkit_Host_promptResult
func Java_lewkit_Host_promptResult(env *C.JNIEnv, _ C.jclass, text C.jstring, code C.jint) {
	fn := promptFn.Load()
	if fn == nil || *fn == nil {
		return
	}
	(*fn)(hostGoString(env, text), int(code))
}

func hostGoString(env *C.JNIEnv, text C.jstring) string {
	if text == 0 {
		return ""
	}
	raw := C.lewkit_go_string(env, text)
	if raw == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(raw))
	return C.GoString(raw)
}
