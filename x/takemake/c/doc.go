// Package c is one C translation unit.
//
// [New] returns an empty [File]. Methods return the receiver and store
// the child: type, const, func, and object. A function keeps its body.
// Size, alignment, and field offset live on the type. [Take] parses C.
// [Make] prints C. The contract is in DESIGN.md beside the parent
// directory. This scaffold returns
// [github.com/lewtec/lewkit/x/errs.ErrScaffold].
package c
