// Package zip is one zip archive.
//
// [New] returns an empty [File]. Methods return the receiver and store
// an entry: name, mode, time, and bytes or a symlink target. [Take]
// reads an archive. [Make] writes it. Entries stay in archive order.
// The contract is in DESIGN.md beside the parent directory.
package zip
