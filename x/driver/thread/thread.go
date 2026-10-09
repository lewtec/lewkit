// Package thread is the UI thread for this process.
//
// The JNI backend posts work onto Android's main looper when a Java VM is
// already running. Otherwise the OS backend locks the process thread and
// runs Loop. Win32 uses the same process thread and Loop drains that
// thread's message queue between jobs. Linux Loop drains the GTK queue when
// this thread owns it. AppKit and Win32 require that thread.
package thread

import "context"

// OSThread is the current OS thread id. Zero means the id is unavailable.
func OSThread() uint64 { return osThread() }

// Driver runs functions on the UI thread.
type Driver interface {
	Bind()
	Bound() bool
	On() bool
	OnIdle(fn func())
	Loop(ctx context.Context)
	Do(fn func())
	Go(fn func())
	Enqueue(fn func())
}
