// cmd/go skips files whose names start with underscore. x/build/androidtoolexec
// passes this file to the compiler for android/arm64 with cgo off. The
// compiler accepts cgo_export_dynamic only in a file named _cgo_*.
// The library build enables the assembly entries with -tags androidnocgo.
// Keep these names in step with host_nocgo_android_arm64.s.

package entry

//go:cgo_export_dynamic JNI_OnLoad JNI_OnLoad
//go:cgo_export_dynamic Java_lewkit_Hook_call Java_lewkit_Hook_call
//go:cgo_export_dynamic Java_lewkit_GoProxy_nativeInvoke Java_lewkit_GoProxy_nativeInvoke
