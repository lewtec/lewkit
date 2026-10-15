// Package wim is one WIM image.
//
// [New] returns an empty [File]. Methods return the receiver and store
// an entry: name, mode, time, and bytes. [Take] reads the image.
// [Make] writes it. The contract is in DESIGN.md beside the parent
// directory. This scaffold returns
// [github.com/lewtec/lewkit/x/errs.ErrScaffold].
package wim
