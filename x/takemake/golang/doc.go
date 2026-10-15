// Package golang is one Go source file.
//
// [NewFile] returns an empty [File] for that package name. Methods
// return the receiver and store the child: Func, Type, Var, Const, Id,
// Qual, Lit, Params, Block, and Call. [Take] parses source into that
// file. [Make] prints it. The contract is in DESIGN.md beside the parent
// directory. This scaffold returns
// [github.com/lewtec/lewkit/x/errs.ErrScaffold].
package golang
