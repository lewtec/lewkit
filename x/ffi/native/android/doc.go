// Package android binds libandroid and JNI_GetCreatedJavaVMs.
// JNI_OnLoad notes the JavaVM and the loader thread. JavaVMs and OnLooper
// use that note and do not dlopen while it is set.
package android

//go:generate go run ../../../../cmd/lewkit generate header --package android --bind JNI --out jni_bind.go jni.h
//go:generate go run ../../../../cmd/lewkit generate header --package android --bind Looper --out looper_bind.go looper.h
