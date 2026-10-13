// cmd/go skips files whose names start with underscore. x/build/androidtoolexec
// passes this file to the compiler for android/arm64 with cgo off. The
// compiler accepts cgo_export_dynamic only in a file named _cgo_*.
// The wrapper hashes this file into the compiler version so a cached
// object from a build that did not see it is not reused.
// The assembly entries are android && !cgo && arm64 && appmode.
// Keep these names in step with host_nocgo_android_arm64.s.

package android

//go:cgo_export_dynamic JNI_OnLoad JNI_OnLoad
//go:cgo_export_dynamic Java_lewkit_Hook_call Java_lewkit_Hook_call
//go:cgo_export_dynamic Java_lewkit_GoProxy_nativeInvoke Java_lewkit_GoProxy_nativeInvoke
