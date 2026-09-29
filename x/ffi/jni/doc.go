// Package jni calls public Java methods and fields from Go on Android.
//
// Bind has to run on the Java thread that loaded the library. It keeps that
// thread's ClassLoader and that thread's JNIEnv. SetRunner then runs
// CallStatic, New, Class, field reads, Proxy, and Release on that same thread.
// FindClass on a goroutine attached later does not see the app loader, so
// those calls load classes through the saved loader.
//
// Arguments may be a string, a boolean, an integer, a float, nil, or a *Ref.
// CallStatic, Ref.Call, and New choose one public method or constructor by
// name and by which parameter types accept those arguments. A tie is an error.
// A returned string or boxed number becomes a Go value. Any other object comes
// back as a *Ref. The caller releases a *Ref once.
//
// StaticField and Ref.Field read a public field. A primitive comes back as a
// Go value. An object comes back as a *Ref.
//
// Proxy builds one java.lang.reflect.Proxy for an interface. The callback runs
// on the Java thread that invoked the method. *Ref arguments belong to the
// callback. Release drops the proxy and the callback. Proxy uses lewkit.GoProxy
// and one native dispatcher, so a call does not add a JNI export.
//
// Lookup walks the class's public methods on each call.
// A Kotlin method is static only when it is @JvmStatic.
package jni
