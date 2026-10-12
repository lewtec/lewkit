// Package decompview serves a browser preview of a sqlite program database.
//
// The lewkit command is decomp view. Cmd+K and Ctrl+K jump to an artifact or a function.
// A note keyword belongs to the function that contains that address.
// The page shows the summary, the listing, the comment, and matched source.
// A call links to that function, and the artifact table shows address and size.
// A saved function name, signature, or address note is written into the sqlite project.
// The file is opened with [github.com/lewtec/lewkit/x/decomp.Open].
package decompview
