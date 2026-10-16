// Package ninja is one ninja build file.
//
// [New] returns an empty [File]. Methods return the child they store:
// [File.Variable], [File.Rule], [File.Build], [File.Pool], [File.Default],
// [File.Include], and [File.Subninja]. A build edge is outputs, a rule,
// inputs, and bindings. [Text] and [Ref] build one piece of an unexpanded
// value. [Take] parses the file. [Make] prints it. [Take], then [Make],
// then [Take] produce the same file.
//
// Values keep ninja's lexical shape: $$ is a dollar, an escaped newline
// joins without inserting a space, and $name stays a reference. Include
// and subninja paths are kept, not loaded. Lowering edges to a workflow
// graph stays in x/workflow/ninja.
//
// The contract is in DESIGN.md beside the parent directory.
package ninja
