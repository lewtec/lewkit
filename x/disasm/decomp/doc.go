// Package decomp is an in-process program database over Capstone.
//
// A [Project] is the session. [Project.Add] loads an ELF, PE, or Mach-O
// artifact and lifts its executable sections through
// [github.com/lewtec/lewkit/x/disasm], which keeps Capstone inside this
// process. The artifact records functions, basic blocks, references, and
// strings. Imports name each thunk, and an export at an instruction is a
// function start. A function body is the code reachable from that start.
// [Open] stores that database in a sqlite file and reloads the saved lift,
// including names, signatures, address notes, and image strings. [Remember]
// returns the stored artifact when the path and SHA-256 match.
// [New] keeps the same database in memory. [Artifact.SetName],
// [Artifact.SetNote], [Artifact.SetSignature], and [Artifact.SetSource]
// write those clues back when the project has a file. [Artifact.Check]
// compares a candidate the caller already wrote and stores it only when
// every block copies the lifted control flow and the source names the
// function, each direct callee, and each referenced string.
// [Artifact.Slug] is the image hash used in the preview. [Artifact.Stream]
// prints the instructions in address order with function comments and address notes.
package decomp
