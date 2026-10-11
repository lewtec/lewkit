// Package clib is the runtime for a shared library built without cgo.
//
// lewkit generate clib reads a C header and writes one stub variable per
// function. The package init assigns those variables. A C caller enters
// the matching export. The first call boots the Go runtime on a new
// thread. The caller then writes the arguments to a pipe and parks on a
// futex until a goroutine runs the stub and stores the result.
//
// The C thread does not enter the Go scheduler. Its libc thread pointer
// stays in place, and the cloned thread is the one that runs Go.
// Assign the stubs from init and do not call back into the library from
// there: init runs before the runtime reports itself ready.
//
// linux and android share this path on amd64, arm64, 386, and arm.
// amd64 is SysV. arm64 is AAPCS64. arm is AAPCS hard-float. 386 is cdecl
// and returns a float in ST(0). The entry assembly is built only with
// the clibexport tag, and the link passes -checklinkname=0 because the
// entry refers to runtime startup symbols. The internal PIE link is
// linux/amd64, linux/arm64, and android/arm64. arm uses GOARM=7.
package clib
