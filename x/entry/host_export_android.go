//go:build android && cgo

package entry

/*
#include <jni.h>
*/
import "C"
import (
	"unsafe"

	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
)

//export JNI_OnLoad
func JNI_OnLoad(vm *C.JavaVM, _ unsafe.Pointer) C.jint {
	// The VM calls this on the thread that loaded the library, after the
	// linker lock is released. init has already finished. Keeping the VM
	// here lets later threads ask about it without dlopen.
	androidffi.NoteVM(uintptr(unsafe.Pointer(vm)))
	androidffi.NoteLoader()
	return C.JNI_VERSION_1_6
}

//export Java_lewkit_Hook_call
func Java_lewkit_Hook_call(env *C.JNIEnv, _ C.jclass, name C.jstring, args C.jobjectArray) C.jlong {
	return C.jlong(hookCall(
		uintptr(unsafe.Pointer(env)),
		uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(args)),
	))
}
