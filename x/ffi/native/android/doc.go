// Package android binds libandroid and JNI_GetCreatedJavaVMs.
// JNI_OnLoad notes the JavaVM and the loader thread. JavaVMs and OnLooper
// use that note and do not dlopen while it is set.
package android
