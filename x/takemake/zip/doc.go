// Package zip is one zip archive.
//
// [New] returns an empty [File]. Methods return the receiver and store
// an entry: name, mode, time, and bytes or a symlink target. [Take]
// reads an archive. [Make] writes it. The contract is in DESIGN.md
// beside the parent directory. This scaffold returns
// [github.com/lewtec/lewkit/x/errs.ErrScaffold].
package zip
