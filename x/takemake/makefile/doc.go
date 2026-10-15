// Package makefile is one Makefile.
//
// [New] returns an empty [File]. Methods return the receiver and store
// the child: variable, rule, and function. [Take] parses the Makefile.
// [Make] prints it. Lowering that file to a workflow graph stays in
// x/workflow/make. The contract is in DESIGN.md beside the parent
// directory. This scaffold returns
// [github.com/lewtec/lewkit/x/errs.ErrScaffold].
package makefile
