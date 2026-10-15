// Package proto is one proto file.
//
// [NewFile] returns an empty [File]. Methods return the receiver and
// store the child: syntax, package, message, and field. [Take] parses
// proto text. [Make] prints it. The contract is in DESIGN.md beside the
// parent directory. This scaffold returns
// [github.com/lewtec/lewkit/x/errs.ErrUnsupported].
package proto
