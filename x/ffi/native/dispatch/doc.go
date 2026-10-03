// Package dispatch runs a function on the platform main queue.
//
// On iOS the UIKit main thread is not the Go UI thread, so [OnMain] uses
// dispatch_sync. On other systems it runs the function on the caller.
package dispatch
