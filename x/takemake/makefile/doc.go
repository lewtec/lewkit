// Package makefile is one Makefile.
//
// [New] returns an empty [File]. Methods return the child they store:
// [File.Variable], [File.Rule], and [File.Function], plus include,
// export, vpath, undefine, and conditional directives. A rule is
// targets, prerequisites, and a recipe. A function is a define/endef
// block. Flag methods are [Variable.Exported], [Variable.Overridden],
// and [Variable.Privately]. [Take] parses the Makefile. [Make] prints
// it. [Take], then [Make], then [Take] produce the same file.
//
// Values keep make's read-time shape: a backslash-newline outside a
// recipe joins with one space, and a comment is removed after that
// join. Conditionals are stored, not evaluated. Lowering the file to a
// workflow graph stays in x/workflow/make.
//
// The contract is in DESIGN.md beside the parent directory.
package makefile
