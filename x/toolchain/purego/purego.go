// Package purego calls C without cgo.
//
// Open loads a shared library. Unix calls purego.Dlopen. Windows calls
// LoadDLL. Register is purego.RegisterLibFunc. The Go compiler in
// x/toolchain/go does not run a C compiler. A call that would have been
// import "C" goes through this package.
package purego

import "github.com/ebitengine/purego"

// Register binds fn to the C symbol name in handle. fn is a pointer to
// a function whose arguments fit in uintptr.
func Register(fn any, handle uintptr, name string) {
	purego.RegisterLibFunc(fn, handle, name)
}
