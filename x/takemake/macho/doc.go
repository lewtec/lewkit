// Package macho is one Mach-O image.
//
// [New] returns an empty [File]. Methods return the receiver and store
// the child: CPU, segment, section, symbol, dylib, and entry. [Take]
// reads a Mach-O into that file. [Make] writes it. The contract is in
// DESIGN.md beside the parent directory. This scaffold returns
// [github.com/lewtec/lewkit/x/errs.ErrScaffold].
package macho
