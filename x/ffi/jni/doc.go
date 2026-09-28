// Package jni calls public Java methods from Go on Android.
//
// Bind has to run on the Java thread that loaded the library. It keeps that
// thread's ClassLoader. FindClass on a goroutine attached later does not see
// the app loader, so CallStatic and New load classes through the saved loader.
//
// Arguments may be a string, a boolean, an integer, a float, nil, or a *Ref.
// CallStatic, Ref.Call, and New choose one public method or constructor by
// name and by which parameter types accept those arguments. A tie is an error.
// A returned string or boxed number becomes a Go value. Any other object comes
// back as a *Ref. The caller releases a *Ref once.
//
// Lookup walks the class's public methods on each call.
// A Kotlin method is static only when it is @JvmStatic.
package jni
