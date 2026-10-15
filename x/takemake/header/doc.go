// Package header is one C header.
//
// [New] returns an empty [File]. Methods return the receiver and store
// the child: type, const, func, and object. Size, alignment, and field
// offset live on the type. [Take] parses a header. [Make] prints C.
// The contract is in DESIGN.md beside the parent directory. This
// scaffold returns [github.com/lewtec/lewkit/x/errs.ErrScaffold].
package header
