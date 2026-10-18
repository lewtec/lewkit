// Package gotool drives Go's compiler and linker as a library.
//
// The source is vendored with modot (file place of github:golang/go at
// go1.27.0) into _upstream. That pin keeps upstream import paths, and
// cmd/compile/internal and cmd/link/internal have different parents, so one
// package still cannot import both. rewrite drops the internal element and
// retargets the standard-library internal packages those trees import onto
// the same tree. The child process calls gc.MainArgs and ld.Run.
//
// Compile and Link are functions of one package or one image. Each call
// starts a process that owns the compiler or linker tables and exits with
// them, so the caller does not reset those tables and does not take a lock.
// x/workflow runs one step per package. Independent packages run together,
// and the link step waits on them. BuildModule keeps the caller session.
// Each package and the link are CPU tasks under the current task, so a
// progress view already on ctx shows those rows. The caller is Control:
// a CPU task that waits on those compiles holds a pool slot.
// The child uses a private flag set and does not read os.Args or
// flag.CommandLine. Exits are patched into panics, which that process
// recovers. -c=1 keeps the compile to one worker; the workflow pool is the
// parallelism. lewkit release go loads the module in the parent and calls
// Build. Flags may follow the package arguments. -ldflags are Image.Ldflags.
// -pkgdir supplies the standard library and any package that contains
// assembly; cmd/asm is not vendored. //go:embed configuration is -embedcfg.
// An Object is one package archive. A Target is the GOOS, GOARCH, link
// mode, runtime archives, and extra linker flags for one image. Linux,
// a shared object, and tamago are targets. ImportCfg rejects an object
// whose GOOS or GOARCH differs. A unit that lists .s or .syso files is
// ErrNoAsm, because cmd/asm is not vendored. BuildObjects compiles the
// units and links the one named main.
// BuildModule compiles one module directory and links its main package.
// The child process receives that GOOS and GOARCH. GOARCH other than amd64
// or arm64 is an error. A target that must link externally is an error.
// iOS links internally. Host go still refuses ios/arm64 without cgo, so the
// iOS standard-library list does not load package main.
// Android and any other default-PIE target links as an internal PIE, and
// Module.Library forces that PIE, including on Linux. The compiler passes
// -shared for that PIE except on Windows. go list -export receives
// -buildmode=pie for the same link. Standard-library and assembly archives
// come from that list, or from Pkgdir when that is set. A main package
// that contains assembly uses the list archive when the list recorded one.
// Toolexec is that list's -toolexec when the basename starts with _cgo_
// and ends with _export.go. lewkit release build, run, and build-all
// use this compiler unless --go host selects the go command.
// Compile and Link still take the files and the importcfg the caller prepared.
// SharedLibrary compiles one main package from source into a shared object.
// RunSource compiles and runs a main package from source. Linux holds
// that source in a memfd. Darwin and Windows use a temporary file, so
// the package builds on those hosts.
// CompileSource compiles that source in this process. Bind returns one of
// its functions, and the caller invokes it here. Host function addresses
// come from the process pclntab. Linux maps the code with mmap, Darwin
// with MAP_JIT, and Windows with VirtualAlloc. examples/toolchain share
// calls SharedLibrary. examples/toolchain jit calls CompileSource and Bind.
// mise test:host runs those tests, the FFI packages, and purego.
//
// -buildmode=c-shared is still rejected unless linkmode=external, which
// needs cgo. Calls into C go through x/toolchain/purego. The cgo-free
// shared object compiles with -shared and //go:cgo_export_dynamic, links
// as PIE with -linkmode=internal, then clears DF_1_PIE and drops PT_INTERP.
// Bare metal passes x/toolchain/tamago link flags through Image.Ldflags.
//
//	go test -count=1 -timeout 20m ./x/toolchain/go
package gotool
