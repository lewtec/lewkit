// Package view serves a browser preview of a sqlite program database.
//
// The lewkit command is decomp view. Cmd+K and Ctrl+K jump to an artifact or a function.
// A note keyword belongs to the function that contains that address.
// The page shows the summary, the instruction stream, and the comment.
// The function page shows highlighted assembly: each instruction is an anchor.
// An address in the image jumps to that instruction, to the function that contains it, or to the address page.
// A displacement outside the image stays plain.
// The stream page uses the same highlighting.
// Each artifact URL is the image hash slug.
// A call links to that function, and the artifact table shows address and size.
// A saved function name, signature, held source, or address note is written into the sqlite project.
// Check stores a candidate when the blocks agree.
// The file is opened with [github.com/lewtec/lewkit/x/disasm/decomp.Open].
package view
