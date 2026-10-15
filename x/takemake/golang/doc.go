// Package golang is one Go source file.
//
// [NewFile] returns an empty [File] for that package name. Methods return
// the receiver and store the child: [File.Func], [File.Type], [File.Var],
// [File.Const], [Func.Id], [Func.Params], and [Func.Block]. [Id], [Qual],
// and [Lit] build expressions. [Expr.Call] builds a call. [Take] parses
// source into that file. [Make] prints it. A composed file and a taken
// file are the same type. The contract is in DESIGN.md beside the parent
// directory.
package golang
