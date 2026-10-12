// Package decomp is an in-process program database over Capstone.
//
// A [Project] is the session. [Project.Add] loads an ELF, PE, or Mach-O
// artifact and lifts its executable sections through
// [github.com/lewtec/lewkit/x/disasm], which keeps Capstone inside this
// process. The artifact records functions, basic blocks, references, and
// strings. [Project.Attach] sets the [Model] that writes a decompilation.
// [Artifact.Match] keeps that decompilation only when every block copies
// the lifted control flow and the source names the function, each direct
// callee, and each referenced string.
package decomp
