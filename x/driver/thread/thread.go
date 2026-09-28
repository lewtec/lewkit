// Package thread is the UI thread for this process.
//
// The JNI backend posts work onto Android's main looper when a Java VM is
// already running. Otherwise the OS backend locks the process thread and
// runs Loop, which is what AppKit and Win32 require.
package thread

import "context"

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
