// Package clib is the runtime for a shared library built without cgo.
//
// lewkit generate clib reads a C header and writes one stub variable per
// function. The package init assigns those variables. A C caller enters
// the matching export. The first call boots the Go runtime on a new
// thread. The caller then writes the arguments to a pipe and waits until
// a goroutine runs the stub and stores the result.
//
// The C thread does not enter the Go scheduler. Its host thread pointer
// stays in place, and the new thread is the one that runs Go.
// Assign the stubs from init and do not call back into the library from
// there: init runs before the runtime reports itself ready.
//
// linux and android share a path on amd64, arm64, 386, and arm.
// amd64 is SysV. arm64 is AAPCS64. arm is AAPCS hard-float. 386 is cdecl
// and returns a float in ST(0). Darwin amd64 is SysV and Darwin arm64 is
// AAPCS64. Windows amd64 uses one slot per argument, Windows arm64 is
// AAPCS64, and Windows 386 is cdecl with 8-byte alignment. The first call
// enters the runtime through _rt0_*_lib. Linux parks on a futex. Windows
// and Darwin yield until the result is stored. The entry assembly is
// built only with the clibexport tag, and the link passes
// -checklinkname=0 because the entry refers to runtime startup symbols.
// The internal PIE link is linux/amd64, linux/arm64, android/arm64,
// windows/amd64, windows/arm64, windows/386, darwin/amd64, and
// darwin/arm64. The build tags match that set. The darwin files exclude
// ios. arm uses GOARM=7.
package clib
