//go:build android && cgo

package jni

/*
#include <jni.h>
*/
import "C"
import "unsafe"

//export Java_lewkit_GoProxy_nativeInvoke
func Java_lewkit_GoProxy_nativeInvoke(envPtr *C.JNIEnv, _ C.jclass, id C.jlong, name C.jstring, args C.jobjectArray) C.jobject {
	p := InvokeProxy(
		uintptr(unsafe.Pointer(envPtr)),
		int64(id),
		uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(args)),
	)
	return C.jobject(unsafe.Pointer(p))
}
