// Package ninja is one ninja build file.
//
// [New] returns an empty [File]. Methods return the receiver and store
// the child: variable, rule, and build edge. [Take] parses the file.
// [Make] prints it. Lowering edges to a workflow graph stays in
// x/workflow/ninja. The contract is in DESIGN.md beside the parent
// directory. This scaffold returns
// [github.com/lewtec/lewkit/x/errs.ErrScaffold].
package ninja
