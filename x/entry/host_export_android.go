//go:build android && cgo

package entry

/*
#cgo LDFLAGS: -landroid
#include <android/native_window_jni.h>
#include <jni.h>
#include <stdlib.h>

void lewkit_note_vm(JavaVM *vm);
void lewkit_bind_host(JNIEnv *env);
JNIEnv *lewkit_attach(void);
char *lewkit_go_string(JNIEnv *env, jstring s);
*/
import "C"
import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sync/atomic"
	"unsafe"

	"github.com/lewtec/lewkit/x/driver/thread"
	"github.com/lewtec/lewkit/x/ffi/jni"
	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
)

//export JNI_OnLoad
func JNI_OnLoad(vm *C.JavaVM, _ unsafe.Pointer) C.jint {
	// The VM calls this on the thread that loaded the library, after the
	// linker lock is released. init has already finished. Keeping the VM
	// here lets later threads ask about it without dlopen.
	C.lewkit_note_vm(vm)
	androidffi.NoteVM(uintptr(unsafe.Pointer(vm)))
	androidffi.NoteLoader()
	return C.JNI_VERSION_1_6
}

type surfaceBox struct {
	ptr    uintptr
	width  int
	height int
}

var surfaceCh = make(chan surfaceBox, 1)

// ShowSurface tells the host the Go app is a native window, not a page.
func ShowSurface() {
	if _, err := jni.CallStatic("lewkit.Host", "openSurface"); err != nil {
		slog.Error("android surface", "err", err)
	}
}

// RequestSurface asks the Android host for a native window and waits for it.
// The size is the surface the activity reported, which is the GUI window size.
func RequestSurface(ctx context.Context) (uintptr, int, int, error) {
	if _, err := jni.CallStatic("lewkit.Host", "openSurface"); err != nil {
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
	if _, err := jni.CallStatic("lewkit.Host", "ready", url); err != nil {
		slog.Error("android ready", "err", err)
	}
}

// NotifyFail tells the Android host why startup stopped.
func NotifyFail(message string) {
	if _, err := jni.CallStatic("lewkit.Host", "fail", message); err != nil {
		slog.Error("android fail", "err", err)
	}
}

//export Java_lewkit_Host_start
func Java_lewkit_Host_start(env *C.JNIEnv, _ C.jclass, file C.jstring) {
	// The JNIEnv passed in is only valid on this thread.
	runtime.LockOSThread()
	C.lewkit_bind_host(env)
	jni.SetCurrentEnv(func() uintptr {
		env := C.lewkit_attach()
		if env == nil {
			return 0
		}
		return uintptr(unsafe.Pointer(env))
	})
	// entry.Run pumps the queue on this goroutine. Java calls from request
	// goroutines join that queue instead of attaching a new JNI thread.
	jni.SetRunner(func(fn func()) { thread.Do(fn) })
	if err := jni.Bind(uintptr(unsafe.Pointer(env)), "lewkit/Host"); err != nil {
		slog.Error("jni bind", "err", err)
		return
	}
	raw := C.lewkit_go_string(env, file)
	if raw == nil {
		return
	}
	path := C.GoString(raw)
	C.free(unsafe.Pointer(raw))
	_ = os.Setenv("ELETROCROMO_NO_UI", "1")
	_ = os.Setenv("ELETROCROMO_READY_FILE", path)
	// Bind before the app runs. A failure before entry.Run has no loop yet,
	// and reporting it must run on this thread instead of waiting for that loop.
	thread.Bind()
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

//export Java_lewkit_Host_surfaceLost
func Java_lewkit_Host_surfaceLost(env *C.JNIEnv, class C.jclass) {
	DeliverSurfaceLost()
}

//export Java_lewkit_Host_pointer
func Java_lewkit_Host_pointer(env *C.JNIEnv, class C.jclass, x, y, action C.jint) {
	DeliverPointer(int(x), int(y), int(action))
}

//export Java_lewkit_Host_resize
func Java_lewkit_Host_resize(env *C.JNIEnv, class C.jclass, width, height C.jint) {
	DeliverResize(int(width), int(height))
}

//export Java_lewkit_Host_obscure
func Java_lewkit_Host_obscure(env *C.JNIEnv, class C.jclass, left, top, right, bottom, width, height C.jint) {
	DeliverInsets(int(left), int(top), int(right), int(bottom), int(width), int(height))
}
