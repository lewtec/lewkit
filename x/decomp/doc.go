// Package decomp is an in-process program database over Capstone.
//
// A [Project] is the session. [Project.Add] loads an ELF, PE, or Mach-O
// artifact and lifts its executable sections through
// [github.com/lewtec/lewkit/x/disasm], which keeps Capstone inside this
// process. The artifact records functions, basic blocks, references, and
// strings. [Open] stores that database in a sqlite file and reloads the
// saved lift, including names and address notes. [New] keeps the same
// database in memory. [Artifact.SetName] and [Artifact.SetNote] write
// those clues back when the project has a file. [Project.Attach]
// sets the [Model] that writes a decompilation.
// [Harness] runs the Grok harness, which uses the signed-in subscription.
// [Chat] is a direct completion for a caller who passes an endpoint.
// [Artifact.Match] keeps that decompilation only when every block copies
// the lifted control flow and the source names the function, each direct
// callee, and each referenced string.
package decomp
