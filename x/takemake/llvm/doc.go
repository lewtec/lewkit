// Package llvm is one LLVM IR module.
//
// [New] returns an empty [File]. Methods return the receiver and store
// the child: named types, globals, and functions. A function keeps its
// basic blocks. [Take] parses LLVM IR assembly. [Make] prints that same
// assembly. Attribute groups, comdats, and metadata stay on the file.
// Bitcode is outside this file. A Go translation of the instructions is
// not this package.
// The contract is in DESIGN.md beside the parent directory.
package llvm
