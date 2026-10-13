// Package toolchain is the Go toolchain.
//
// x/toolchain/go is the compiler and linker, linked into lewkit by default.
// Each package is one x/workflow step, and that step runs in its own process.
// lewkit release go is that compiler's command:
// build flags may follow the package arguments, and -pkgdir supplies the
// standard library and packages that contain assembly.
//
// x/toolchain/purego is the stand-in for cgo. A program calls C with
// Open and Register. Unix opens the library with Dlopen. Windows loads
// a DLL. -buildmode=c-shared still needs a C compiler.
// x/toolchain/c reads a C header and returns the Go package that
// represents those declarations. That package is one compile unit.
//
// x/toolchain/tamago is the bare-metal target. LinkFlags set the entry
// and the load address, and goos is the runtime/goos overlay. A tamago
// link uses those flags as its target. The runtime archives are objects
// of GOOS tamago, and a linux archive does not join that link.
package toolchain
