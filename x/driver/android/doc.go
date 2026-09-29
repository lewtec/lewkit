// Package android is the application context behind Android drivers.
//
// Context reads lewkit.Host's application context through x/ffi/jni.
// The caller releases that reference. Ref, Int, Text, and Bool turn a
// call result into a Go value.
package android
